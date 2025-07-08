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
	summary, err := GetSessionSummary(client, sessionID)
	if err != nil {
		log.Printf("Failed to fetch session summary: %v", err)
	}
	context.Summary = summary

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
		return fmt.Errorf("failed to fetch session summaries: %w", err)
	}
	var summaries []types.SessionSummary
	if err := json.Unmarshal(summaryResp, &summaries); err != nil {
		return fmt.Errorf("failed to parse session summaries: %w", err)
	}

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
		return fmt.Errorf("failed to count messages: %w", err)
	}
	var newMessages []types.Message
	if err := json.Unmarshal(countResp, &newMessages); err != nil {
		return fmt.Errorf("failed to parse messages: %w", err)
	}
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
		return fmt.Errorf("failed to fetch messages: %w", err)
	}
	var messages []types.Message
	if err := json.Unmarshal(allResp, &messages); err != nil {
		return fmt.Errorf("failed to parse messages: %w", err)
	}
	if len(messages) < 5 {
		return nil
	}

	// Generate smart context
	smartContext, err := BuildSmartContext(client, sessionID, userID)
	if err != nil {
		return fmt.Errorf("failed to build smart context: %w", err)
	}

	// Generate summary and title
	summary, title, err := llm.GenerateSessionSummaryAndTitle(messages, smartContext)
	if err != nil {
		return fmt.Errorf("failed to generate summary and title: %w", err)
	}

	// Save summary
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
		return fmt.Errorf("failed to save summary: %w", err)
	}

	// Save session title
	_, err = UpdateSessionTitle(client, sessionID, userID, title)
	if err != nil {
		return fmt.Errorf("failed to update session title: %w", err)
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
		Is("deleted_at", "null").
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

func GetSessionSummary(client *supabase.Client, sessionID string) (string, error) {
	summaryResp, _, err := client.From("session_summaries").
		Select("summary", "", false).
		Eq("session_id", sessionID).
		Execute()
	if err != nil {
		return "", fmt.Errorf("failed to fetch session summary: %w", err)
	}

	var summaries []types.SessionSummary
	if err := json.Unmarshal(summaryResp, &summaries); err != nil {
		return "", fmt.Errorf("failed to unmarshal session summary: %w", err)
	}

	if len(summaries) == 0 {
		return "", nil // No summary yet
	}

	return summaries[0].Summary, nil
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

// DeleteSession soft deletes a session and all related data
func DeleteSession(client *supabase.Client, sessionID, userID string) error {
	if sessionID == "" || userID == "" {
		return fmt.Errorf("session ID and user ID are required")
	}

	// Start transaction-like operations
	now := time.Now()

	// Soft delete the session
	_, _, err := client.From("sessions").
		Update(map[string]interface{}{
			"deleted_at": now.Format(time.RFC3339),
		}, "", "").
		Eq("id", sessionID).
		Eq("user_id", userID).
		Is("deleted_at", "null"). // Only delete if not already deleted
		Execute()

	if err != nil {
		return fmt.Errorf("failed to delete session: %w", err)
	}

	// Soft delete related messages
	_, _, err = client.From("messages").
		Update(map[string]interface{}{
			"deleted_at": now.Format(time.RFC3339),
		}, "", "").
		Eq("session_id", sessionID).
		Eq("user_id", userID).
		Is("deleted_at", "null").
		Execute()

	if err != nil {
		log.Printf("Warning: failed to soft delete messages for session %s: %v", sessionID, err)
	}

	// Soft delete related tasks
	_, _, err = client.From("tasks").
		Update(map[string]interface{}{
			"deleted_at": now.Format(time.RFC3339),
		}, "", "").
		Eq("session_id", sessionID).
		Eq("user_id", userID).
		Is("deleted_at", "null").
		Execute()

	if err != nil {
		log.Printf("Warning: failed to soft delete tasks for session %s: %v", sessionID, err)
	}

	// Soft delete user activities
	_, _, err = client.From("user_activities").
		Update(map[string]interface{}{
			"deleted_at": now.Format(time.RFC3339),
		}, "", "").
		Eq("session_id", sessionID).
		Eq("user_id", userID).
		Is("deleted_at", "null").
		Execute()

	if err != nil {
		log.Printf("Warning: failed to soft delete user activities for session %s: %v", sessionID, err)
	}

	// Soft delete session summary
	_, _, err = client.From("session_summaries").
		Update(map[string]interface{}{
			"deleted_at": now.Format(time.RFC3339),
		}, "", "").
		Eq("session_id", sessionID).
		Eq("user_id", userID).
		Is("deleted_at", "null").
		Execute()

	if err != nil {
		log.Printf("Warning: failed to soft delete session summary for session %s: %v", sessionID, err)
	}

	return nil
}

// RestoreSession restores a soft-deleted session and all related data
func RestoreSession(client *supabase.Client, sessionID, userID string) error {
	if sessionID == "" || userID == "" {
		return fmt.Errorf("session ID and user ID are required")
	}

	// Restore the session
	_, _, err := client.From("sessions").
		Update(map[string]interface{}{
			"deleted_at": nil,
		}, "", "").
		Eq("id", sessionID).
		Eq("user_id", userID).
		Not("deleted_at", "is", "null"). // Only restore if deleted
		Execute()

	if err != nil {
		return fmt.Errorf("failed to restore session: %w", err)
	}

	// Restore related data
	tables := []string{"messages", "tasks", "user_activities", "session_summaries"}
	for _, table := range tables {
		_, _, err = client.From(table).
			Update(map[string]interface{}{
				"deleted_at": nil,
			}, "", "").
			Eq("session_id", sessionID).
			Eq("user_id", userID).
			Not("deleted_at", "is", "null").
			Execute()

		if err != nil {
			log.Printf("Warning: failed to restore %s for session %s: %v", table, sessionID, err)
		}
	}

	return nil
}

// HardDeleteSession permanently deletes a session and all related data
// Use with extreme caution - this cannot be undone
func HardDeleteSession(client *supabase.Client, sessionID, userID string) error {
	if sessionID == "" || userID == "" {
		return fmt.Errorf("session ID and user ID are required")
	}

	// Delete in reverse dependency order
	tables := []string{"user_activities", "tasks", "messages", "session_summaries", "sessions"}

	for _, table := range tables {
		_, _, err := client.From(table).
			Delete("", "").
			Eq("session_id", sessionID).
			Eq("user_id", userID).
			Execute()

		if err != nil {
			return fmt.Errorf("failed to hard delete from %s: %w", table, err)
		}
	}

	return nil
}

// GetDeletedSessions returns soft-deleted sessions for a user
func GetDeletedSessions(client *supabase.Client, userID string) ([]types.Session, error) {
	if userID == "" {
		return nil, fmt.Errorf("missing user ID")
	}

	resp, _, err := client.From("sessions").
		Select("*", "", false).
		Eq("user_id", userID).
		Not("deleted_at", "is", "null").
		Order("deleted_at", &postgrest.OrderOpts{Ascending: false}).
		Execute()

	if err != nil {
		return nil, err
	}

	var sessions []types.Session
	if err := json.Unmarshal(resp, &sessions); err != nil {
		return nil, fmt.Errorf("failed to decode session data: %w", err)
	}

	return sessions, nil
}

// CleanupOldDeletedSessions permanently removes soft-deleted sessions older than specified duration
func CleanupOldDeletedSessions(client *supabase.Client, olderThan time.Duration) error {
	cutoff := time.Now().Add(-olderThan)

	// Get sessions to be permanently deleted
	resp, _, err := client.From("sessions").
		Select("id, user_id", "", false).
		Not("deleted_at", "is", "null").
		Lt("deleted_at", cutoff.Format(time.RFC3339)).
		Execute()

	if err != nil {
		return fmt.Errorf("failed to fetch old deleted sessions: %w", err)
	}

	var sessions []types.Session
	if err := json.Unmarshal(resp, &sessions); err != nil {
		return fmt.Errorf("failed to decode session data: %w", err)
	}

	// Hard delete each session
	for _, session := range sessions {
		if err := HardDeleteSession(client, session.ID, session.UserID); err != nil {
			log.Printf("Failed to cleanup session %s: %v", session.ID, err)
		}
	}

	return nil
}
