package types

import "time"

type Task struct {
	ID            string     `json:"id,omitempty"`
	UserID        string     `json:"user_id"`
	GoalID        *string    `json:"goal_id,omitempty"` // nullable
	Title         string     `json:"title"`
	Description   string     `json:"description"` // <-- new field
	Status        string     `json:"status"`
	DueDate       *time.Time `json:"due_date,omitempty"`
	AISuggested   bool       `json:"ai_suggested"`
	CreatedAt     time.Time  `json:"created_at"`
	SessionID     *string    `json:"session_id,omitempty"` // for task association with chat sessions
	Decision      string     `json:"decision,omitempty"`   // approved | declined | undecided
	FollowUpDueAt time.Time  `json:"follow_up_due_at,omitempty"`
	FollowedUp    bool       `json:"followed_up,omitempty"`
}

type TaskResponse struct {
	Success      bool   `json:"success"`
	Task         Task   `json:"task,omitempty"`  // the created task
	ErrorMessage string `json:"error,omitempty"` // only set on failure
}

type DeleteTaskResponse struct {
	Success      bool   `json:"success"`
	ErrorMessage string `json:"error,omitempty"`   // only set on failure
	Message      string `json:"message,omitempty"` // confirmation message
}

type GetTasksResponse struct {
	Success      bool   `json:"success"`
	Tasks        []Task `json:"tasks,omitempty"`
	Total        int    `json:"total,omitempty"`  // Optional: total count for pagination
	Limit        int    `json:"limit,omitempty"`  // Echoed back from request
	Offset       int    `json:"offset,omitempty"` // Echoed back from request
	ErrorMessage string `json:"error,omitempty"`  // Only set on failure
}

type SupabaseGetTasksResponse struct {
	Data  []Task `json:"data"`
	Count int64  `json:"count"`
}
