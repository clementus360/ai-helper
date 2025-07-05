package supabase

import (
	"clementus360/ai-helper/types"
	"encoding/json"
	"fmt"
	"time"

	"github.com/supabase-community/postgrest-go"
	"github.com/supabase-community/supabase-go"
)

// Track user activity with enhanced metadata
func TrackUserActivity(client *supabase.Client, userID, sessionID, activityType, content string, metadata map[string]interface{}) error {
	metadataJSON, _ := json.Marshal(metadata)

	activity := types.UserActivity{
		UserID:       userID,
		SessionID:    sessionID,
		ActivityType: activityType,
		Content:      content,
		Metadata:     string(metadataJSON),
		CreatedAt:    time.Now(),
	}

	_, _, err := client.From("user_activities").Insert(activity, false, "", "", "").Execute()
	if err != nil {
		return fmt.Errorf("failed to track user activity: %w", err)
	}

	return nil
}

// Get user activities for analysis
func GetUserActivities(client *supabase.Client, userID string, since time.Time, limit int) ([]types.UserActivity, error) {
	resp, _, err := client.From("user_activities").
		Select("*", "", false).
		Eq("user_id", userID).
		Gte("created_at", since.Format(time.RFC3339)).
		Order("created_at", &postgrest.OrderOpts{Ascending: false}).
		Limit(limit, "").
		Execute()

	if err != nil {
		return nil, fmt.Errorf("failed to fetch user activities: %w", err)
	}

	var activities []types.UserActivity
	if err := json.Unmarshal(resp, &activities); err != nil {
		return nil, fmt.Errorf("failed to unmarshal activities: %w", err)
	}

	return activities, nil
}

// Get session activities
func GetSessionActivities(client *supabase.Client, sessionID string, limit int) ([]types.UserActivity, error) {
	resp, _, err := client.From("user_activities").
		Select("*", "", false).
		Eq("session_id", sessionID).
		Order("created_at", &postgrest.OrderOpts{Ascending: false}).
		Limit(limit, "").
		Execute()

	if err != nil {
		return nil, fmt.Errorf("failed to fetch session activities: %w", err)
	}

	var activities []types.UserActivity
	if err := json.Unmarshal(resp, &activities); err != nil {
		return nil, fmt.Errorf("failed to unmarshal activities: %w", err)
	}

	return activities, nil
}
