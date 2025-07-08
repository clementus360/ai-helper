package routes

import (
	"clementus360/ai-helper/handlers"
	"net/http"
)

// RegisterSessionRoutes registers all session-related routes
func RegisterSessionRoutes(mux *http.ServeMux) {
	// Basic session operations
	mux.HandleFunc("GET /sessions", handlers.GetSessionsHandler)
	mux.HandleFunc("PATCH /sessions/update", handlers.UpdateSessionHandler)

	// Session deletion operations
	mux.HandleFunc("DELETE /sessions", handlers.DeleteSessionHandler)
	mux.HandleFunc("POST /sessions/restore", handlers.RestoreSessionHandler)
	mux.HandleFunc("GET /sessions/deleted", handlers.GetDeletedSessionsHandler)
	mux.HandleFunc("DELETE /sessions/permanent", handlers.HardDeleteSessionHandler)
}
