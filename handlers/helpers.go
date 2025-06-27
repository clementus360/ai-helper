package handlers

import (
	"clementus360/ai-helper/types"
	"encoding/json"
	"net/http"
)

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(payload)
}

func writeError(w http.ResponseWriter, message string, status int) {
	resp := types.ChatResponse{
		Success:      false,
		ErrorMessage: message,
	}
	writeJSON(w, status, resp)

}
