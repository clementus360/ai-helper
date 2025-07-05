package config

import "clementus360/ai-helper/types"

// Context configuration
var ContextConfig = types.ContextConfig{
	MaxTokens:                6000,
	MaxRecentMessages:        10,
	MaxKeyTasks:              5,
	SummaryMaxLength:         500,
	MessagePriorityThreshold: 2,
}

// Activity types constants
const (
	ActivityTypeMessage       = "message"
	ActivityTypeTaskCreated   = "task_created"
	ActivityTypeTaskUpdated   = "task_updated"
	ActivityTypeTaskCompleted = "task_completed"
	ActivityTypeTaskDeleted   = "task_deleted"
	ActivityTypeAIResponse    = "ai_response"
	ActivityTypeTasksCreated  = "tasks_created"
)
