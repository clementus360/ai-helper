package main

import (
	"clementus360/ai-helper/config"
	"clementus360/ai-helper/handlers"
	"clementus360/ai-helper/middleware"
	"clementus360/ai-helper/supabase"
	"log"
	"net/http"
)

func main() {

	config.LoadEnv()
	config.InitLogger()
	supabase.Init()

	mux := http.NewServeMux()

	mux.HandleFunc("POST /chat", handlers.ChatHandler)
	mux.HandleFunc("GET /chat", handlers.GetMessagesHandler)
	mux.HandleFunc("POST /tasks/create", handlers.CreateTaskHandler)
	mux.HandleFunc("PATCH /tasks/update", handlers.UpdateTaskHandler)
	mux.HandleFunc("DELETE /tasks/delete", handlers.DeleteTaskHandler)
	mux.HandleFunc("GET /tasks", handlers.GetTasksHandler)
	mux.HandleFunc("GET /task", handlers.GetSingleTaskHandler)
	mux.HandleFunc("GET /sessions", handlers.GetSessionsHandler)
	mux.HandleFunc("PATCH /sessions/update", handlers.UpdateSessionHandler)

	// Wrap the mux with CORS middleware
	corsHandler := middleware.CORSMiddleware(mux)

	log.Println("Server is running on port 8080")
	log.Fatal(http.ListenAndServe(":8080", corsHandler))
}
