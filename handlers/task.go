package handlers

import (
	supabaselocal "clementus360/ai-helper/supabase"
	"net/http"

	"github.com/google/uuid"
)

func DeclineTaskHandler(w http.ResponseWriter, r *http.Request) {
	taskID := r.URL.Query().Get("id")
	if taskID == "" {
		writeError(w, "Missing task ID", http.StatusBadRequest)
		return
	}
	if _, err := uuid.Parse(taskID); err != nil {
		writeError(w, "Invalid task ID", http.StatusBadRequest)
		return
	}

	updates := map[string]interface{}{
		"decision": "declined",
	}

	_, _, err := supabaselocal.Client.From("tasks").
		Update(updates, "", "").
		Eq("id", taskID).
		Execute()
	if err != nil {
		writeError(w, "Could not decline task", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"message": "Task declined",
	})
}

func AcceptTaskHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, "Only POST is allowed", http.StatusMethodNotAllowed)
		return
	}

	taskID := r.URL.Query().Get("id")
	if taskID == "" {
		writeError(w, "Missing task ID", http.StatusBadRequest)
		return
	}

	if _, err := uuid.Parse(taskID); err != nil {
		writeError(w, "Invalid task ID format", http.StatusBadRequest)
		return
	}

	updates := map[string]interface{}{
		"decision": "accepted",
	}

	_, _, err := supabaselocal.Client.From("tasks").
		Update(updates, "", "").
		Eq("id", taskID).
		Execute()

	if err != nil {
		writeError(w, "Could not accept task", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"message": "Task accepted",
	})
}
