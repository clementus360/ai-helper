package types

import "time"

// Enhanced context types
type UserActivity struct {
	ID           string    `json:"id,omitempty"`
	UserID       string    `json:"user_id"`
	SessionID    string    `json:"session_id"`
	ActivityType string    `json:"activity_type"` // "message", "task_created", "task_updated", "task_completed", "task_deleted"
	Content      string    `json:"content"`
	Metadata     string    `json:"metadata,omitempty"` // JSON string for additional context
	CreatedAt    time.Time `json:"created_at"`
}

type SessionMetrics struct {
	SessionID           string    `json:"session_id"`
	UserID              string    `json:"user_id"`
	MessageCount        int       `json:"message_count"`
	TasksCreated        int       `json:"tasks_created"`
	TasksCompleted      int       `json:"tasks_completed"`
	LastActiveAt        time.Time `json:"last_active_at"`
	DominantMood        string    `json:"dominant_mood,omitempty"`
	PrimaryTopics       []string  `json:"primary_topics,omitempty"`
	UserEngagementLevel string    `json:"engagement_level"` // "low", "medium", "high"
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`
}

type UserPatterns struct {
	UserID                 string    `json:"user_id"`
	PreferredResponseStyle string    `json:"preferred_response_style"` // "task_focused", "discussion_focused", "balanced"
	CommonStruggles        []string  `json:"common_struggles"`
	SuccessfulStrategies   []string  `json:"successful_strategies"`
	TimePreferences        string    `json:"time_preferences,omitempty"`
	LastAnalyzed           time.Time `json:"last_analyzed"`
	CreatedAt              time.Time `json:"created_at"`
	UpdatedAt              time.Time `json:"updated_at"`
}

type SmartContext struct {
	Summary         string         `json:"summary"`
	RecentMessages  []Message      `json:"recent_messages"`
	KeyTasks        []Task         `json:"key_tasks"`
	SessionMetrics  SessionMetrics `json:"session_metrics"`
	UserPatterns    UserPatterns   `json:"user_patterns"`
	PrioritySignals []string       `json:"priority_signals"`
}

// Enhanced session context (backward compatible)
type SessionContext struct {
	Summary        string    `json:"summary"`
	RecentMessages []Message `json:"recent_messages"`
	// Can be extended with SmartContext fields as needed
}
