package supabase

import (
	"clementus360/ai-helper/types"
	"encoding/json"
	"fmt"
	"time"

	"github.com/supabase-community/supabase-go"
)

// Get or create session metrics
func GetOrCreateSessionMetrics(client *supabase.Client, sessionID, userID string) (types.SessionMetrics, error) {
	// Try to get existing metrics
	resp, _, err := client.From("session_metrics").
		Select("*", "", false).
		Eq("session_id", sessionID).
		Execute()

	if err != nil {
		return types.SessionMetrics{}, fmt.Errorf("failed to fetch session metrics: %w", err)
	}

	var metrics []types.SessionMetrics
	if err := json.Unmarshal(resp, &metrics); err != nil {
		return types.SessionMetrics{}, fmt.Errorf("failed to unmarshal metrics: %w", err)
	}

	// Return existing metrics if found
	if len(metrics) > 0 {
		return metrics[0], nil
	}

	// Create new metrics
	newMetrics := types.SessionMetrics{
		SessionID:           sessionID,
		UserID:              userID,
		MessageCount:        0,
		TasksCreated:        0,
		TasksCompleted:      0,
		LastActiveAt:        time.Now(),
		UserEngagementLevel: "medium",
		CreatedAt:           time.Now(),
		UpdatedAt:           time.Now(),
	}

	_, _, err = client.From("session_metrics").Insert(newMetrics, false, "", "", "").Execute()
	if err != nil {
		return types.SessionMetrics{}, fmt.Errorf("failed to create session metrics: %w", err)
	}

	return newMetrics, nil
}

// Update session metrics
func UpdateSessionMetrics(client *supabase.Client, sessionID string, updates map[string]interface{}) error {
	updates["updated_at"] = time.Now()
	updates["last_active_at"] = time.Now()

	_, _, err := client.From("session_metrics").
		Update(updates, "", "").
		Eq("session_id", sessionID).
		Execute()

	if err != nil {
		return fmt.Errorf("failed to update session metrics: %w", err)
	}

	return nil
}

// Increment session metric counters
func IncrementSessionCounter(client *supabase.Client, sessionID, counterType string) error {
	// Using raw SQL for atomic increment
	err := client.Rpc("increment_session_counter", "", map[string]interface{}{
		"input_session_id": sessionID,
		"input_counter":    counterType,
	})

	if err != "" {
		return fmt.Errorf("failed to increment session counter: %s", err)
	}

	return nil
}
