package handlers

import (
	"clementus360/ai-helper/config"
	"clementus360/ai-helper/supabase"
	"clementus360/ai-helper/types"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"
)

func CreateTaskHandler(w http.ResponseWriter, r *http.Request) {

	var task types.Task

	if err := json.NewDecoder(r.Body).Decode(&task); err != nil {
		config.Logger.Error("Failed to decode task JSON:", err)
		writeError(w, "Invalid JSON body", http.StatusBadRequest)
		return
	}

	// Basic validation
	if task.Title == "" {
		writeError(w, "Missing user_id or title", http.StatusBadRequest)
		return
	}

	// Get Supabase client from request
	supabaseClient, userId, err := supabase.SupabaseClientFromRequest(r)
	if err != nil {
		config.Logger.Error("Failed to create Supabase client:", err)
		writeError(w, "Failed to create Supabase client", http.StatusInternalServerError)
		return
	}

	task.UserID = userId // Set the user ID from the request context

	// Save the task
	savedTask, err := supabase.InsertAndReturnTask(supabaseClient, task)
	if err != nil {
		config.Logger.Error("Failed to save task:", err)
		writeError(w, "Failed to create task", http.StatusInternalServerError)
		return
	}

	// Track task creation activity
	sessionID := ""
	if savedTask.SessionID != nil {
		sessionID = *savedTask.SessionID
	}

	go func() {
		if err := supabase.TrackUserActivity(supabaseClient, userId, sessionID, "task_created", savedTask.Title, map[string]interface{}{
			"task_id":      savedTask.ID,
			"ai_suggested": false,
			"has_due_date": savedTask.DueDate != nil,
		}); err != nil {
			config.Logger.Warn("TrackUserActivity failed:", err)
		}
	}()

	// Update session metrics
	if sessionID != "" {
		go func() {
			if err := supabase.IncrementSessionCounter(supabaseClient, sessionID, "task_created"); err != nil {
				config.Logger.Warn("Failed to incemment session counter:", err)
			}
		}()
	}

	writeJSON(w, http.StatusCreated, types.TaskResponse{
		Success: true,
		Task:    savedTask,
	})
}

func DeleteTaskHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		writeError(w, "Only DELETE is allowed", http.StatusMethodNotAllowed)
		return
	}

	query := r.URL.Query()
	taskID := query.Get("id")

	if taskID == "" {
		writeError(w, "Missing task ID", http.StatusBadRequest)
		return
	}

	supabaseClient, userId, err := supabase.SupabaseClientFromRequest(r)
	if err != nil {
		config.Logger.Error("Failed to create Supabase client:", err)
		writeError(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Attempt to fetch task to get session ID before deletion
	var sessionID string
	tasks, err := supabase.GetSingleTask(supabaseClient, userId, taskID)
	if err != nil {
		config.Logger.Warn("Failed to fetch task before deletion:", err)
	} else if len(tasks) > 0 && tasks[0].SessionID != nil {
		sessionID = *tasks[0].SessionID
	}

	if err := supabase.DeleteTask(supabaseClient, taskID, userId); err != nil {
		config.Logger.Error("Failed to delete task:", err)
		writeError(w, "Could not delete task", http.StatusInternalServerError)
		return
	}

	// Track task deletion
	go func() {
		err := supabase.TrackUserActivity(supabaseClient, userId, sessionID, "task_deleted", fmt.Sprintf("Deleted task %s", taskID), map[string]interface{}{
			"task_id": taskID,
		})
		if err != nil {
			config.Logger.Warn("TrackUserActivity failed:", err)
		}
	}()

	writeJSON(w, http.StatusOK, types.DeleteTaskResponse{
		Success:      true,
		Message:      "Task deleted successfully",
		ErrorMessage: "",
	})
}

func UpdateTaskHandler(w http.ResponseWriter, r *http.Request) {
	taskID := r.URL.Query().Get("id")
	if taskID == "" {
		writeError(w, "Missing task ID", http.StatusBadRequest)
		return
	}
	if _, err := uuid.Parse(taskID); err != nil {
		config.Logger.Error("Invalid task ID format:", err)
		writeError(w, "Invalid task ID", http.StatusBadRequest)
		return
	}

	var updates map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil || len(updates) == 0 {
		config.Logger.Error("Failed to decode update JSON:", err)
		writeError(w, "Invalid or empty update payload", http.StatusBadRequest)
		return
	}

	client, userID, err := supabase.SupabaseClientFromRequest(r)
	if err != nil {
		config.Logger.Error("Failed to create Supabase client:", err)
		writeJSON(w, http.StatusUnauthorized, types.TaskResponse{
			Success:      false,
			ErrorMessage: "Unauthorized",
		})
		return
	}

	updatedTask, err := supabase.UpdateTask(client, taskID, userID, updates)
	if err != nil {
		config.Logger.Error("Failed to update task:", err)
		writeJSON(w, http.StatusInternalServerError, types.TaskResponse{
			Success:      false,
			ErrorMessage: err.Error(),
		})
		return
	}

	// Track task update activity
	sessionID := ""
	if updatedTask.SessionID != nil {
		sessionID = *updatedTask.SessionID
	}

	go func() {
		if err := supabase.TrackUserActivity(client, userID, sessionID, "task_updated", updatedTask.Title, map[string]interface{}{
			"task_id": updatedTask.ID,
			"updates": updates,
			"status":  updatedTask.Status,
		}); err != nil {
			config.Logger.Warn("TrackUserActivity failed:", err)
		}
	}()

	// Special tracking for task completion
	if status, ok := updates["status"]; ok && status == "completed" {
		go func() {
			if err := supabase.TrackUserActivity(client, userID, sessionID, "task_completed", updatedTask.Title, map[string]interface{}{
				"task_id":         updatedTask.ID,
				"completion_time": time.Now(),
			}); err != nil {
				config.Logger.Warn("TrackUserActivity failed:", err)
			}
		}()

		// Update session metrics
		if sessionID != "" {
			go func() {
				if err := supabase.IncrementSessionCounter(client, sessionID, "task_completed"); err != nil {
					config.Logger.Warn("Failed to incemment session counter:", err)
				}
			}()
		}
	}

	writeJSON(w, http.StatusOK, types.TaskResponse{
		Success: true,
		Task:    updatedTask,
	})
}

func GetTasksHandler(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	sessionID := q.Get("session_id")
	status := q.Get("status")
	limitStr := q.Get("limit")
	offsetStr := q.Get("offset")
	search := q.Get("search")
	sortBy := q.Get("sort_by")       // e.g., "created_at", "title", "status"
	sortOrder := q.Get("sort_order") // "asc" or "desc"

	limit := 20 // default
	offset := 0
	var err error

	if limitStr != "" {
		limit, err = strconv.Atoi(limitStr)
		if err != nil || limit < 1 {
			config.Logger.Error("Invalid limit value:", err)
			writeError(w, "Invalid limit value", http.StatusBadRequest)
			return
		}
	}

	if offsetStr != "" {
		offset, err = strconv.Atoi(offsetStr)
		if err != nil || offset < 0 {
			config.Logger.Error("Invalid offset value:", err)
			writeError(w, "Invalid offset value", http.StatusBadRequest)
			return
		}
	}

	supabaseClient, userId, err := supabase.SupabaseClientFromRequest(r)
	if err != nil {
		config.Logger.Error("Failed to create Supabase client:", err)
		writeError(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	tasks, total, err := supabase.GetTasks(supabaseClient, userId, sessionID, status, limit, offset, search, sortBy, sortOrder)
	if err != nil {
		config.Logger.Error("Failed to fetch tasks:", err)
		writeError(w, "Failed to fetch tasks", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, types.GetTasksResponse{
		Success: true,
		Tasks:   tasks,
		Limit:   limit,
		Offset:  offset,
		Total:   int(total),
	})
}

// get a single task by ID
func GetSingleTaskHandler(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	taskID := q.Get("id")

	supabaseClient, userId, err := supabase.SupabaseClientFromRequest(r)
	if err != nil {
		config.Logger.Error("Failed to create Supabase client:", err)
		writeError(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	task, err := supabase.GetSingleTask(supabaseClient, userId, taskID)
	if err != nil {
		config.Logger.Error("Failed to fetch tasks:", err)
		writeError(w, "Failed to fetch task", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, types.GetSingleTaskResponse{
		Success: true,
		Task:    task,
	})
}
