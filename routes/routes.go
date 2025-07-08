package routes

import "net/http"

// RegisterAllRoutes registers all application routes
func RegisterAllRoutes(mux *http.ServeMux) {
	RegisterChatRoutes(mux)
	RegisterTaskRoutes(mux)
	RegisterSessionRoutes(mux)
}

// Alternative approach - if you prefer a single registration function
func RegisterRoutes(mux *http.ServeMux) {
	RegisterAllRoutes(mux)
}
