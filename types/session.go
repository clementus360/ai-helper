package types

import "time"

// New session-related types
type Session struct {
	ID        string     `json:"id,omitempty"` // <-- omitempty is critical
	UserID    string     `json:"user_id"`
	Title     string     `json:"title"`
	CreatedAt *time.Time `json:"created_at,omitempty"`
}

type SessionSummary struct {
	SessionID   string    `json:"session_id"`
	UserID      string    `json:"user_id"`
	Summary     string    `json:"summary"`
	LastUpdated time.Time `json:"last_updated,omitempty"`
}

type GetSessionsResponse struct {
	Success  bool      `json:"success"`
	Sessions []Session `json:"sessions"`
}

type SessionResponse struct {
	Success bool    `json:"success"`
	Session Session `json:"session"`
}
