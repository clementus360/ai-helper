package types

import "time"

type Task struct {
	ID          string     `json:"id,omitempty"`
	UserID      string     `json:"user_id"`
	GoalID      *string    `json:"goal_id,omitempty"` // nullable
	Title       string     `json:"title"`
	Description string     `json:"description"` // <-- new field
	Status      string     `json:"status"`
	DueDate     *time.Time `json:"due_date,omitempty"`
	AISuggested bool       `json:"ai_suggested"`
	CreatedAt   time.Time  `json:"created_at"`
	SessionID   string     `json:"session_id"`         // for task association with chat sessions
	Decision    string     `json:"decision,omitempty"` // approved | declined | undecided
}
