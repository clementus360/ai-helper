// middleware/middleware.go
package middleware

import (
	"net/http"
)

// CORSMiddleware handles CORS headers
func CORSMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Set CORS headers
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With")
		w.Header().Set("Access-Control-Max-Age", "86400")

		// Handle preflight requests
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// LoggingMiddleware logs HTTP requests
// func LoggingMiddleware(next http.Handler) http.Handler {
// 	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
// 		start := time.Now()

// 		// Create a wrapper to capture response status
// 		wrapper := &responseWrapper{
// 			ResponseWriter: w,
// 			statusCode:     http.StatusOK,
// 		}

// 		// Call the next handler
// 		next.ServeHTTP(wrapper, r)

// 		// Log the request
// 		duration := time.Since(start)
// 		config.Logger.Info("HTTP Request",
// 			"method", r.Method,
// 			"path", r.URL.Path,
// 			"status", wrapper.statusCode,
// 			"duration", duration.String(),
// 			"remote_addr", r.RemoteAddr,
// 			"user_agent", r.UserAgent(),
// 		)
// 	})
// }

// responseWrapper wraps http.ResponseWriter to capture status code
type responseWrapper struct {
	http.ResponseWriter
	statusCode int
}

func (w *responseWrapper) WriteHeader(statusCode int) {
	w.statusCode = statusCode
	w.ResponseWriter.WriteHeader(statusCode)
}

// AuthMiddleware can be added later for authentication
func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// TODO: Implement authentication logic
		// For now, just pass through
		next.ServeHTTP(w, r)
	})
}

// RateLimitMiddleware can be added later for rate limiting
func RateLimitMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// TODO: Implement rate limiting logic
		// For now, just pass through
		next.ServeHTTP(w, r)
	})
}

// Chain allows chaining multiple middleware functions
func Chain(middlewares ...func(http.Handler) http.Handler) func(http.Handler) http.Handler {
	return func(final http.Handler) http.Handler {
		for i := len(middlewares) - 1; i >= 0; i-- {
			final = middlewares[i](final)
		}
		return final
	}
}
