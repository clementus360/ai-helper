package llm

import (
	"bytes"
	"clementus360/ai-helper/config"
	"clementus360/ai-helper/types"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"
)

const apiURL = "https://generativelanguage.googleapis.com/v1beta/models/gemini-2.0-flash:generateContent"

type GeminiStructuredResponse struct {
	Response    string             `json:"response"`
	ActionItems []GeminiTaskItem   `json:"action_items"`
	DeleteTasks []string           `json:"delete_tasks,omitempty"`
	UpdateTasks []GeminiTaskUpdate `json:"update_tasks,omitempty"`
}

type GeminiTaskUpdate struct {
	ID            string     `json:"id"`
	Title         string     `json:"title,omitempty"`
	Description   string     `json:"description,omitempty"`
	Status        string     `json:"status,omitempty"`
	DueDate       *time.Time `json:"due_date,omitempty"`
	Decision      string     `json:"decision,omitempty"`
	FollowUpDueAt *time.Time `json:"follow_up_due_at,omitempty"`
	FollowedUp    *bool      `json:"followed_up,omitempty"`
}

type GeminiTaskItem struct {
	Title       string `json:"title"`
	Description string `json:"description"`
}

func GeminiGenerateResponse(userInput string, context types.SmartContext) (GeminiStructuredResponse, error) {
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		return GeminiStructuredResponse{}, fmt.Errorf("GEMINI_API_KEY not set")
	}

	// Add input validation
	if strings.TrimSpace(userInput) == "" {
		return GeminiStructuredResponse{
			Response:    "I'd love to help! Could you tell me what's on your mind or what you're struggling with?",
			ActionItems: []GeminiTaskItem{},
		}, nil
	}

	// Trim context to fit token limits
	trimmedContext := TrimContextForTokens(context, 6000)

	// Build enhanced prompt
	prompt := BuildSmartPrompt(trimmedContext, userInput)

	// Enhanced request body with generation config for more consistent JSON
	body := map[string]interface{}{
		"contents": []map[string]interface{}{
			{
				"parts": []map[string]string{
					{"text": prompt},
				},
			},
		},
		"generationConfig": map[string]interface{}{
			"temperature":     0.3,
			"maxOutputTokens": 1000,
			"topP":            0.8,
		},
	}

	jsonData, err := json.Marshal(body)
	if err != nil {
		return GeminiStructuredResponse{}, fmt.Errorf("failed to marshal request: %v", err)
	}

	// Create the HTTP request
	req, err := http.NewRequest("POST", apiURL+"?key="+apiKey, bytes.NewReader(jsonData))
	if err != nil {
		return GeminiStructuredResponse{}, fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")

	// Add timeout to prevent hanging
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return GeminiStructuredResponse{}, fmt.Errorf("request failed: %v", err)
	}
	defer resp.Body.Close()

	// Check HTTP status
	if resp.StatusCode != http.StatusOK {
		return GeminiStructuredResponse{}, fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	var res map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return GeminiStructuredResponse{}, fmt.Errorf("failed to decode response: %v", err)
	}

	// Extract text from Gemini API response
	text, err := extractTextFromResponse(res)
	if err != nil {
		config.Logger.Printf("Failed to extract text from response: %v", err)
		return createFallbackResponse(userInput, ""), nil
	}

	fmt.Println("Extracted text:", text) // Debugging output

	// Try to parse JSON for structured response
	structured, err := parseStructuredResponseRobust(text)
	if err != nil {
		config.Logger.Printf("Failed to parse structured response: %v\nOriginal text: %s\n", err, text)
		return createFallbackResponse(userInput, text), nil
	}

	// Validate the structured response
	if err := validateResponse(structured); err != nil {
		config.Logger.Printf("Response validation failed: %v\nResponse: %s\n", err, structured.Response)
		return createFallbackResponse(userInput, text), nil
	}

	// Replace task IDs with titles in the response
	for _, task := range context.KeyTasks {
		structured.Response = strings.ReplaceAll(structured.Response, task.ID, task.Title)
	}

	return structured, nil
}

