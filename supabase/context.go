package supabase

import (
	"clementus360/ai-helper/types"
	"encoding/json"
	"fmt"
	"time"

	"github.com/supabase-community/postgrest-go"
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

	// Apply smart filtering while maintaining chronological order
	return messages, nil
}

func getKeyTasks(client *supabase.Client, sessionID, userID string) ([]types.Task, error) {
	// Get pending tasks, but also include recently completed ones for context
	// This helps the AI understand what's been accomplished
	query := client.From("tasks").
		Select("*", "exact", false).
		Eq("user_id", userID)

	if sessionID != "" {
		query = query.Eq("session_id", sessionID)
	}

	// Get pending tasks AND recently completed tasks (last 7 days)
	sevenDaysAgo := time.Now().AddDate(0, 0, -7)
	query = query.Or(
		fmt.Sprintf("status.eq.pending,and(status.eq.completed,created_at.gte.%s)",
			sevenDaysAgo.Format("2006-01-02T15:04:05")),
		"",
	)

	// Order by priority: pending first, then by due date, then by creation date
	query = query.Order("status", &postgrest.OrderOpts{Ascending: true}). // pending comes before completed
										Order("due_date", &postgrest.OrderOpts{Ascending: true, NullsFirst: false}).
										Order("created_at", &postgrest.OrderOpts{Ascending: false}).
										Limit(15, "") // Increased limit to include more context

	resp, _, err := query.Execute()
	if err != nil {
		return nil, fmt.Errorf("failed to get key tasks: %w", err)
	}

	var tasks []types.Task
	if err := json.Unmarshal(resp, &tasks); err != nil {
		return nil, fmt.Errorf("failed to decode task data: %w", err)
	}

	return tasks, nil
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
