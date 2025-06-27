package main

import (
	"clementus360/ai-helper/config"
	"clementus360/ai-helper/handlers"
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
	mux.HandleFunc("/tasks/decline", handlers.DeclineTaskHandler)
	mux.HandleFunc("/tasks/accept", handlers.AcceptTaskHandler)
	// mux.HandleFunc("GET /goals", goalsHandler)
	// mux.HandleFunc("POST /goals", addGoalHandler)
	// mux.HandleFunc("POST /tasks", addTaskHandler)
	// mux.HandleFunc("GET /tasks/today", tasksHandler)
	// mux.HandleFunc("UPDATE /tasks/{id}/status", updateTaskHandler)

	log.Println("Server is running on port 8080")
	log.Fatal(http.ListenAndServe(":8080", mux))
}
