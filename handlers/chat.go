package handlers

import (
	"clementus360/ai-helper/config"
	"clementus360/ai-helper/llm"
	"clementus360/ai-helper/supabase"
	"clementus360/ai-helper/types"
	"encoding/json"
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

	// Get session context (recent messages + summary)
	context, err := supabase.GetSessionContext(supabaseClient, sessionID, userId)
	if err != nil {
		config.Logger.Warn("Failed to get session context:", err)
		// Continue without context rather than failing
		context = types.SessionContext{}
	}

	// Save the user message to Supabase
	if err := supabase.SaveMessage(supabaseClient, userId, sessionID, "user", req.Message); err != nil {
		config.Logger.Error("Failed to save message:", err)
		writeError(w, "Could not save message", http.StatusInternalServerError)
		return
	}

	// Generate AI response with fallback
	structuredResp, err := llm.GeminiGenerateResponse(req.Message, context)
	if err != nil {
		config.Logger.Error("Failed to get AI response:", err)
		// Fallback response instead of hard failure
		structuredResp = llm.GeminiStructuredResponse{
			Response:    "I'm having trouble processing that right now. Could you rephrase what you're struggling with?",
			ActionItems: []llm.GeminiTaskItem{},
		}
	}

	// Save AI response message
	if err := supabase.SaveMessage(supabaseClient, userId, sessionID, "ai", structuredResp.Response); err != nil {
		config.Logger.Warn("Failed to save AI message:", err)
	}

	// ✅ Save action items if any
	var tasks []types.Task
	if len(structuredResp.ActionItems) > 0 {
		for _, item := range structuredResp.ActionItems {
			tasks = append(tasks, types.Task{
				Title:       item.Title,
				Description: item.Description,
				Status:      "pending",
				SessionID:   &sessionID,
				AISuggested: true,
				CreatedAt:   time.Now(),
			})
		}

		if err := supabase.SaveTasks(supabaseClient, userId, tasks); err != nil {
			config.Logger.Warn("Failed to save AI-suggested tasks:", err)
		}
	}

	// Check if we need to update session summary
	go func() {
		if err := supabase.UpdateSessionSummaryIfNeeded(supabaseClient, sessionID, userId); err != nil {
			config.Logger.Warn("Failed to update session summary:", err)
		}
	}()

	// Send structured response to frontend
	resp := types.ChatResponse{
		Success:     true,
		UserMessage: req.Message,
		AIResponse:  structuredResp.Response,
		ActionItems: tasks,
		SessionID:   sessionID, // Include session ID in response
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
