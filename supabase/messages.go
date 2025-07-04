package supabase

import (
	"clementus360/ai-helper/types"
	"encoding/json"
	"time"

	"github.com/supabase-community/postgrest-go"
	"github.com/supabase-community/supabase-go"
)

func SaveMessage(client *supabase.Client, userID, sessionID, sender, content string) error {
	message := types.Message{
		UserID:    userID,
		SessionID: sessionID,
		Sender:    sender,
		Content:   content,
		CreatedAt: time.Now(),
	}

	_, _, err := client.From("messages").Insert(message, false, "", "", "").Execute()
	return err
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
