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
	mux.HandleFunc("POST /tasks/create", handlers.CreateTaskHandler)
	mux.HandleFunc("PATCH /tasks/update", handlers.UpdateTaskHandler)
	mux.HandleFunc("DELETE /tasks/delete", handlers.DeleteTaskHandler)
	mux.HandleFunc("GET /tasks", handlers.GetTasksHandler)
	// mux.HandleFunc("GET /goals", goalsHandler)
	// mux.HandleFunc("POST /goals", addGoalHandler)
	// mux.HandleFunc("GET /tasks/today", tasksHandler)
	// mux.HandleFunc("UPDATE /tasks/{id}/status", updateTaskHandler)

	log.Println("Server is running on port 8080")
	log.Fatal(http.ListenAndServe(":8080", mux))
}