// Extract text from Gemini API response with proper error handling
func extractTextFromResponse(res map[string]interface{}) (string, error) {
	candidates, ok := res["candidates"].([]interface{})
	if !ok || len(candidates) == 0 {
		return "", fmt.Errorf("no candidates returned from Gemini")
	}

	candidate, ok := candidates[0].(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("invalid candidate format")
	}

	content, ok := candidate["content"].(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("no content in candidate")
	}

	parts, ok := content["parts"].([]interface{})
	if !ok || len(parts) == 0 {
		return "", fmt.Errorf("no parts in content")
	}

	part, ok := parts[0].(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("invalid part format")
	}

	text, ok := part["text"].(string)
	if !ok {
		return "", fmt.Errorf("no text in part")
	}

	return text, nil
}

// Robust JSON parsing that tries multiple extraction methods
func parseStructuredResponseRobust(text string) (GeminiStructuredResponse, error) {
	// Try multiple JSON extraction strategies
	strategies := []func(string) (string, bool){
		extractCompleteJSON,
		extractJSONFromCodeBlock,
		extractJSONFromBraces,
		extractPartialJSON,
		extractJSONWithRepair,
	}

	for _, strategy := range strategies {
		if jsonStr, found := strategy(text); found {
			var structured GeminiStructuredResponse
			if err := json.Unmarshal([]byte(jsonStr), &structured); err == nil {
				if err := validateResponse(structured); err == nil {
					config.Logger.Printf("Successfully parsed JSON using strategy")
					return structured, nil
				} else {
					config.Logger.Printf("JSON parsed but validation failed: %v", err)
				}
			} else {
				config.Logger.Printf("JSON unmarshaling failed: %v", err)
			}
		}
	}

	return GeminiStructuredResponse{}, fmt.Errorf("no valid JSON found in response")
}

// New strategy: Attempt to repair malformed JSON
func extractJSONWithRepair(text string) (string, bool) {
	// Try to fix common JSON issues
	fixedText := fixCommonJSONIssues(text)

	// Additional repairs
	fixedText = repairMalformedJSON(fixedText)

	// Try parsing as JSON
	if json.Valid([]byte(fixedText)) {
		return fixedText, true
	}

	// Try extracting the largest valid JSON fragment
	startIdx := strings.Index(fixedText, "{")
	if startIdx == -1 {
		return "", false
	}

	braceCount := 0
	inString := false
	escaped := false
	endIdx := len(fixedText)

	for i := startIdx; i < len(fixedText); i++ {
		char := fixedText[i]

		if escaped {
			escaped = false
			continue
		}

		if char == '\\' {
			escaped = true
			continue
		}

		if char == '"' {
			inString = !inString
			continue
		}

		if !inString {
			if char == '{' {
				braceCount++
			} else if char == '}' {
				braceCount--
				if braceCount == 0 {
					candidate := fixedText[startIdx : i+1]
					if json.Valid([]byte(candidate)) {
						return candidate, true
					}
					endIdx = i + 1
				}
			}
		}
	}

	// Try the largest possible fragment
	candidate := fixedText[startIdx:endIdx]
	if json.Valid([]byte(candidate)) {
		return candidate, true
	}

	return "", false
}

// Repair common JSON issues
func repairMalformedJSON(text string) string {
	// Remove trailing commas before } or ]
	text = regexp.MustCompile(`,\s*([}\]])`).ReplaceAllString(text, "$1")

	// Add missing closing braces
	openBraces := strings.Count(text, "{") - strings.Count(text, "}")
	if openBraces > 0 {
		text += strings.Repeat("}", openBraces)
	}

	// Add missing closing brackets
	openBrackets := strings.Count(text, "[") - strings.Count(text, "]")
	if openBrackets > 0 {
		text += strings.Repeat("]", openBrackets)
	}

	// Fix unquoted keys
	text = regexp.MustCompile(`([{,]\s*)([a-zA-Z_][a-zA-Z0-9_]*)\s*:`).ReplaceAllString(text, `$1"$2":`)

	return text
}

// Strategy 1: Try the entire text as JSON
func extractCompleteJSON(text string) (string, bool) {
	cleaned := strings.TrimSpace(text)
	if json.Valid([]byte(cleaned)) {
		return cleaned, true
	}
	return "", false
}

// Strategy 2: Extract JSON from markdown code blocks
func extractJSONFromCodeBlock(text string) (string, bool) {
	// Match ```json ... ``` or ``` ... ```
	codeBlockRegex := regexp.MustCompile("(?s)```(?:json)?\\s*(\\{.*?\\})\\s*```")
	matches := codeBlockRegex.FindStringSubmatch(text)
	if len(matches) > 1 {
		candidate := strings.TrimSpace(matches[1])
		if json.Valid([]byte(candidate)) {
			return candidate, true
		}
	}
	return "", false
}

