package supabase

import (
	"clementus360/ai-helper/llm"
	"clementus360/ai-helper/types"
	"encoding/json"
	"fmt"
	"log"
	"slices"
	"strings"
	"time"

	"github.com/supabase-community/postgrest-go"
	"github.com/supabase-community/supabase-go"
)

const (
	MAX_CONTEXT_MESSAGES     = 10
	SUMMARY_UPDATE_THRESHOLD = 20
)

// GetOrCreateActiveSession returns recent session ID or creates a new session
func GetOrCreateActiveSession(client *supabase.Client, userID string, forceNew bool) (string, error) {
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

	// If forceNew is false and we have recent sessions, return the first one
	// This allows us to reuse existing sessions without creating new ones
	// unless the user explicitly requests a new session
	if !forceNew {
		if len(sessions) > 0 {
			return sessions[0].ID, nil
		}
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

	// generate summary using LLM
	summary, err := llm.GenerateSessionSummary(messages)
	if err != nil {
		return err
	}

	data := types.SessionSummary{
		SessionID:   sessionID,
		UserID:      userID,
		Summary:     summary,
		LastUpdated: time.Now(),
	}

	_, _, err = client.From("session_summaries").
		Upsert(data, "", "", "").
		Execute()

	if err != nil {
		return err
	}

	// Fetch session to check current title
	sessionResp, _, err := client.From("sessions").
		Select("id, title", "", false).
		Eq("id", sessionID).
		Eq("user_id", userID).
		Single().
		Execute()
	if err != nil {
		return fmt.Errorf("failed to fetch session: %w", err)
	}

	var session types.Session
	if err := json.Unmarshal(sessionResp, &session); err != nil {
		return fmt.Errorf("failed to parse session: %w", err)
	}

	if session.Title == "" || strings.ToLower(session.Title) == "untitled" {
		// Generate smart context
		smartContext, err := BuildSmartContext(client, sessionID, userID)
		if err != nil {
			return fmt.Errorf("failed to build smart context: %w", err)
		}

		// Ask Gemini for a session title
		title, err := llm.GenerateSessionTitle(smartContext)
		if err != nil {
			return fmt.Errorf("failed to generate session title: %w", err)
		}

		// Save session title
		_, err = UpdateSessionTitle(client, sessionID, userID, title)
		if err != nil {
			return fmt.Errorf("failed to update session title: %w", err)
		}
	}

	return nil
}

func GetSessions(client *supabase.Client, userID string) ([]types.Session, error) {
	if userID == "" {
		return nil, fmt.Errorf("missing user ID")
	}

	query := client.From("sessions").
		Select("*", "", false).
		Eq("user_id", userID).
		Order("created_at", &postgrest.OrderOpts{Ascending: false})

	resp, _, err := query.Execute()
	if err != nil {
		return nil, err
	}

	var sessions []types.Session
	if err := json.Unmarshal(resp, &sessions); err != nil {
		return nil, fmt.Errorf("failed to decode session data: %w", err)
	}

	return sessions, nil
}

func UpdateSessionTitle(client *supabase.Client, sessionID, userID, newTitle string) (types.Session, error) {
	var updated []types.Session

	resp, _, err := client.From("sessions").
		Update(map[string]interface{}{"title": newTitle}, "", "").
		Eq("id", sessionID).
		Eq("user_id", userID).
		Execute()

	if err != nil {
		return types.Session{}, fmt.Errorf("failed to update session title: %w", err)
	}

	if err := json.Unmarshal(resp, &updated); err != nil {
		return types.Session{}, fmt.Errorf("failed to parse update result: %w", err)
	}

	if len(updated) == 0 {
		return types.Session{}, fmt.Errorf("no session found or updated")
	}

	return updated[0], nil
}
