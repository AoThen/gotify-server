package auth

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

type limiterEntry struct {
	limiter    *rate.Limiter
	lastAccess time.Time
}

type RateLimiter struct {
	limiters   map[string]*limiterEntry
	mu         sync.RWMutex
	rate       rate.Limit
	burst      int
	maxEntries int
	stopCh     chan struct{}
}

func NewRateLimiter(r rate.Limit, burst int) *RateLimiter {
	rl := &RateLimiter{
		limiters:   make(map[string]*limiterEntry),
		rate:       r,
		burst:      burst,
		maxEntries: 10000,
		stopCh:     make(chan struct{}),
	}
	go rl.cleanupLoop()
	return rl
}

func (rl *RateLimiter) cleanupLoop() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			rl.cleanup()
		case <-rl.stopCh:
			return
		}
	}
}

func (rl *RateLimiter) cleanup() {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	expiry := now.Add(-1 * time.Hour)

	if len(rl.limiters) <= rl.maxEntries {
		return
	}

	var toDelete []string
	for ip, entry := range rl.limiters {
		if entry.lastAccess.Before(expiry) {
			toDelete = append(toDelete, ip)
		}
	}

	for _, ip := range toDelete {
		delete(rl.limiters, ip)
	}

	if len(rl.limiters) > rl.maxEntries {
		count := 0
		for ip := range rl.limiters {
			delete(rl.limiters, ip)
			count++
			if len(rl.limiters) <= rl.maxEntries/2 {
				break
			}
		}
	}
}

func (rl *RateLimiter) Close() {
	close(rl.stopCh)
}

func (rl *RateLimiter) getLimiter(ip string) *rate.Limiter {
	rl.mu.Lock()

	entry, exists := rl.limiters[ip]
	if !exists {
		entry = &limiterEntry{
			limiter:    rate.NewLimiter(rl.rate, rl.burst),
			lastAccess: time.Now(),
		}
		rl.limiters[ip] = entry
		rl.mu.Unlock()
		return entry.limiter
	}

	entry.lastAccess = time.Now()
	rl.mu.Unlock()
	return entry.limiter
}

func (rl *RateLimiter) RateLimit() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		ip := ctx.ClientIP()
		limiter := rl.getLimiter(ip)

		if !limiter.Allow() {
			ctx.JSON(http.StatusTooManyRequests, gin.H{
				"error":            "Too Many Requests",
				"errorCode":        429,
				"errorDescription": "Rate limit exceeded. Please try again later.",
			})
			ctx.Abort()
			return
		}

		ctx.Next()
	}
}