// Strategy 3: Extract JSON object from braces (most common case)
func extractJSONFromBraces(text string) (string, bool) {
	// First try to fix common JSON issues
	fixedText := fixCommonJSONIssues(text)

	// Find the largest valid JSON object in the text
	var bestJSON string
	var bestLength int

	// Look for all potential JSON objects
	braceRegex := regexp.MustCompile(`\{[^{}]*(?:\{[^{}]*\}[^{}]*)*\}`)
	matches := braceRegex.FindAllString(fixedText, -1)

	for _, match := range matches {
		// Try to expand the match to include nested objects
		expanded := expandJSONMatch(fixedText, match)
		if json.Valid([]byte(expanded)) && len(expanded) > bestLength {
			bestJSON = expanded
			bestLength = len(expanded)
		}
	}

	if bestJSON != "" {
		return bestJSON, true
	}

	// If no valid JSON found, try with more aggressive pattern
	return extractJSONWithNestedBraces(fixedText)
}

// Fix common JSON formatting issues
func fixCommonJSONIssues(text string) string {
	// Unescape newlines and quotes properly
	text = strings.ReplaceAll(text, `\n`, "\n")
	text = strings.ReplaceAll(text, `\"`, `"`)

	// Fix common trailing comma issues
	text = regexp.MustCompile(`,\s*}`).ReplaceAllString(text, "}")
	text = regexp.MustCompile(`,\s*]`).ReplaceAllString(text, "]")

	return text
}

// More aggressive JSON extraction with nested brace handling
func extractJSONWithNestedBraces(text string) (string, bool) {
	// Find the start of a JSON object
	startIdx := strings.Index(text, "{")
	if startIdx == -1 {
		return "", false
	}

	// Count braces to find the end
	braceCount := 0
	inString := false
	escaped := false

	for i := startIdx; i < len(text); i++ {
		char := text[i]

		if escaped {
			escaped = false
			continue
		}

		if char == '\\' {
			escaped = true
			continue
		}

		if char == '"' && !escaped {
			inString = !inString
			continue
		}

		if !inString {
			if char == '{' {
				braceCount++
			} else if char == '}' {
				braceCount--
				if braceCount == 0 {
					candidate := text[startIdx : i+1]
					// Try to fix and validate
					fixed := fixCommonJSONIssues(candidate)
					if json.Valid([]byte(fixed)) {
						return fixed, true
					}
				}
			}
		}
	}

	return "", false
}

// Strategy 4: Try to extract and reconstruct partial JSON
func extractPartialJSON(text string) (string, bool) {
	// Look for JSON-like patterns and try to reconstruct
	lines := strings.Split(text, "\n")
	var jsonLines []string
	inJSON := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "{") {
			inJSON = true
		}
		if inJSON {
			jsonLines = append(jsonLines, line)
		}
		if inJSON && strings.HasSuffix(trimmed, "}") && !strings.Contains(trimmed, "{") {
			break
		}
	}

	if len(jsonLines) > 0 {
		candidate := strings.Join(jsonLines, "\n")
		if json.Valid([]byte(candidate)) {
			return candidate, true
		}
	}
	return "", false
}

// Helper function to expand JSON match to include nested structures
func expandJSONMatch(text, match string) string {
	startIdx := strings.Index(text, match)
	if startIdx == -1 {
		return match
	}

	// Try to expand backwards and forwards to capture the complete JSON
	expanded := match
	braceCount := 0
	inString := false
	escaped := false

	// Start from the beginning of the match and expand
	for i := startIdx; i < len(text); i++ {
		char := text[i]

		if escaped {
			escaped = false
			continue
		}

		if char == '\\' {
			escaped = true
			continue
		}

		if char == '"' {
			inString = !inString
			continue
		}

		if !inString {
			if char == '{' {
				braceCount++
			} else if char == '}' {
				braceCount--
				if braceCount == 0 {
					expanded = text[startIdx : i+1]
					break
				}
			}
		}
	}

	return expanded
}

