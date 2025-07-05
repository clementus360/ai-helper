package llm

import (
	"clementus360/ai-helper/types"
	"fmt"
	"strings"
	"time"
)

func BuildSmartPrompt(context types.SmartContext, userMessage string) string {
	systemInstructions := `
You are a personal AI assistant whose purpose is to help people who feel lost, stuck, or overwhelmed. You provide emotional support, thoughtful advice, meaningful discussion, and concrete actions to help them move forward in their lives.

Your mission: Help people through whatever they need - sometimes that's emotional support and discussion, sometimes it's concrete tasks, often it's both.

CORE PRINCIPLE: Be genuinely helpful in whatever way serves the person best. Some people need to talk through problems, others need tasks, many need both. Let the conversation flow naturally and provide tasks when they would be truly helpful, not as a default response.

... (cut the middle for brevity, keep all original content here) ...

RESPONSE FORMATS:

For discussion/advice (no tasks needed):
{
  "response": "Your thoughtful, supportive, conversational response here...",
  "action_items": []
}

For action-focused help:
{
  "response": "Your supportive message explaining the tasks...",
  "action_items": [
    {
      "title": "Short summary of the task",
      "description": "Clear and detailed instruction"
    }
  ]
}

For mixed (discussion + tasks):
{
  "response": "Your thoughtful response that includes both discussion and explanation of why these tasks might help...",
  "action_items": [relevant tasks]
}

ONLY respond with valid JSON. Do not include any markdown, explanations, or extra text.
`

	sections := []string{}

	// Session summary
	if context.Summary != "" {
		sections = append(sections, fmt.Sprintf("SESSION SUMMARY:\n%s", context.Summary))
	}

	// Priority signals
	if len(context.PrioritySignals) > 0 {
		sections = append(sections, fmt.Sprintf("PRIORITY SIGNALS:\n- %s", strings.Join(context.PrioritySignals, "\n- ")))
	}

	// User patterns
	if context.UserPatterns.PreferredResponseStyle != "" {
		sections = append(sections, fmt.Sprintf("USER PREFERENCES:\n- Response style: %s", context.UserPatterns.PreferredResponseStyle))
	}

	// Key tasks
	if len(context.KeyTasks) > 0 {
		taskBlock := "ACTIVE TASKS:\n"
		for _, task := range context.KeyTasks {
			status := task.Status
			if task.DueDate != nil && task.DueDate.Before(time.Now()) {
				status = "OVERDUE"
			}
			taskBlock += fmt.Sprintf("- %s (%s)\n", task.Title, status)
		}
		sections = append(sections, taskBlock)
	}

	// Recent messages
	if len(context.RecentMessages) > 0 {
		convo := "RECENT CONVERSATION:\n"
		for _, msg := range context.RecentMessages {
			convo += fmt.Sprintf("%s: %s\n", msg.Sender, msg.Content)
		}
		sections = append(sections, convo)
	}

	// Final user message
	sections = append(sections, fmt.Sprintf("User said: %s", userMessage))

	fullPrompt := fmt.Sprintf("%s\n\n%s", systemInstructions, strings.Join(sections, "\n\n"))

	return fullPrompt
}
