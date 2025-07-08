package routes

import (
	"clementus360/ai-helper/handlers"
	"net/http"
)

// RegisterChatRoutes registers all chat-related routes
func RegisterChatRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /chat", handlers.ChatHandler)
	mux.HandleFunc("GET /chat", handlers.GetMessagesHandler)
}