// Create a fallback response when JSON parsing fails
func createFallbackResponse(userInput, rawText string) GeminiStructuredResponse {
	// Try to extract partial task data
	partialData := extractPartialDataFromBrokenJSON(rawText)

	// Use raw text if non-empty
	if strings.TrimSpace(rawText) != "" {
		return GeminiStructuredResponse{
			Response:    strings.TrimSpace(rawText),
			ActionItems: partialData.ActionItems,
			DeleteTasks: partialData.DeleteTasks,
			UpdateTasks: partialData.UpdateTasks,
		}
	}

	// Fallback only if raw text is empty
	responseText := "I'm sorry, I couldn't generate a response. Could you clarify what you're trying to do or what specific help you need? For example, are you asking about a task, code, or something else?"
	if strings.Contains(strings.ToLower(userInput), "code") {
		responseText = "It looks like you're asking about code, but I couldn't generate one. Could you specify what kind of code you're looking for (e.g., language, functionality)? I'll provide a complete example."
	}

	return GeminiStructuredResponse{
		Response:    responseText,
		ActionItems: partialData.ActionItems,
		DeleteTasks: partialData.DeleteTasks,
		UpdateTasks: partialData.UpdateTasks,
	}
}

// Extract partial data from broken JSON
func extractPartialDataFromBrokenJSON(text string) GeminiStructuredResponse {
	result := GeminiStructuredResponse{
		ActionItems: []GeminiTaskItem{},
		DeleteTasks: []string{},
		UpdateTasks: []GeminiTaskUpdate{},
	}

	// Try to extract JSON fragment
	jsonStr, found := extractJSONWithRepair(text)
	if !found {
		jsonStr, found = extractJSONFromBraces(text)
	}
	if !found {
		jsonStr, found = extractJSONFromCodeBlock(text)
	}
	if !found {
		jsonStr, found = extractPartialJSON(text)
	}

	if found {
		var partial GeminiStructuredResponse
		if err := json.Unmarshal([]byte(jsonStr), &partial); err == nil {
			return partial
		}
	}

	// Fallback to regex-based extraction for action items
	actionItemsRegex := regexp.MustCompile(`"action_items"\s*:\s*\[([^\]]+)\]`)
	if matches := actionItemsRegex.FindStringSubmatch(text); len(matches) > 1 {
		itemRegex := regexp.MustCompile(`\{\s*"title"\s*:\s*"([^"]+)"\s*,\s*"description"\s*:\s*"([^"]+)"\s*\}`)
		itemMatches := itemRegex.FindAllStringSubmatch(matches[1], -1)
		for _, item := range itemMatches {
			if len(item) > 2 {
				result.ActionItems = append(result.ActionItems, GeminiTaskItem{
					Title:       item[1],
					Description: item[2],
				})
			}
		}
	}

	// Extract delete tasks
	deleteTasksRegex := regexp.MustCompile(`"delete_tasks"\s*:\s*\[([^\]]+)\]`)
	if matches := deleteTasksRegex.FindStringSubmatch(text); len(matches) > 1 {
		stringRegex := regexp.MustCompile(`"([^"]+)"`)
		stringMatches := stringRegex.FindAllStringSubmatch(matches[1], -1)
		for _, str := range stringMatches {
			if len(str) > 1 {
				result.DeleteTasks = append(result.DeleteTasks, str[1])
			}
		}
	}

	// Extract update tasks (simplified)
	updateTasksRegex := regexp.MustCompile(`"update_tasks"\s*:\s*\[([^\]]+)\]`)
	if matches := updateTasksRegex.FindStringSubmatch(text); len(matches) > 1 {
		itemRegex := regexp.MustCompile(`\{\s*"id"\s*:\s*"([^"]+)"(?:\s*,\s*"title"\s*:\s*"([^"]*)")?(?:\s*,\s*"description"\s*:\s*"([^"]*)")?(?:\s*,\s*"status"\s*:\s*"([^"]*)")?\s*\}`)
		itemMatches := itemRegex.FindAllStringSubmatch(matches[1], -1)
		for _, item := range itemMatches {
			if len(item) > 1 {
				update := GeminiTaskUpdate{ID: item[1]}
				if len(item) > 2 && item[2] != "" {
					update.Title = item[2]
				}
				if len(item) > 3 && item[3] != "" {
					update.Description = item[3]
				}
				if len(item) > 4 && item[4] != "" {
					update.Status = item[4]
				}
				result.UpdateTasks = append(result.UpdateTasks, update)
			}
		}
	}

	return result
}

