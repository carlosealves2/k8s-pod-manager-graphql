package middleware

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// InMemoryRateLimiter implements RateLimiter using in-memory storage
type InMemoryRateLimiter struct {
	requests  map[string]int
	resetTime map[string]time.Time
	limit     int
	window    time.Duration
	mu        sync.RWMutex

	// Cleanup mechanism
	cleanupInterval time.Duration
	stopCleanup     chan struct{}
	cleanupOnce     sync.Once
}

// InMemoryRateLimiterConfig configures the in-memory rate limiter
type InMemoryRateLimiterConfig struct {
	Limit           int           // Maximum requests per window
	Window          time.Duration // Time window (default: 1 minute)
	CleanupInterval time.Duration // How often to clean up old entries (default: 5 minutes)
}

// NewInMemoryRateLimiter creates a new in-memory rate limiter
func NewInMemoryRateLimiter(config InMemoryRateLimiterConfig) RateLimiter {
	if config.Window == 0 {
		config.Window = time.Minute
	}
	if config.CleanupInterval == 0 {
		config.CleanupInterval = 5 * time.Minute
	}

	rl := &InMemoryRateLimiter{
		requests:        make(map[string]int),
		resetTime:       make(map[string]time.Time),
		limit:           config.Limit,
		window:          config.Window,
		cleanupInterval: config.CleanupInterval,
		stopCleanup:     make(chan struct{}),
	}

	// Start cleanup goroutine
	go rl.startCleanup()

	return rl
}

// Allow checks if a request is allowed for the given key
func (r *InMemoryRateLimiter) Allow(ctx context.Context, key string) (*RateLimitResult, error) {
	now := time.Now()

	r.mu.Lock()
	defer r.mu.Unlock()

	// Check if we need to reset the window for this key
	if reset, exists := r.resetTime[key]; !exists || now.After(reset) {
		r.requests[key] = 0
		r.resetTime[key] = now.Add(r.window)
	}

	// Increment request count
	r.requests[key]++
	currentCount := r.requests[key]
	resetTime := r.resetTime[key]

	result := &RateLimitResult{
		Allowed:   currentCount <= r.limit,
		Limit:     r.limit,
		Remaining: max(0, r.limit-currentCount),
		ResetTime: resetTime,
	}

	if !result.Allowed {
		result.RetryAfter = time.Until(resetTime)
	}

	return result, nil
}

// Reset resets the rate limit for a given key
func (r *InMemoryRateLimiter) Reset(ctx context.Context, key string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	delete(r.requests, key)
	delete(r.resetTime, key)

	return nil
}

// startCleanup runs a cleanup goroutine to remove expired entries
func (r *InMemoryRateLimiter) startCleanup() {
	ticker := time.NewTicker(r.cleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			r.cleanup()
		case <-r.stopCleanup:
			return
		}
	}
}

// cleanup removes expired entries to prevent memory leaks
func (r *InMemoryRateLimiter) cleanup() {
	now := time.Now()

	r.mu.Lock()
	defer r.mu.Unlock()

	for key, resetTime := range r.resetTime {
		if now.After(resetTime) {
			delete(r.requests, key)
			delete(r.resetTime, key)
		}
	}
}

// Stop stops the cleanup goroutine
func (r *InMemoryRateLimiter) Stop() {
	r.cleanupOnce.Do(func() {
		close(r.stopCleanup)
	})
}

// RateLimitMiddleware handles rate limiting for HTTP requests
type RateLimitMiddleware struct {
	limiter   RateLimiter
	keyFunc   func(ctx HTTPContext) string
	skipPaths map[string]bool
	logger    Logger
}

// RateLimitConfig configures the rate limit middleware
type RateLimitConfig struct {
	Limiter   RateLimiter
	KeyFunc   func(ctx HTTPContext) string // Function to extract rate limit key (default: IP address)
	SkipPaths []string                     // Paths to skip rate limiting
	Logger    Logger                       // Optional logger for rate limit events
}

// NewRateLimitMiddleware creates a new rate limiting middleware
func NewRateLimitMiddleware(config RateLimitConfig) Middleware {
	if config.KeyFunc == nil {
		config.KeyFunc = func(ctx HTTPContext) string {
			return ctx.Request().IP()
		}
	}

	skipPaths := make(map[string]bool)
	for _, path := range config.SkipPaths {
		skipPaths[path] = true
	}

	return &RateLimitMiddleware{
		limiter:   config.Limiter,
		keyFunc:   config.KeyFunc,
		skipPaths: skipPaths,
		logger:    config.Logger,
	}
}

// Handle implements the Middleware interface
func (m *RateLimitMiddleware) Handle(ctx HTTPContext) error {
	req := ctx.Request()

	// Skip rate limiting for certain paths
	if m.skipPaths[req.Path()] {
		return ctx.Next()
	}

	// Extract rate limit key
	key := m.keyFunc(ctx)

	// Check rate limit
	result, err := m.limiter.Allow(context.Background(), key)
	if err != nil {
		if m.logger != nil {
			m.logger.Error(context.Background(), "Rate limiter error", err,
				Field{Key: "key", Value: key},
				Field{Key: "path", Value: req.Path()},
			)
		}
		// Continue processing on rate limiter error
		return ctx.Next()
	}

	// Set rate limit headers
	resp := ctx.Response()
	resp.SetHeader("X-RateLimit-Limit", fmt.Sprintf("%d", result.Limit))
	resp.SetHeader("X-RateLimit-Remaining", fmt.Sprintf("%d", result.Remaining))
	resp.SetHeader("X-RateLimit-Reset", fmt.Sprintf("%d", result.ResetTime.Unix()))

	// Check if request is allowed
	if !result.Allowed {
		resp.SetHeader("Retry-After", fmt.Sprintf("%.0f", result.RetryAfter.Seconds()))
		resp.SetStatus(429) // Too Many Requests

		if m.logger != nil {
			m.logger.Warn(context.Background(), "Rate limit exceeded",
				Field{Key: "key", Value: key},
				Field{Key: "path", Value: req.Path()},
				Field{Key: "retry_after", Value: result.RetryAfter.Seconds()},
			)
		}

		return resp.JSON(map[string]interface{}{
			"error":       "Rate limit exceeded",
			"retry_after": result.ResetTime.Unix(),
			"limit":       result.Limit,
		})
	}

	return ctx.Next()
}

// Helper function since Go doesn't have a built-in max for ints
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}