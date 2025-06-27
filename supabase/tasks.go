package supabase

import (
	"clementus360/ai-helper/types"
	"time"

	"github.com/supabase-community/supabase-go"
)

func SaveTasks(client *supabase.Client, userID string, items []types.Task) error {
	// Assuming the Task struct includes Title and Description
	for i := range items {
		items[i].UserID = userID
		items[i].Status = "pending"
		items[i].AISuggested = true
		if items[i].CreatedAt.IsZero() {
			items[i].CreatedAt = time.Now()
		}
	}

	_, _, err := client.From("tasks").Insert(items, false, "", "", "").Execute()
	return err
}
