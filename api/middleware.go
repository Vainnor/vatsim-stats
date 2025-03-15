package api

import (
	"net/http"
	"sync"
	"time"
)

type RateLimiter struct {
	requests map[string]*ClientRequests
	mu       sync.RWMutex
}

type ClientRequests struct {
	count    int
	lastSeen time.Time
}

const (
	maxRequests    = 100             // Maximum requests per window
	windowDuration = time.Minute * 5 // Window duration
)

var limiter = &RateLimiter{
	requests: make(map[string]*ClientRequests),
}

func RateLimit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check for API key in Authorization header
		apiKey := r.Header.Get("Authorization")
		if apiKey != "" && ValidateAPIKey(apiKey) {
			// API key is valid, bypass rate limiting
			next.ServeHTTP(w, r)
			return
		}

		// Get client IP
		clientIP := r.RemoteAddr

		limiter.mu.Lock()
		defer limiter.mu.Unlock()

		// Clean up old entries
		now := time.Now()
		for ip, req := range limiter.requests {
			if now.Sub(req.lastSeen) > windowDuration {
				delete(limiter.requests, ip)
			}
		}

		// Get or create client requests
		client, exists := limiter.requests[clientIP]
		if !exists {
			client = &ClientRequests{
				count:    0,
				lastSeen: now,
			}
			limiter.requests[clientIP] = client
		}

		// Check if window has expired
		if now.Sub(client.lastSeen) > windowDuration {
			client.count = 0
			client.lastSeen = now
		}

		// Check rate limit
		if client.count >= maxRequests {
			w.Header().Set("X-RateLimit-Limit", "100")
			w.Header().Set("X-RateLimit-Remaining", "0")
			w.Header().Set("X-RateLimit-Reset", time.Unix(client.lastSeen.Add(windowDuration).Unix(), 0).Format(time.RFC3339))
			http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
			return
		}

		// Increment request count
		client.count++
		client.lastSeen = now

		// Set rate limit headers
		w.Header().Set("X-RateLimit-Limit", "100")
		w.Header().Set("X-RateLimit-Remaining", string(maxRequests-client.count))
		w.Header().Set("X-RateLimit-Reset", time.Unix(client.lastSeen.Add(windowDuration).Unix(), 0).Format(time.RFC3339))

		next.ServeHTTP(w, r)
	})
}
