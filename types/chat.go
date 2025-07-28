package types

import (
	"time"
)

type Message struct {
	ID            string    `json:"id,omitempty"`
	UserID        string    `json:"user_id"`
	Sender        string    `json:"sender"`
	Content       string    `json:"content"`
	CreatedAt     time.Time `json:"created_at,omitempty"`
	SessionID     string    `json:"session_id"`                // for associating messages with chat sessions
	UserMessageID string    `json:"user_message_id,omitempty"` // for linking to user messages
}

type ChatRequest struct {
	Message   string `json:"message"`
	SessionID string `json:"session_id,omitempty"`
	ForceNew  bool   `json:"force_new,omitempty"` // if true, create a new session even if one exists
}

type ChatResponse struct {
	Success      bool   `json:"success"`
	UserMessage  string `json:"user_message"`
	AIResponse   string `json:"ai_response,omitempty"`  // blank for now
	ActionItems  []Task `json:"action_items,omitempty"` // future task suggestions
	ErrorMessage string `json:"error,omitempty"`        // only set on failure
	SessionID    string `json:"session_id"`
}

type GetMessagesResponse struct {
	Success  bool      `json:"success"`
	Messages []Message `json:"messages"`
}
