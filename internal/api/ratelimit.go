package api

import (
	"sync"
	"time"
)

// RateLimiter implements a token bucket rate limiter per client IP
type RateLimiter struct {
	mu      sync.Mutex
	clients map[string]*clientLimit
	limit   int
	window  time.Duration
}

type clientLimit struct {
	count     int
	resetTime time.Time
}

// NewRateLimiter creates a new rate limiter with the specified limit per minute
func NewRateLimiter(limitPerMinute int) *RateLimiter {
	if limitPerMinute < 1 {
		limitPerMinute = 100
	}
	rl := &RateLimiter{
		clients: make(map[string]*clientLimit),
		limit:   limitPerMinute,
		window:  time.Minute,
	}
	go rl.cleanup()
	return rl
}

// Allow checks if a request from the given IP is allowed
func (rl *RateLimiter) Allow(ip string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	cl, exists := rl.clients[ip]

	if !exists || now.After(cl.resetTime) {
		rl.clients[ip] = &clientLimit{
			count:     1,
			resetTime: now.Add(rl.window),
		}
		return true
	}

	if cl.count >= rl.limit {
		return false
	}

	cl.count++
	return true
}

// cleanup periodically removes expired client entries
func (rl *RateLimiter) cleanup() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		rl.mu.Lock()
		now := time.Now()
		for ip, cl := range rl.clients {
			if now.After(cl.resetTime) {
				delete(rl.clients, ip)
			}
		}
		rl.mu.Unlock()
	}
}
