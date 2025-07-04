package supabase

import (
	"clementus360/ai-helper/types"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/supabase-community/postgrest-go"
	"github.com/supabase-community/supabase-go"
)

// SaveTasks saves multiple tasks for a user, applying defaults
func SaveTasks(client *supabase.Client, userID string, items []types.Task) error {
	// Assuming the Task struct includes Title and Description
	for i := range items {
		items[i].UserID = userID
		items[i].Status = "pending"
		items[i].AISuggested = true
		items[i].FollowedUp = false
		if items[i].CreatedAt.IsZero() {
			items[i].CreatedAt = time.Now()
		}
		if items[i].FollowUpDueAt.IsZero() {
			items[i].FollowUpDueAt = time.Now().Add(48 * time.Hour)
		}
	}

	_, _, err := client.From("tasks").Insert(items, false, "", "", "").Execute()
	return err
}

// InsertAndReturnTask inserts a task and returns the saved task with defaults applied
func InsertAndReturnTask(client *supabase.Client, task types.Task) (types.Task, error) {
	// Ensure defaults
	if task.Status == "" {
		task.Status = "pending"
	}
	if task.CreatedAt.IsZero() {
		task.CreatedAt = time.Now()
	}
	if task.FollowUpDueAt.IsZero() {
		task.FollowUpDueAt = task.CreatedAt.Add(48 * time.Hour)
	}

	task.AISuggested = false
	task.FollowedUp = false

	resp, _, err := client.From("tasks").Insert(task, true, "", "", "").Execute()
	if err != nil {
		return types.Task{}, err
	}

	var saved []types.Task
	if err := json.Unmarshal(resp, &saved); err != nil || len(saved) == 0 {
		return types.Task{}, fmt.Errorf("failed to decode inserted task")
	}

	return saved[0], nil
}

// DeleteTask deletes a task by ID and user ID for security
func DeleteTask(client *supabase.Client, taskID, userID string) error {
	if taskID == "" || userID == "" {
		return fmt.Errorf("missing task ID or user ID")
	}

	_, _, err := client.
		From("tasks").
		Delete("", "").
		Eq("id", taskID).
		Eq("user_id", userID). // extra safety
		Execute()

	return err
}

// UpdateTask updates a task by ID and user ID, and returns the updated task
func UpdateTask(client *supabase.Client, taskID, userID string, updates map[string]interface{}) (types.Task, error) {
	if taskID == "" || userID == "" {
		return types.Task{}, fmt.Errorf("missing task ID or user ID")
	}
	if len(updates) == 0 {
		return types.Task{}, fmt.Errorf("empty update payload")
	}

	fmt.Println("Updating task:", taskID, "for user:", userID)
	fmt.Println("Update payload:", updates)

	// Update and return the updated row
	resp, _, err := client.
		From("tasks").
		Update(updates, "", ""). // Return all fields
		Eq("id", taskID).
		Eq("user_id", userID).
		Execute() // Don't use Single()

	if err != nil {
		fmt.Printf("Update error: %v\n", err)
		return types.Task{}, err
	}

	fmt.Printf("Update response: %s\n", string(resp))

	var updated []types.Task
	if err := json.Unmarshal(resp, &updated); err != nil {
		return types.Task{}, fmt.Errorf("failed to decode updated task: %v", err)
	}

	if len(updated) == 0 {
		return types.Task{}, fmt.Errorf("no rows were updated - task may not exist or you may not have permission")
	}

	if len(updated) > 1 {
		return types.Task{}, fmt.Errorf("multiple rows updated unexpectedly")
	}

	return updated[0], nil
}

// GetTasks retrieves all tasks for a user, optionally filtering by status
func GetTasks(client *supabase.Client, userID, sessionID, status string, limit, offset int, search, sortBy, sortOrder string) ([]types.Task, int64, error) {
	if userID == "" {
		return nil, 0, fmt.Errorf("missing user ID")
	}

	query := client.From("tasks").
		Select("*", "exact", false).
		Eq("user_id", userID)

	if sessionID != "" {
		query = query.Eq("session_id", sessionID)
	}
	if status != "" {
		query = query.Eq("status", status)
	}
	if limit > 0 {
		query = query.Limit(limit, "")
	}
	if offset > 0 {
		query = query.Range(offset, offset+limit-1, "")
	}
	if search != "" {
		// Match title or description with case-insensitive partial match
		query = query.Or(fmt.Sprintf("title.ilike.*%s*,description.ilike.*%s*", search, search), "")
	}
	if sortBy != "" {
		direction := "asc"
		if strings.ToLower(sortOrder) == "desc" {
			direction = "desc"
		}
		query = query.Order(sortBy, &postgrest.OrderOpts{Ascending: direction == "asc"})
	}

	resp, count, err := query.Execute()
	if err != nil {
		return nil, 0, err
	}

	var tasks []types.Task
	if err := json.Unmarshal(resp, &tasks); err != nil {
		return nil, 0, fmt.Errorf("failed to decode task data: %w", err)
	}

	return tasks, count, nil
}

// GetTasks retrieves one task for a user
func GetSingleTask(client *supabase.Client, userID, taskID string) ([]types.Task, error) {
	if userID == "" {
		return []types.Task{}, fmt.Errorf("missing user ID")
	}

	query := client.From("tasks").
		Select("*", "exact", false).
		Eq("user_id", userID).
		Eq("id", taskID)

	resp, _, err := query.Execute()
	if err != nil {
		return []types.Task{}, err
	}

	fmt.Println(resp)

	var task []types.Task
	if err := json.Unmarshal(resp, &task); err != nil {
		return []types.Task{}, fmt.Errorf("failed to decode task data: %w", err)
	}

	return task, nil
}
