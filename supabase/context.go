package supabase

import (
	"clementus360/ai-helper/types"
	"fmt"
	"slices"
	"strings"

	"github.com/supabase-community/supabase-go"
)

// Build smart context with enhanced data
func BuildSmartContext(client *supabase.Client, sessionID, userID string) (types.SmartContext, error) {
	context := types.SmartContext{}

	// 1. Get session summary (existing function)
	summary, err := GetSessionSummary(client, sessionID)
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
	// Only fetch pending tasks for this session to keep context concise
	tasks, _, err := GetTasks(client, userID, sessionID, "pending", 10, 0, "", "due_date", "asc")
	if err != nil {
		return nil, fmt.Errorf("failed to get key tasks: %w", err)
	}
	return tasks, nil
}

func prioritizeMessages(messages []types.Message, limit int) []types.Message {
	type ScoredMessage struct {
		Message types.Message
		Score   int
	}

	var scored []ScoredMessage
	for _, msg := range messages {
		score := 0
		if msg.Sender == "user" && len(msg.Content) > 80 {
			score += 2
		}
		if msg.Sender == "ai" && strings.Contains(msg.Content, "task") {
			score += 2
		}
		if msg.Sender == "user" && strings.Contains(msg.Content, "?") {
			score++
		}
		scored = append(scored, ScoredMessage{Message: msg, Score: score})
	}

	// Sort by score desc
	slices.SortFunc(scored, func(a, b ScoredMessage) int {
		return b.Score - a.Score
	})

	// Return top N messages
	var prioritized []types.Message
	for i := 0; i < limit && i < len(scored); i++ {
		prioritized = append(prioritized, scored[i].Message)
	}
	return prioritized
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
