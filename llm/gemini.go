package llm

import (
	"bytes"
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
	DueDate       *time.Time `json:"due_date,omitempty"`         // MISSING FIELD - this is why due dates aren't working
	Decision      string     `json:"decision,omitempty"`         // MISSING FIELD
	FollowUpDueAt *time.Time `json:"follow_up_due_at,omitempty"` // MISSING FIELD
	FollowedUp    *bool      `json:"followed_up,omitempty"`      // MISSING FIELD
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
			"temperature":     0.3, // Lower temperature for more consistent formatting
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

	// Enhanced response parsing with better error handling
	text, err := extractTextFromResponse(res)
	if err != nil {
		return GeminiStructuredResponse{}, err
	}

	// Clean and parse the LLM's JSON output with fallback handling
	structured, err := parseStructuredResponse(text)
	if err != nil {
		// Fallback: create a basic response if JSON parsing fails
		return GeminiStructuredResponse{
			Response:    fmt.Sprintf("I understand you're dealing with: %s. Let me ask a few questions to better help you. What specifically feels most challenging about this situation right now?", userInput),
			ActionItems: []GeminiTaskItem{},
		}, nil
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

// Enhanced JSON parsing with cleanup and validation
func parseStructuredResponse(text string) (GeminiStructuredResponse, error) {
	cleanedText := cleanJSONResponse(text)

	var structured GeminiStructuredResponse
	if err := json.Unmarshal([]byte(cleanedText), &structured); err != nil {
		return GeminiStructuredResponse{}, fmt.Errorf("failed to parse JSON response: %v\nOriginal text: %s", err, text)
	}

	if err := validateResponse(structured); err != nil {
		return GeminiStructuredResponse{}, err
	}

	return structured, nil
}

// Clean JSON response by removing common formatting issues
func cleanJSONResponse(text string) string {
	// Remove code block markers if present
	text = regexp.MustCompile("```json\n?").ReplaceAllString(text, "")
	text = regexp.MustCompile("```\n?").ReplaceAllString(text, "")

	// Remove any leading/trailing whitespace
	text = strings.TrimSpace(text)

	// Find JSON object boundaries
	startIdx := strings.Index(text, "{")
	if startIdx == -1 {
		return text
	}

	// Find the last closing brace
	endIdx := strings.LastIndex(text, "}")
	if endIdx == -1 || endIdx <= startIdx {
		return text
	}

	return text[startIdx : endIdx+1]
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

	// Parse JSON response
	var structured struct {
		Summary string `json:"summary"`
		Title   string `json:"title"`
	}
	cleanedText := cleanJSONResponse(text)
	if err := json.Unmarshal([]byte(cleanedText), &structured); err != nil {
		return "", "", fmt.Errorf("failed to parse JSON response: %v\nOriginal text: %s", err, text)
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
