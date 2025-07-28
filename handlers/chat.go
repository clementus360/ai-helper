package handlers

import (
	"clementus360/ai-helper/config"
	"clementus360/ai-helper/llm"
	"clementus360/ai-helper/supabase"
	"clementus360/ai-helper/types"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
)

func ChatHandler(w http.ResponseWriter, r *http.Request) {
	// Parse and validate the request body
	var req types.ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, "Invalid JSON body", http.StatusBadRequest)
		return
	}
	if req.Message == "" {
		writeError(w, "Missing user_id or message", http.StatusBadRequest)
		return
	}

	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		config.Logger.Warn("Missing Authorization header")
		writeError(w, "Missing Authorization header", http.StatusUnauthorized)
		return
	}
	supabaseClient, userId, err := supabase.SupabaseClientFromRequest(r)
	if err != nil {
		config.Logger.Error("Failed to create Supabase client:", err)
		writeError(w, "Failed to create Supabase client", http.StatusInternalServerError)
		return
	}

	// Get or create active session
	var sessionID string
	if req.SessionID != "" {
		sessionID = req.SessionID // Use provided session
	} else {
		var err error
		sessionID, err = supabase.GetOrCreateActiveSession(supabaseClient, userId, req.ForceNew) // Create new one
		if err != nil {
			config.Logger.Error("Failed to get or create session:", err)
			writeError(w, "Could not manage session", http.StatusInternalServerError)
			return
		}
	}

	// Get SMART context instead of basic context
	smartContext, err := supabase.BuildSmartContext(supabaseClient, sessionID, userId)
	if err != nil {
		config.Logger.Warn("Failed to get smart context:", err)
		// Continue with basic context as fallback
		basicContext, _ := supabase.GetSessionContext(supabaseClient, sessionID, userId)
		smartContext = types.SmartContext{
			Summary:        basicContext.Summary,
			RecentMessages: basicContext.RecentMessages,
		}
	}

	// Save the user message
	userMessageId, err := supabase.SaveMessage(supabaseClient, userId, sessionID, "user", "", req.Message)
	if err != nil {
		config.Logger.Error("Failed to save message:", err)
		writeError(w, "Could not save message", http.StatusInternalServerError)
		return
	}

	// Track user message activity
	go func() {
		if err := supabase.TrackUserActivity(supabaseClient, userId, sessionID, "message", req.Message, map[string]interface{}{
			"message_length": len(req.Message),
			"timestamp":      time.Now(),
		}); err != nil {
			config.Logger.Warn("TrackUserActivity failed:", err)
		}
	}()

	// Generate AI response with enhanced context
	structuredResp, err := llm.GenerateResponse(req.Message, smartContext, "gemini")
	if err != nil {
		config.Logger.Error("Failed to get AI response:", err)
		structuredResp = llm.GeminiStructuredResponse{
			Response:    "I'm having trouble processing that right now. Could you rephrase what you're struggling with?",
			ActionItems: []llm.GeminiTaskItem{},
		}
	}

	// Save AI response
	messageId, err := supabase.SaveMessage(supabaseClient, userId, sessionID, "ai", userMessageId, structuredResp.Response)
	if err != nil {
		config.Logger.Warn("Failed to save AI message:", err)
	}

	// Track AI response activity
	go func() {
		if err := supabase.TrackUserActivity(supabaseClient, userId, sessionID, "ai_response", structuredResp.Response, map[string]interface{}{
			"response_length":    len(structuredResp.Response),
			"action_items_count": len(structuredResp.ActionItems),
		}); err != nil {
			config.Logger.Warn("TrackUserActivity failed:", err)
		}
	}()

	// Save action items and track activity
	var tasks []types.Task
	if len(structuredResp.ActionItems) > 0 {
		for _, item := range structuredResp.ActionItems {
			tasks = append(tasks, types.Task{
				Title:       item.Title,
				Description: item.Description,
				Status:      "pending",
				SessionID:   &sessionID,
				MessageID:   &messageId, // Associate with the user message
				AISuggested: true,
				CreatedAt:   time.Now(),
			})
		}
		if err := supabase.SaveTasks(supabaseClient, userId, tasks); err != nil {
			config.Logger.Warn("Failed to save AI-suggested tasks:", err)
		} else {
			// Track task creation activity
			go func() {
				if err := supabase.TrackUserActivity(supabaseClient, userId, sessionID, "tasks_created", fmt.Sprintf("Created %d AI-suggested tasks", len(tasks)), map[string]interface{}{
					"task_count":   len(tasks),
					"ai_suggested": true,
				}); err != nil {
					config.Logger.Warn("Failed to track user activity ", err)
				}

				if err := supabase.IncrementSessionCounter(supabaseClient, sessionID, "task_created"); err != nil {
					config.Logger.Warn("Failed to increment task_created session counter:", err)
				}
			}()
		}
	}

	// After getting smartContext, add validation data:
	keyTasks := smartContext.KeyTasks // You already have this in smartContext

	// Replace your task deletion section with:
	if len(structuredResp.DeleteTasks) > 0 {
		go func() {
			deletedCount := 0
			for _, taskID := range structuredResp.DeleteTasks {
				// Add validation
				if taskID == "" {
					config.Logger.Warn("Empty task ID in delete request")
					continue
				}

				// Verify task belongs to user by checking if it's in keyTasks
				found := false
				for _, task := range keyTasks {
					if task.ID == taskID {
						found = true
						break
					}
				}

				if !found {
					config.Logger.Warn("Attempted to delete non-existent or unauthorized task:", taskID)
					continue
				}

				if err := supabase.DeleteTask(supabaseClient, taskID, userId); err != nil {
					config.Logger.Warn("Failed to delete assistant-suggested task:", taskID, "error:", err)
				} else {
					deletedCount++
					config.Logger.Info("AI successfully deleted task:", taskID)
				}
			}

			if deletedCount > 0 {
				_ = supabase.TrackUserActivity(supabaseClient, userId, sessionID, "tasks_deleted",
					fmt.Sprintf("Assistant deleted %d tasks", deletedCount), map[string]interface{}{
						"deleted_count": deletedCount,
						"task_ids":      structuredResp.DeleteTasks,
					})
			}
		}()
	}

	// Replace your task update section with:
	if len(structuredResp.UpdateTasks) > 0 {
		go func() {
			updatedCount := 0
			for _, update := range structuredResp.UpdateTasks {
				// Add validation
				if update.ID == "" {
					config.Logger.Warn("Empty task ID in update request")
					continue
				}

				// Verify task belongs to user
				found := false
				for _, task := range keyTasks {
					if task.ID == update.ID {
						found = true
						break
					}
				}

				if !found {
					config.Logger.Warn("Attempted to update non-existent or unauthorized task:", update.ID)
					continue
				}

				payload := map[string]interface{}{}
				hasUpdates := false

				if update.Title != "" {
					payload["title"] = update.Title
					hasUpdates = true
				}
				if update.Description != "" {
					payload["description"] = update.Description
					hasUpdates = true
				}
				if update.Status != "" {
					// Validate status
					validStatuses := []string{"pending", "completed", "cancelled"}
					isValid := false
					for _, status := range validStatuses {
						if update.Status == status {
							isValid = true
							break
						}
					}
					if !isValid {
						config.Logger.Warn("Invalid status for task update:", update.Status)
						continue
					}
					payload["status"] = update.Status
					hasUpdates = true
				}

				// Handle DueDate updates - including clearing the date
				// Check if DueDate field was explicitly set in the update request
				if update.DueDate != nil {
					if !update.DueDate.IsZero() {
						payload["due_date"] = update.DueDate
					} else {
						// Clear the due date by setting it to null
						payload["due_date"] = nil
					}
					hasUpdates = true
				}

				// Handle Decision updates if applicable
				if update.Decision != "" {
					payload["decision"] = update.Decision
					hasUpdates = true
				}

				// Note: We need to check the UpdateTasks struct definition to handle
				// FollowUpDueAt and FollowedUp properly without nil pointer errors

				if hasUpdates {
					if updatedTask, err := supabase.UpdateTask(supabaseClient, update.ID, userId, payload); err != nil {
						config.Logger.Warn("Failed to update assistant-suggested task:", update.ID, "error:", err)
					} else {
						updatedCount++
						config.Logger.Info("AI successfully updated task:", update.ID, "changes:", payload)
						config.Logger.Info("Updated task details:", updatedTask.Title, updatedTask.Status)
					}
				} else {
					config.Logger.Warn("No valid fields to update for task:", update.ID)
				}
			}

			if updatedCount > 0 {
				_ = supabase.TrackUserActivity(supabaseClient, userId, sessionID, "tasks_updated",
					fmt.Sprintf("Assistant updated %d tasks", updatedCount), map[string]interface{}{
						"updated_count": updatedCount,
						"updates":       structuredResp.UpdateTasks,
					})
			}
		}()
	}

	// Update session metrics asynchronously
	go func() {
		if err := supabase.IncrementSessionCounter(supabaseClient, sessionID, "message"); err != nil {
			config.Logger.Warn("Failed to incemment session counter:", err)
		}
		if err := supabase.UpdateSessionSummaryIfNeeded(supabaseClient, sessionID, userId); err != nil {
			config.Logger.Warn("Failed to update session summary:", err)
		}
	}()

	// Send response
	resp := types.ChatResponse{
		Success:     true,
		UserMessage: req.Message,
		AIResponse:  structuredResp.Response,
		ActionItems: tasks,
		SessionID:   sessionID,
	}

	writeJSON(w, http.StatusOK, resp)
}

func GetMessagesHandler(w http.ResponseWriter, r *http.Request) {
	sessionID := r.URL.Query().Get("session_id")
	if sessionID == "" {
		writeError(w, "Missing session_id", http.StatusBadRequest)
		return
	}
	if _, err := uuid.Parse(sessionID); err != nil {
		writeError(w, "Invalid session_id", http.StatusBadRequest)
		return
	}

	// Auth check
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		writeError(w, "Missing Authorization header", http.StatusUnauthorized)
		return
	}
	// Create Supabase client
	client, userID, err := supabase.SupabaseClientFromRequest(r)
	if err != nil {
		writeError(w, "Failed to create Supabase client", http.StatusInternalServerError)
		return
	}

	messages, err := supabase.GetMessages(client, sessionID, userID)
	if err != nil {
		config.Logger.Error("Failed to fetch messages:", err)
		writeError(w, "Could not fetch messages", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, types.GetMessagesResponse{
		Success:  true,
		Messages: messages,
	})
}
