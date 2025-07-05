package supabase

import (
	"clementus360/ai-helper/types"
	"fmt"

	"github.com/supabase-community/supabase-go"
)

// Build smart context with enhanced data
func BuildSmartContext(client *supabase.Client, sessionID, userID string) (types.SmartContext, error) {
	context := types.SmartContext{}

	// 1. Get session summary (existing function)
	summary, err := getSessionSummary(client, sessionID)
	if err != nil {
		// Log but don't fail
		fmt.Printf("Warning: Could not fetch session summary: %v\n", err)
	}
	context.Summary = summary

	// 2. Get recent messages with smart filtering
	recentMessages, err := getRecentMessagesWithPriority(client, sessionID, userID, 10)
	if err != nil {
		fmt.Printf("Warning: Could not fetch recent messages: %v\n", err)
	}
	context.RecentMessages = recentMessages

	// 3. Get key tasks
	keyTasks, err := getKeyTasks(client, sessionID, userID)
	if err != nil {
		fmt.Printf("Warning: Could not fetch key tasks: %v\n", err)
	}
	context.KeyTasks = keyTasks

	// 4. Get session metrics
	metrics, err := GetOrCreateSessionMetrics(client, sessionID, userID)
	if err != nil {
		fmt.Printf("Warning: Could not fetch session metrics: %v\n", err)
	}
	context.SessionMetrics = metrics

	// 5. Get user patterns
	patterns, err := GetUserPatterns(client, userID)
	if err != nil {
		fmt.Printf("Warning: Could not fetch user patterns: %v\n", err)
	}
	context.UserPatterns = patterns

	// 6. Generate priority signals
	context.PrioritySignals = generatePrioritySignals(context)

	return context, nil
}

// Helper functions (internal to this package)
func getSessionSummary(client *supabase.Client, sessionID string) (string, error) {
	// Your existing summary logic
	return "", nil
}

func getRecentMessagesWithPriority(client *supabase.Client, sessionID, userID string, limit int) ([]types.Message, error) {
	// Get more messages than needed for filtering
	messages, err := GetRecentMessages(client, sessionID, userID, limit*2)
	if err != nil {
		return nil, err
	}

	// Apply smart filtering
	return prioritizeMessages(messages, limit), nil
}

func getKeyTasks(client *supabase.Client, sessionID, userID string) ([]types.Task, error) {
	// Your existing task fetching logic with priority filtering
	return nil, nil
}

func prioritizeMessages(messages []types.Message, limit int) []types.Message {
	if len(messages) <= limit {
		return messages
	}

	// Score and sort messages by importance
	// Implementation depends on your specific scoring algorithm
	return messages[:limit]
}

func generatePrioritySignals(context types.SmartContext) []string {
	var signals []string

	// Analyze context and generate priority signals
	if context.SessionMetrics.DominantMood != "" {
		signals = append(signals, fmt.Sprintf("User's dominant mood: %s", context.SessionMetrics.DominantMood))
	}

	if context.SessionMetrics.TasksCreated > 0 && context.SessionMetrics.TasksCompleted == 0 {
		signals = append(signals, "User creates tasks but may need help with execution")
	}

	return signals
}
