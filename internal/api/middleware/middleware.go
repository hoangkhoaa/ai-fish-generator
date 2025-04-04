package middleware

import (
	"log"
	"net/http"
	"time"
)

// Middleware type for HTTP handlers
type Middleware func(http.HandlerFunc) http.HandlerFunc

// ApplyMiddleware applies multiple middleware to a handler
func ApplyMiddleware(h http.HandlerFunc, middleware ...Middleware) http.HandlerFunc {
	for _, m := range middleware {
		h = m(h)
	}
	return h
}

// Logging middleware logs request details
func Logging() Middleware {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			// Create a wrapper for the response writer
			lw := &loggingResponseWriter{
				ResponseWriter: w,
				statusCode:     http.StatusOK, // Default status code
			}

			// Call the next handler
			next(lw, r)

			// Log the request details
			log.Printf("| %s | %s | %s | %d | %s",
				r.Method,
				r.URL.Path,
				time.Since(start),
				lw.statusCode,
				r.RemoteAddr,
			)
		}
	}
}

// CORS middleware adds CORS headers to responses
func CORS() Middleware {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			// Add CORS headers
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

			// Handle preflight requests
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusOK)
				return
			}

			// Call the next handler
			next(w, r)
		}
	}
}

// RateLimiting middleware implements a simple rate limiter
func RateLimiting(requestsPerMinute int) Middleware {
	// Create a map to store client request counts
	clients := make(map[string]int)
	lastCleanup := time.Now()

	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			// Clean up the map every minute
			if time.Since(lastCleanup) > time.Minute {
				clients = make(map[string]int)
				lastCleanup = time.Now()
			}

			// Get client IP
			clientIP := r.RemoteAddr

			// Check if client has exceeded rate limit
			if clients[clientIP] >= requestsPerMinute {
				http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
				return
			}

			// Increment request count
			clients[clientIP]++

			// Call the next handler
			next(w, r)
		}
	}
}

// loggingResponseWriter is a wrapper around http.ResponseWriter
type loggingResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

// WriteHeader captures the status code
func (lw *loggingResponseWriter) WriteHeader(code int) {
	lw.statusCode = code
	lw.ResponseWriter.WriteHeader(code)
}

// Authentication middleware checks if the request is authenticated
func Authentication() Middleware {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			// Get API key from request
			apiKey := r.Header.Get("X-API-Key")

			// Check if API key is valid (simplified for demo)
			if apiKey == "" {
				http.Error(w, "Unauthorized: API key required", http.StatusUnauthorized)
				return
			}

			// In a real application, you would validate the API key against a database
			// For demo purposes, we'll accept any non-empty key

			// Call the next handler
			next(w, r)
		}
	}
}
