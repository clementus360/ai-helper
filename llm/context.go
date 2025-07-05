package llm

import (
	"clementus360/ai-helper/types"
)

// Token estimation and context trimming
func EstimateTokens(text string) int {
	// Rough estimation: ~4 characters per token
	return len(text) / 4
}

func TrimContextForTokens(context types.SmartContext, maxTokens int) types.SmartContext {
	trimmedContext := context

	// Keep trimming until we're under the limit
	for EstimateTokens(BuildSmartPrompt(trimmedContext, "test")) > maxTokens {
		// Trim in priority order
		if len(trimmedContext.RecentMessages) > 3 {
			trimmedContext.RecentMessages = trimmedContext.RecentMessages[:len(trimmedContext.RecentMessages)-1]
		} else if len(trimmedContext.KeyTasks) > 2 {
			trimmedContext.KeyTasks = trimmedContext.KeyTasks[:len(trimmedContext.KeyTasks)-1]
		} else if len(trimmedContext.Summary) > 200 {
			trimmedContext.Summary = trimmedContext.Summary[:200] + "..."
		} else {
			break // Can't trim further
		}
	}

	return trimmedContext
}