// Validate the structured response
func validateResponse(response GeminiStructuredResponse) error {
	if response.Response == "" {
		return fmt.Errorf("response field is empty")
	}

	// ActionItems can be empty, but if present, should not contain empty strings
	for i, item := range response.ActionItems {
		if strings.TrimSpace(item.Title) == "" {
			return fmt.Errorf("action item %d has empty title", i)
		}
		if strings.TrimSpace(item.Description) == "" {
			return fmt.Errorf("action item %d has empty description", i)
		}
	}

	// Validate update tasks
	for i, update := range response.UpdateTasks {
		if strings.TrimSpace(update.ID) == "" {
			return fmt.Errorf("update task %d has empty ID", i)
		}
	}

	// Validate delete tasks
	for i, deleteID := range response.DeleteTasks {
		if strings.TrimSpace(deleteID) == "" {
			return fmt.Errorf("delete task %d has empty ID", i)
		}
	}

	return nil
}

// GenerateSessionSummaryAndTitle generates both a summary and title in one API call
func GenerateSessionSummaryAndTitle(messages []types.Message, context types.SmartContext) (string, string, error) {
	apiKey := os.Getenv("GEMINI_API_KEY_SUMMARY_TITLE")
	if apiKey == "" {
		return "", "", fmt.Errorf("GEMINI_API_KEY_SUMMARY_TITLE not set")
	}

	// Build message log
	var chatLog strings.Builder
	for _, msg := range messages {
		if msg.Sender == "user" {
			chatLog.WriteString("User: ")
		} else {
			chatLog.WriteString("AI: ")
		}
		chatLog.WriteString(msg.Content)
		chatLog.WriteString("\n")
	}

	// Prompt for both summary and title
	prompt := fmt.Sprintf(`You are a helpful assistant. Based on the following conversation and context:

Conversation:
%s
Context: %v

Provide a JSON response with:
- summary: A clear and concise paragraph summarizing the conversation
- title: A short session title (<8 words)

Respond in valid JSON format only. Example:
{
  "summary": "The user discussed communication challenges and received tasks to improve.",
  "title": "Improving Communication Skills"
}`, chatLog.String(), context)

	body := map[string]interface{}{
		"contents": []map[string]interface{}{
			{
				"parts": []map[string]string{
					{"text": prompt},
				},
			},
		},
		"generationConfig": map[string]interface{}{
			"temperature":     0.3,
			"maxOutputTokens": 300,
			"topP":            0.8,
		},
	}

	jsonData, err := json.Marshal(body)
	if err != nil {
		return "", "", fmt.Errorf("failed to marshal request: %v", err)
	}

	req, err := http.NewRequest("POST", apiURL+"?key="+apiKey, bytes.NewReader(jsonData))
	if err != nil {
		return "", "", fmt.Errorf("failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", "", fmt.Errorf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", "", fmt.Errorf("failed to decode response: %v", err)
	}

	text, err := extractTextFromResponse(result)
	if err != nil {
		return "", "", err
	}

	// Use the robust JSON extraction for summary/title as well
	jsonStr, found := extractJSONFromBraces(text)
	if !found {
		jsonStr, found = extractJSONFromCodeBlock(text)
	}
	if !found {
		jsonStr, found = extractCompleteJSON(text)
	}

	if !found {
		return "", "", fmt.Errorf("no valid JSON found in summary response: %s", text)
	}

	// Parse JSON response
	var structured struct {
		Summary string `json:"summary"`
		Title   string `json:"title"`
	}

	if err := json.Unmarshal([]byte(jsonStr), &structured); err != nil {
		return "", "", fmt.Errorf("failed to parse JSON response: %v\nJSON: %s", err, jsonStr)
	}

	if structured.Summary == "" || structured.Title == "" {
		return "", "", fmt.Errorf("empty summary or title in response")
	}

	return strings.TrimSpace(structured.Summary), strings.TrimSpace(structured.Title), nil
}

// Backward compatibility wrapper
func GeminiGenerateResponseCompat(userMessage string, context types.SessionContext) (GeminiStructuredResponse, error) {
	smartContext := types.SmartContext{
		Summary:        context.Summary,
		RecentMessages: context.RecentMessages,
	}

	return GeminiGenerateResponse(userMessage, smartContext)
}

// Deprecated functions kept for compatibility
func cleanJSONResponse(text string) string {
	jsonBlock, found := extractJSONFromBraces(text)
	if found {
		return jsonBlock
	}
	return strings.TrimSpace(text)
}

func attemptExtractJSONBlock(text string) (string, bool) {
	return extractJSONFromBraces(text)
}
