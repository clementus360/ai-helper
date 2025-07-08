package routes

import (
	"clementus360/ai-helper/handlers"
	"net/http"
)

// RegisterTaskRoutes registers all task-related routes
func RegisterTaskRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /tasks/create", handlers.CreateTaskHandler)
	mux.HandleFunc("PATCH /tasks/update", handlers.UpdateTaskHandler)
	mux.HandleFunc("DELETE /tasks/delete", handlers.DeleteTaskHandler)
	mux.HandleFunc("GET /tasks", handlers.GetTasksHandler)
	mux.HandleFunc("GET /task", handlers.GetSingleTaskHandler)
}
