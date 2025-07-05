package types

import "time"

type ActivityMetadata struct {
	MessageLength    int       `json:"message_length,omitempty"`
	TaskCount        int       `json:"task_count,omitempty"`
	AIsuggested      bool      `json:"ai_suggested,omitempty"`
	HasDueDate       bool      `json:"has_due_date,omitempty"`
	CompletionTime   time.Time `json:"completion_time,omitempty"`
	ResponseLength   int       `json:"response_length,omitempty"`
	ActionItemsCount int       `json:"action_items_count,omitempty"`
}

type ContextConfig struct {
	MaxTokens                int `json:"max_tokens"`
	MaxRecentMessages        int `json:"max_recent_messages"`
	MaxKeyTasks              int `json:"max_key_tasks"`
	SummaryMaxLength         int `json:"summary_max_length"`
	MessagePriorityThreshold int `json:"message_priority_threshold"`
}
