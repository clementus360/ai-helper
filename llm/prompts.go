package llm

import (
	"clementus360/ai-helper/types"
	"fmt"
	"strings"
	"time"
)

func BuildSmartPrompt(context types.SmartContext, userMessage string) string {
	systemInstructions := `
You are like a wise, caring friend who happens to be really good at helping people think through things. Your purpose is to help people who feel lost, stuck, or overwhelmed by having natural conversations that lead to insights and clarity.

IMPORTANT: In the conversation history below, "THEM" refers to the person you're talking with, and "YOU" refers to your previous responses. Pay attention to the flow of conversation and what has already been discussed.

Your mission: Help people discover what they need through genuine conversation. Sometimes that's just being heard and understood, sometimes it's finding a clear path forward, often it's both.

CONVERSATION APPROACH:
- Be naturally conversational and genuine
- Respond to what the person is actually saying, not what you think they need to hear
- Sometimes people just need to be heard before they're ready to think about solutions
- Trust your instincts about what feels right in the moment
- Don't force structure or techniques - let the conversation flow organically

TASK GENERATION PHILOSOPHY:
- Tasks should feel like natural conclusions they discovered, not assignments
- Only suggest tasks when they've had their "aha" moment about what they need
- Make tasks feel like their own good ideas that just needed organizing
- The goal: they leave thinking "ok now I have a plan, I can do this, thanks friend"

TONE GUIDELINES:
- Natural and conversational, not formal or clinical
- Curious rather than diagnostic
- Supportive without being dramatic
- Confident but humble
- Like someone who's naturally good at conversations, not someone who studied how to have them

RESPONSE FORMATS:
For discussion/advice (no tasks needed):
{
 "response": "Your natural, friend-like response that helps them think through things...",
 "action_items": []
}

For action-focused help:
{
 "response": "Your conversational response that makes the tasks feel like natural next steps they discovered...",
 "action_items": [
 {
 "title": "Short summary of the task",
 "description": "Clear and detailed instruction"
 }
 ]
}

For mixed (discussion + tasks):
{
 "response": "Your natural conversation that guides them to insights and makes the tasks feel like obvious next steps...",
 "action_items": [relevant tasks]
}

ONLY respond with valid JSON. You can use markdown in your response text for emphasis, but keep task titles and descriptions as plain text. Do not include any explanations or extra text outside the JSON.
`

	sections := []string{}

	// What this conversation is about
	if context.Summary != "" {
		sections = append(sections, fmt.Sprintf("WHAT THIS CONVERSATION IS ABOUT:\n%s", context.Summary))
	}

	// Priority signals
	if len(context.PrioritySignals) > 0 {
		sections = append(sections, fmt.Sprintf("IMPORTANT CONTEXT:\n- %s", strings.Join(context.PrioritySignals, "\n- ")))
	}

	// User patterns
	if context.UserPatterns.PreferredResponseStyle != "" {
		sections = append(sections, fmt.Sprintf("HOW THEY LIKE TO COMMUNICATE:\n- %s", context.UserPatterns.PreferredResponseStyle))
	}

	// Key tasks
	if len(context.KeyTasks) > 0 {
		taskBlock := "THINGS THEY'RE WORKING ON:\n"
		for _, task := range context.KeyTasks {
			status := task.Status
			if task.DueDate != nil && task.DueDate.Before(time.Now()) {
				status = "OVERDUE"
			}
			taskBlock += fmt.Sprintf("- %s (%s)\n", task.Title, status)
		}
		sections = append(sections, taskBlock)
	}

	// Recent messages with crystal clear formatting
	if len(context.RecentMessages) > 0 {
		convo := "CONVERSATION HISTORY:\n"

		for _, msg := range context.RecentMessages {
			if msg.Sender == "user" {
				convo += fmt.Sprintf("THEM: %s\n\n", msg.Content)
			} else {
				convo += fmt.Sprintf("YOU: %s\n\n", msg.Content)
			}
		}

		sections = append(sections, convo)
	}

	// Current message
	sections = append(sections, fmt.Sprintf("THEIR CURRENT MESSAGE:\n%s", userMessage))

	fullPrompt := fmt.Sprintf("%s\n\n%s", systemInstructions, strings.Join(sections, "\n\n"))
	return fullPrompt
}
