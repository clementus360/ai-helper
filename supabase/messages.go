package supabase

import (
	"clementus360/ai-helper/types"
	"encoding/json"
	"fmt"
	"time"

	"github.com/supabase-community/postgrest-go"
	"github.com/supabase-community/supabase-go"
)

func SaveMessage(client *supabase.Client, userID, sessionID, sender, UserMessageID, content string) (string, error) {
	message := types.Message{
		UserID:        userID,
		SessionID:     sessionID,
		Sender:        sender,
		Content:       content,
		UserMessageID: UserMessageID,
		CreatedAt:     time.Now(),
	}

	var inserted []types.Message

	resp, _, err := client.From("messages").
		Insert(message, false, "return=representation", "", "").Execute()

	if err != nil {
		return "", err
	}

	if err := json.Unmarshal(resp, &inserted); err != nil {
		return "", err
	}

	if len(inserted) == 0 || inserted[0].ID == "" {
		return "", fmt.Errorf("message insert succeeded but no ID returned")
	}

	return inserted[0].ID, nil
}

func GetMessages(client *supabase.Client, sessionID, userID string) ([]types.Message, error) {
	var messages []types.Message

	query := client.
		From("messages").
		Select("*", "", false).
		Eq("session_id", sessionID).
		Eq("user_id", userID).
		Order("created_at", &postgrest.OrderOpts{Ascending: true}) // ascending

	data, _, err := query.Execute()
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(data, &messages)
	if err != nil {
		return nil, err
	}

	return messages, nil
}

func GetRecentMessages(client *supabase.Client, sessionID, userID string, limit int) ([]types.Message, error) {
	var messages []types.Message

	query := client.
		From("messages").
		Select("sender, content, created_at, session_id", "", false).
		Eq("user_id", userID).
		Eq("session_id", sessionID).
		Order("created_at", &postgrest.OrderOpts{Ascending: false}).
		Limit(limit, "") // Get double to allow for filtering

	data, _, err := query.Execute()
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(data, &messages)
	if err != nil {
		return nil, err
	}

	return messages, nil
}
