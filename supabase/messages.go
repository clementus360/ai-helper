package supabase

import (
	"clementus360/ai-helper/types"
	"fmt"
	"time"

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

	fmt.Println(message)

	_, _, err := client.From("messages").Insert(message, false, "", "", "").Execute()
	return err
}
