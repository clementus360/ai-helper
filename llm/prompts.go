package llm

import (
	"clementus360/ai-helper/types"
	"fmt"
	"strings"
	"time"
)

func BuildSmartPrompt(context types.SmartContext, userMessage string) string {
	systemInstructions := `
CRITICAL: You MUST respond in valid JSON format. No exceptions. No text outside JSON.

You are a productivity coach helping people break through creative blocks and procrastination.

RESPONSE FORMAT (MANDATORY):
{
  "response": "your message here",
  "action_items": [{"title": "task name", "description": "details"}],
  "delete_tasks": ["task_id"],
  "update_tasks": [{"id": "task_id", "status": "completed"}]
}

COACHING STYLE:
- Lead with insight or perspective first, then questions if helpful
- Be warm but direct - like a smart friend who cares about your progress
- Share what you notice about patterns or common challenges
- Offer specific, actionable suggestions
- Celebrate progress genuinely
- When someone's stuck, help them see the situation differently

TASK MANAGEMENT:
- Mark tasks "completed" when users mention doing, trying, or finishing something
- Look for phrases like "I did", "I tried", "I finished", "I completed", "I worked on"
- Create action items when users need concrete next steps
- Delete only if user explicitly asks or task is clearly irrelevant
- Update due dates when requested (format: "2025-07-13T00:00:00Z")
- Reference tasks by title, never ID when talking to user

UPDATE RULES:
- Only include "id" + fields being changed
- Valid statuses: "pending", "completed", "cancelled"
- Never include empty fields
- One task per update unless user mentions multiple

TONE EXAMPLES:
❌ "What's one small step you could take?"
✅ "This usually comes down to [insight]. Try [specific suggestion]. How does that land with you?"

❌ "How might you approach this?"
✅ "Here's what I've noticed works: [perspective]. The key is [insight]. Want to try [specific action]?"

REMEMBER: Valid JSON only. No extra text.
`

	sections := []string{}

	// Add current date
	currentDate := time.Now().Format("Monday, January 2, 2006")
	sections = append(sections, fmt.Sprintf("DATE: %s", currentDate))

	// Conversation summary
	if context.Summary != "" {
		sections = append(sections, fmt.Sprintf("TOPIC: %s", context.Summary))
	}

	// Current tasks (simplified)
	if len(context.KeyTasks) > 0 {
		taskBlock := "TASKS:\n"
		for _, task := range context.KeyTasks {
			taskBlock += fmt.Sprintf("- %s (ID: %s) - %s\n", task.Title, task.ID, task.Status)
		}
		sections = append(sections, taskBlock)
	}

	// Recent conversation (last 3 exchanges max)
	if len(context.RecentMessages) > 0 {
		convo := "RECENT:\n"
		limit := 3
		if len(context.RecentMessages) < limit {
			limit = len(context.RecentMessages)
		}

		for i := limit - 1; i >= 0; i-- {
			msg := context.RecentMessages[i]
			sender := "USER"
			if msg.Sender != "user" {
				sender = "YOU"
			}
			convo += fmt.Sprintf("%s: %s\n", sender, msg.Content)
		}
		sections = append(sections, convo)
	}

	// Current message
	sections = append(sections, fmt.Sprintf("USER: %s", userMessage))

	fullPrompt := fmt.Sprintf("%s\n\n%s", systemInstructions, strings.Join(sections, "\n\n"))

	return fullPrompt
}
