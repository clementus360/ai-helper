package main

import (
	"clementus360/ai-helper/config"
	"clementus360/ai-helper/middleware"
	"clementus360/ai-helper/routes"
	"clementus360/ai-helper/supabase"
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	// Initialize configuration and dependencies
	if err := initializeApp(); err != nil {
		log.Fatalf("Failed to initialize application: %v", err)
	}

	// Setup server
	server := &http.Server{
		Addr:         ":8080",
		Handler:      setupRoutes(),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server with graceful shutdown
	startServerWithGracefulShutdown(server)
}

// initializeApp initializes all application dependencies
func initializeApp() error {
	config.LoadEnv()
	config.InitLogger()

	supabase.Init()

	config.Logger.Info("Application initialized successfully")
	return nil
}

// setupRoutes configures all application routes
func setupRoutes() http.Handler {
	mux := http.NewServeMux()

	// Register all route groups
	routes.RegisterChatRoutes(mux)
	routes.RegisterTaskRoutes(mux)
	routes.RegisterSessionRoutes(mux)

	// Apply middleware
	handler := middleware.CORSMiddleware(mux)
	// handler = middleware.LoggingMiddleware(handler)

	return handler
}

// startServerWithGracefulShutdown starts the server with graceful shutdown support
func startServerWithGracefulShutdown(server *http.Server) {
	// Channel to listen for interrupt signals
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	// Start server in a goroutine
	go func() {
		config.Logger.Info("Server starting on port 8080")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			config.Logger.Error("Server failed to start", "error", err)
			os.Exit(1)
		}
	}()

	// Wait for interrupt signal
	<-stop
	config.Logger.Info("Shutting down server...")

	// Create a context with timeout for graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Attempt graceful shutdown
	if err := server.Shutdown(ctx); err != nil {
		config.Logger.Error("Server forced to shutdown", "error", err)
		os.Exit(1)
	}

	config.Logger.Info("Server gracefully stopped")
}
