package supabase

import (
	"clementus360/ai-helper/llm"
	"clementus360/ai-helper/types"
	"encoding/json"
	"fmt"
	"log"
	"slices"
	"time"

	"github.com/supabase-community/postgrest-go"
	"github.com/supabase-community/supabase-go"
)

const (
	MAX_CONTEXT_MESSAGES     = 10
	SUMMARY_UPDATE_THRESHOLD = 20
)

// GetOrCreateActiveSession returns recent session ID or creates a new session
func GetOrCreateActiveSession(client *supabase.Client, userID string) (string, error) {
	cutoff := time.Now().Add(-24 * time.Hour)
	var sessions []types.Session

	resp, _, err := client.From("sessions").
		Select("id, user_id, title, created_at", "", false).
		Eq("user_id", userID).
		Gte("created_at", cutoff.Format(time.RFC3339)).
		Order("created_at", nil).
		Limit(1, "").
		Execute()

	if err != nil {
		return "", err
	}

	if err := json.Unmarshal(resp, &sessions); err != nil {
		return "", err
	}
	if len(sessions) > 0 {
		return sessions[0].ID, nil
	}

	// Create new session
	newSession := types.Session{
		UserID: userID,
		Title:  time.Now().Format("Jan 2, 3:04PM"),
		// Do NOT set CreatedAt
	}

	created := []types.Session{newSession}

	// Insert new session
	resp, _, err = client.From("sessions").Insert(created, false, "", "", "").Execute()
	if err != nil {
		return "", fmt.Errorf("failed to insert session: %w", err)
	}

	if err := json.Unmarshal(resp, &created); err != nil {
		return "", err
	}
	return created[0].ID, nil
}

func GetSessionContext(client *supabase.Client, sessionID, userID string) (types.SessionContext, error) {
	context := types.SessionContext{}

	// Get session summary (non-critical, log but don't fail)
	summaryResp, _, err := client.From("session_summaries").
		Select("summary", "", false).
		Eq("session_id", sessionID).
		Execute()

	if err != nil {
		log.Printf("Failed to fetch session summary for session %s: %v", sessionID, err)
	} else {
		var summaries []types.SessionSummary
		if err := json.Unmarshal(summaryResp, &summaries); err != nil {
			log.Printf("Failed to unmarshal session summary: %v", err)
		} else if len(summaries) > 0 {
			context.Summary = summaries[0].Summary
		}
	}

	// Get recent messages (critical)
	msgsResp, _, err := client.From("messages").
		Select("sender, content, created_at, session_id", "", false).
		Eq("user_id", userID).
		Eq("session_id", sessionID).
		Order("created_at", &postgrest.OrderOpts{Ascending: false}).
		Limit(MAX_CONTEXT_MESSAGES, "").
		Execute()

	if err != nil {
		return context, fmt.Errorf("failed to fetch messages: %w", err)
	}

	var messages []types.Message
	if err := json.Unmarshal(msgsResp, &messages); err != nil {
		return context, fmt.Errorf("failed to unmarshal messages: %w", err)
	}

	// Reverse to chronological order
	slices.Reverse(messages) // Go 1.21+ has this built-in

	context.RecentMessages = messages

	return context, nil
}

// UpdateSessionSummaryIfNeeded checks whether a summary update is needed
func UpdateSessionSummaryIfNeeded(client *supabase.Client, sessionID, userID string) error {
	// Get last summary update
	summaryResp, _, err := client.From("session_summaries").
		Select("last_updated", "", false).
		Eq("session_id", sessionID).
		Execute()
	if err != nil {
		return err
	}
	var summaries []types.SessionSummary
	_ = json.Unmarshal(summaryResp, &summaries)

	var lastUpdate time.Time
	if len(summaries) > 0 {
		lastUpdate = summaries[0].LastUpdated
	}

	// Count messages since last update
	countResp, _, err := client.From("messages").
		Select("id", "", false).
		Eq("user_id", userID).
		Eq("session_id", sessionID).
		Gt("created_at", lastUpdate.Format(time.RFC3339)).
		Execute()
	if err != nil {
		return err
	}
	var newMessages []types.Message
	_ = json.Unmarshal(countResp, &newMessages)
	if len(newMessages) < SUMMARY_UPDATE_THRESHOLD {
		return nil
	}

	// Get all messages for summary
	allResp, _, err := client.From("messages").
		Select("sender, content, created_at", "", false).
		Eq("user_id", userID).
		Eq("session_id", sessionID).
		Order("created_at", nil).
		Execute()
	if err != nil {
		return err
	}
	var messages []types.Message
	_ = json.Unmarshal(allResp, &messages)
	if len(messages) < 5 {
		return nil
	}

	summary, err := llm.GenerateSessionSummary(messages)
	if err != nil {
		return err
	}

	data := types.SessionSummary{
		SessionID:   sessionID,
		Summary:     summary,
		LastUpdated: time.Now(),
	}

	_, _, err = client.From("session_summaries").
		Upsert(data, "", "", "").
		Execute()
	return err
}
