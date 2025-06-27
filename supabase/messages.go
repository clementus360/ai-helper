package supabase

import (
	"clementus360/ai-helper/types"

	"github.com/supabase-community/supabase-go"
)

func SaveMessage(client *supabase.Client, userID, sessionID, sender, content string) error {
	message := types.Message{
		UserID:    userID,
		SessionID: sessionID,
		Sender:    sender,
		Content:   content,
	}

	_, _, err := client.From("messages").Insert(message, false, "", "", "").Execute()
	return err
}
