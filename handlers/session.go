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
