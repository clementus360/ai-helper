package llm

import (
	"bytes"
	"clementus360/ai-helper/config"
	"clementus360/ai-helper/types"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"
)

const openaiURL = "https://api.openai.com/v1/chat/completions"

func OpenAIGenerateResponse(userInput string, context types.SmartContext) (GeminiStructuredResponse, error) {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		return GeminiStructuredResponse{}, fmt.Errorf("OPENAI_API_KEY not set")
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

	// OpenAI request body
	body := map[string]interface{}{
		"model": "gpt-3.5-turbo",
		"messages": []map[string]interface{}{
			{
				"role":    "user",
				"content": prompt,
			},
		},
		"temperature": 0.3,
		"max_tokens":  1000,
		"top_p":       0.8,
	}

	jsonData, err := json.Marshal(body)
	if err != nil {
		return GeminiStructuredResponse{}, fmt.Errorf("failed to marshal request: %v", err)
	}

	// Create the HTTP request
	req, err := http.NewRequest("POST", openaiURL, bytes.NewReader(jsonData))
	if err != nil {
		return GeminiStructuredResponse{}, fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

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

	// Extract text from OpenAI API response
	text, err := extractTextFromOpenAIResponse(res)
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

// Extract text from OpenAI API response
func extractTextFromOpenAIResponse(res map[string]interface{}) (string, error) {
	choices, ok := res["choices"].([]interface{})
	if !ok || len(choices) == 0 {
		return "", fmt.Errorf("no choices returned from OpenAI")
	}

	choice, ok := choices[0].(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("invalid choice format")
	}

	message, ok := choice["message"].(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("no message in choice")
	}

	content, ok := message["content"].(string)
	if !ok {
		return "", fmt.Errorf("no content in message")
	}

	return content, nil
}

// OpenAI version of session summary and title generation
func OpenAIGenerateSessionSummaryAndTitle(messages []types.Message, context types.SmartContext) (string, string, error) {
	apiKey := os.Getenv("OPENAI_API_KEY_SUMMARY_TITLE")
	if apiKey == "" {
		return "", "", fmt.Errorf("OPENAI_API_KEY_SUMMARY_TITLE not set")
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
		"model": "gpt-3.5-turbo",
		"messages": []map[string]interface{}{
			{
				"role":    "user",
				"content": prompt,
			},
		},
		"temperature": 0.3,
		"max_tokens":  300,
		"top_p":       0.8,
	}

	jsonData, err := json.Marshal(body)
	if err != nil {
		return "", "", fmt.Errorf("failed to marshal request: %v", err)
	}

	req, err := http.NewRequest("POST", openaiURL, bytes.NewReader(jsonData))
	if err != nil {
		return "", "", fmt.Errorf("failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

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

	text, err := extractTextFromOpenAIResponse(result)
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
func OpenAIGenerateResponseCompat(userMessage string, context types.SessionContext) (GeminiStructuredResponse, error) {
	smartContext := types.SmartContext{
		Summary:        context.Summary,
		RecentMessages: context.RecentMessages,
	}

	return OpenAIGenerateResponse(userMessage, smartContext)
}
