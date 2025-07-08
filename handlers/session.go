package handlers

import (
	"clementus360/ai-helper/config"
	"clementus360/ai-helper/supabase"
	"clementus360/ai-helper/types"
	"encoding/json"
	"net/http"
	"strings"
)

func GetSessionsHandler(w http.ResponseWriter, r *http.Request) {
	supabaseClient, userID, err := supabase.SupabaseClientFromRequest(r)
	if err != nil {
		config.Logger.Error("Failed to create Supabase client:", err)
		writeError(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	sessions, err := supabase.GetSessions(supabaseClient, userID)
	if err != nil {
		config.Logger.Error("Failed to fetch sessions:", err)
		writeError(w, "Failed to fetch sessions", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, types.GetSessionsResponse{
		Success:  true,
		Sessions: sessions,
	})
}

func UpdateSessionHandler(w http.ResponseWriter, r *http.Request) {

	sessionID := r.URL.Query().Get("id")
	if sessionID == "" {
		config.Logger.Warn("Missing session ID in request")
		writeError(w, "Missing session ID", http.StatusBadRequest)
		return
	}

	var body struct {
		Title string `json:"title"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || strings.TrimSpace(body.Title) == "" {
		config.Logger.Warn("Invalid or missing title in request body:", err)
		writeError(w, "Invalid or missing title", http.StatusBadRequest)
		return
	}

	client, userID, err := supabase.SupabaseClientFromRequest(r)
	if err != nil {
		config.Logger.Error("Failed to create Supabase client:", err)
		writeError(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	updated, err := supabase.UpdateSessionTitle(client, sessionID, userID, body.Title)
	if err != nil {
		config.Logger.Error("Failed to update session:", err)
		writeError(w, "Failed to update session", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, types.SessionResponse{
		Success: true,
		Session: updated,
	})
}

// DeleteSessionHandler handles soft deletion of a session
func DeleteSessionHandler(w http.ResponseWriter, r *http.Request) {
	sessionID := r.URL.Query().Get("id")
	if sessionID == "" {
		config.Logger.Warn("Missing session ID in request")
		writeError(w, "Missing session ID", http.StatusBadRequest)
		return
	}

	supabaseClient, userID, err := supabase.SupabaseClientFromRequest(r)
	if err != nil {
		config.Logger.Error("Failed to create Supabase client:", err)
		writeError(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	err = supabase.DeleteSession(supabaseClient, sessionID, userID)
	if err != nil {
		config.Logger.Error("Failed to delete session:", err)
		writeError(w, "Failed to delete session", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, types.BaseResponse{
		Success: true,
		Message: "Session deleted successfully",
	})
}

// RestoreSessionHandler handles restoration of a soft-deleted session
func RestoreSessionHandler(w http.ResponseWriter, r *http.Request) {
	sessionID := r.URL.Query().Get("id")
	if sessionID == "" {
		config.Logger.Warn("Missing session ID in request")
		writeError(w, "Missing session ID", http.StatusBadRequest)
		return
	}

	supabaseClient, userID, err := supabase.SupabaseClientFromRequest(r)
	if err != nil {
		config.Logger.Error("Failed to create Supabase client:", err)
		writeError(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	err = supabase.RestoreSession(supabaseClient, sessionID, userID)
	if err != nil {
		config.Logger.Error("Failed to restore session:", err)
		writeError(w, "Failed to restore session", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, types.BaseResponse{
		Success: true,
		Message: "Session restored successfully",
	})
}

// GetDeletedSessionsHandler returns soft-deleted sessions for a user
func GetDeletedSessionsHandler(w http.ResponseWriter, r *http.Request) {
	supabaseClient, userID, err := supabase.SupabaseClientFromRequest(r)
	if err != nil {
		config.Logger.Error("Failed to create Supabase client:", err)
		writeError(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	sessions, err := supabase.GetDeletedSessions(supabaseClient, userID)
	if err != nil {
		config.Logger.Error("Failed to fetch deleted sessions:", err)
		writeError(w, "Failed to fetch deleted sessions", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, types.GetSessionsResponse{
		Success:  true,
		Sessions: sessions,
	})
}

// HardDeleteSessionHandler permanently deletes a session (admin only or explicit user request)
func HardDeleteSessionHandler(w http.ResponseWriter, r *http.Request) {
	sessionID := r.URL.Query().Get("id")
	if sessionID == "" {
		config.Logger.Warn("Missing session ID in request")
		writeError(w, "Missing session ID", http.StatusBadRequest)
		return
	}

	// Parse request body to check for confirmation
	var body struct {
		Confirm bool `json:"confirm"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || !body.Confirm {
		config.Logger.Warn("Hard delete requires explicit confirmation")
		writeError(w, "Hard delete requires explicit confirmation", http.StatusBadRequest)
		return
	}

	supabaseClient, userID, err := supabase.SupabaseClientFromRequest(r)
	if err != nil {
		config.Logger.Error("Failed to create Supabase client:", err)
		writeError(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	err = supabase.HardDeleteSession(supabaseClient, sessionID, userID)
	if err != nil {
		config.Logger.Error("Failed to permanently delete session:", err)
		writeError(w, "Failed to permanently delete session", http.StatusInternalServerError)
		return
	}

	// Log the hard delete action for audit purposes
	config.Logger.Info("Session permanently deleted", "sessionID", sessionID, "userID", userID)

	writeJSON(w, http.StatusOK, types.BaseResponse{
		Success: true,
		Message: "Session permanently deleted",
	})
}
