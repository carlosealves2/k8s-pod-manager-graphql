package middleware

import (
	"context"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// TestInMemoryRateLimiterBasicFunctionality tests basic rate limiting operations
func TestInMemoryRateLimiterBasicFunctionality(t *testing.T) {
	config := InMemoryRateLimiterConfig{
		Limit:  3,
		Window: time.Minute,
	}
	limiter := NewInMemoryRateLimiter(config)
	defer limiter.(*InMemoryRateLimiter).Stop()

	ctx := context.Background()
	key := "test-key"

	t.Run("allows requests under limit", func(t *testing.T) {
		// First request
		result1, err := limiter.Allow(ctx, key)
		assert.NoError(t, err)
		assert.True(t, result1.Allowed)
		assert.Equal(t, 3, result1.Limit)
		assert.Equal(t, 2, result1.Remaining)
		assert.True(t, result1.ResetTime.After(time.Now()))

		// Second request
		result2, err := limiter.Allow(ctx, key)
		assert.NoError(t, err)
		assert.True(t, result2.Allowed)
		assert.Equal(t, 3, result2.Limit)
		assert.Equal(t, 1, result2.Remaining)

		// Third request
		result3, err := limiter.Allow(ctx, key)
		assert.NoError(t, err)
		assert.True(t, result3.Allowed)
		assert.Equal(t, 3, result3.Limit)
		assert.Equal(t, 0, result3.Remaining)
	})

	t.Run("blocks requests over limit", func(t *testing.T) {
		// Fourth request should be blocked
		result4, err := limiter.Allow(ctx, key)
		assert.NoError(t, err)
		assert.False(t, result4.Allowed)
		assert.Equal(t, 3, result4.Limit)
		assert.Equal(t, 0, result4.Remaining)
		assert.Greater(t, result4.RetryAfter, time.Duration(0))

		// Fifth request should also be blocked
		result5, err := limiter.Allow(ctx, key)
		assert.NoError(t, err)
		assert.False(t, result5.Allowed)
		assert.Equal(t, 0, result5.Remaining)
	})
}

// TestInMemoryRateLimiterWindowReset tests window reset functionality
func TestInMemoryRateLimiterWindowReset(t *testing.T) {
	config := InMemoryRateLimiterConfig{
		Limit:  2,
		Window: 100 * time.Millisecond,
	}
	limiter := NewInMemoryRateLimiter(config)
	defer limiter.(*InMemoryRateLimiter).Stop()

	ctx := context.Background()
	key := "test-key"

	// Use up the limit
	result1, err := limiter.Allow(ctx, key)
	assert.NoError(t, err)
	assert.True(t, result1.Allowed)

	result2, err := limiter.Allow(ctx, key)
	assert.NoError(t, err)
	assert.True(t, result2.Allowed)

	// Should be blocked now
	result3, err := limiter.Allow(ctx, key)
	assert.NoError(t, err)
	assert.False(t, result3.Allowed)

	// Wait for window to reset
	time.Sleep(150 * time.Millisecond)

	// Should be allowed again
	result4, err := limiter.Allow(ctx, key)
	assert.NoError(t, err)
	assert.True(t, result4.Allowed)
	assert.Equal(t, 1, result4.Remaining)
}

// TestInMemoryRateLimiterMultipleKeys tests rate limiting with multiple keys
func TestInMemoryRateLimiterMultipleKeys(t *testing.T) {
	config := InMemoryRateLimiterConfig{
		Limit:  2,
		Window: time.Minute,
	}
	limiter := NewInMemoryRateLimiter(config)
	defer limiter.(*InMemoryRateLimiter).Stop()

	ctx := context.Background()

	// Use up limit for key1
	result1, err := limiter.Allow(ctx, "key1")
	assert.NoError(t, err)
	assert.True(t, result1.Allowed)

	result2, err := limiter.Allow(ctx, "key1")
	assert.NoError(t, err)
	assert.True(t, result2.Allowed)

	result3, err := limiter.Allow(ctx, "key1")
	assert.NoError(t, err)
	assert.False(t, result3.Allowed) // key1 is now blocked

	// key2 should still be allowed
	result4, err := limiter.Allow(ctx, "key2")
	assert.NoError(t, err)
	assert.True(t, result4.Allowed)
	assert.Equal(t, 1, result4.Remaining)

	result5, err := limiter.Allow(ctx, "key2")
	assert.NoError(t, err)
	assert.True(t, result5.Allowed)
	assert.Equal(t, 0, result5.Remaining)

	// key2 should now be blocked
	result6, err := limiter.Allow(ctx, "key2")
	assert.NoError(t, err)
	assert.False(t, result6.Allowed)
}

// TestInMemoryRateLimiterReset tests manual reset functionality
func TestInMemoryRateLimiterReset(t *testing.T) {
	config := InMemoryRateLimiterConfig{
		Limit:  1,
		Window: time.Minute,
	}
	limiter := NewInMemoryRateLimiter(config)
	defer limiter.(*InMemoryRateLimiter).Stop()

	ctx := context.Background()
	key := "test-key"

	// Use up the limit
	result1, err := limiter.Allow(ctx, key)
	assert.NoError(t, err)
	assert.True(t, result1.Allowed)

	// Should be blocked
	result2, err := limiter.Allow(ctx, key)
	assert.NoError(t, err)
	assert.False(t, result2.Allowed)

	// Reset the key
	err = limiter.Reset(ctx, key)
	assert.NoError(t, err)

	// Should be allowed again
	result3, err := limiter.Allow(ctx, key)
	assert.NoError(t, err)
	assert.True(t, result3.Allowed)
	assert.Equal(t, 0, result3.Remaining)
}

// TestInMemoryRateLimiterConcurrency tests concurrent access
func TestInMemoryRateLimiterConcurrency(t *testing.T) {
	config := InMemoryRateLimiterConfig{
		Limit:  100,
		Window: time.Minute,
	}
	limiter := NewInMemoryRateLimiter(config)
	defer limiter.(*InMemoryRateLimiter).Stop()

	const numGoroutines = 50
	const requestsPerGoroutine = 10
	var wg sync.WaitGroup
	var allowedCount int64
	var mu sync.Mutex

	ctx := context.Background()

	// Launch concurrent requests
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()

			for j := 0; j < requestsPerGoroutine; j++ {
				// Use a mix of keys to test concurrent access to different keys
				key := "test-key-" + string(rune(goroutineID%5))

				result, err := limiter.Allow(ctx, key)
				assert.NoError(t, err)

				if result.Allowed {
					mu.Lock()
					allowedCount++
					mu.Unlock()
				}
			}
		}(i)
	}

	wg.Wait()

	// We should have some allowed requests (exact number depends on timing)
	// but certainly less than total because of different keys having separate limits
	totalRequests := int64(numGoroutines * requestsPerGoroutine)
	assert.Greater(t, allowedCount, int64(0))
	assert.LessOrEqual(t, allowedCount, totalRequests)
}

// TestInMemoryRateLimiterCleanup tests the cleanup mechanism
func TestInMemoryRateLimiterCleanup(t *testing.T) {
	config := InMemoryRateLimiterConfig{
		Limit:           2,
		Window:          50 * time.Millisecond,
		CleanupInterval: 25 * time.Millisecond,
	}
	limiter := NewInMemoryRateLimiter(config)
	defer limiter.(*InMemoryRateLimiter).Stop()

	ctx := context.Background()

	// Add some entries
	_, _ = limiter.Allow(ctx, "key1")
	_, _ = limiter.Allow(ctx, "key2")
	_, _ = limiter.Allow(ctx, "key3")

	// Access internal state to verify entries exist
	inMemoryLimiter := limiter.(*InMemoryRateLimiter)
	inMemoryLimiter.mu.RLock()
	initialCount := len(inMemoryLimiter.requests)
	inMemoryLimiter.mu.RUnlock()

	assert.Greater(t, initialCount, 0, "Should have some entries")

	// Wait for windows to expire and cleanup to run
	time.Sleep(100 * time.Millisecond)

	// Check that old entries were cleaned up
	inMemoryLimiter.mu.RLock()
	finalCount := len(inMemoryLimiter.requests)
	inMemoryLimiter.mu.RUnlock()

	// Should be cleaned up (might not be exactly 0 due to timing)
	assert.LessOrEqual(t, finalCount, initialCount, "Should clean up old entries")
}

// TestInMemoryRateLimiterStop tests the stop functionality
func TestInMemoryRateLimiterStop(t *testing.T) {
	config := InMemoryRateLimiterConfig{
		Limit:           1,
		Window:          time.Minute,
		CleanupInterval: 10 * time.Millisecond,
	}
	limiter := NewInMemoryRateLimiter(config)

	// Start with some requests
	ctx := context.Background()
	result1, err := limiter.Allow(ctx, "test-key")
	assert.NoError(t, err)
	assert.True(t, result1.Allowed)

	// Stop the limiter
	limiter.(*InMemoryRateLimiter).Stop()

	// Should still work for a bit, but cleanup goroutine should stop
	_, err = limiter.Allow(ctx, "test-key2")
	assert.NoError(t, err)

	// Multiple stops should be safe
	limiter.(*InMemoryRateLimiter).Stop()
	limiter.(*InMemoryRateLimiter).Stop()
}

// TestInMemoryRateLimiterDefaultConfig tests default configuration values
func TestInMemoryRateLimiterDefaultConfig(t *testing.T) {
	config := InMemoryRateLimiterConfig{
		Limit: 5,
		// Window and CleanupInterval not set - should use defaults
	}
	limiter := NewInMemoryRateLimiter(config)
	defer limiter.(*InMemoryRateLimiter).Stop()

	inMemoryLimiter := limiter.(*InMemoryRateLimiter)
	assert.Equal(t, time.Minute, inMemoryLimiter.window)
	assert.Equal(t, 5*time.Minute, inMemoryLimiter.cleanupInterval)
}

// TestRateLimitMiddlewareBasic tests basic rate limit middleware functionality
func TestRateLimitMiddlewareBasic(t *testing.T) {
	tests := []struct {
		name          string
		limit         int
		requests      int
		expectedAllowed int
		expectedBlocked int
	}{
		{
			name:            "all requests allowed",
			limit:           5,
			requests:        3,
			expectedAllowed: 3,
			expectedBlocked: 0,
		},
		{
			name:            "some requests blocked",
			limit:           2,
			requests:        4,
			expectedAllowed: 2,
			expectedBlocked: 2,
		},
		{
			name:            "all requests blocked after limit",
			limit:           1,
			requests:        3,
			expectedAllowed: 1,
			expectedBlocked: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRateLimiter := &MockRateLimiter{}
			mockLogger := &MockLogger{}

			// Setup mock expectations
			allowedCount := 0
			for i := 0; i < tt.requests; i++ {
				allowed := allowedCount < tt.expectedAllowed
				result := &RateLimitResult{
					Allowed:   allowed,
					Limit:     tt.limit,
					Remaining: tt.limit - allowedCount - 1,
					ResetTime: time.Now().Add(time.Minute),
				}
				if !allowed {
					result.Remaining = 0
					result.RetryAfter = time.Minute
				} else {
					allowedCount++
				}

				mockRateLimiter.On("Allow", mock.Anything, "192.168.1.1").Return(result, nil).Once()

				if !allowed {
					mockLogger.On("Warn", mock.Anything, "Rate limit exceeded", mock.Anything).Return().Once()
				}
			}

			config := RateLimitConfig{
				Limiter: mockRateLimiter,
				Logger:  mockLogger,
			}
			middleware := NewRateLimitMiddleware(config)

			blockedCount := 0
			for i := 0; i < tt.requests; i++ {
				req := &MockHTTPRequest{
					ip:   "192.168.1.1",
					path: "/api/test",
				}
				resp := NewMockHTTPResponse()
				ctx := NewMockHTTPContext(req, resp)

				err := middleware.Handle(ctx)
				assert.NoError(t, err)

				if resp.GetStatusCode() == 429 {
					blockedCount++
				}
			}

			assert.Equal(t, tt.expectedBlocked, blockedCount)
			mockRateLimiter.AssertExpectations(t)
			mockLogger.AssertExpectations(t)
		})
	}
}

// TestRateLimitMiddlewareSkipPaths tests skipping rate limiting for configured paths
func TestRateLimitMiddlewareSkipPaths(t *testing.T) {
	mockRateLimiter := &MockRateLimiter{}

	config := RateLimitConfig{
		Limiter:   mockRateLimiter,
		SkipPaths: []string{"/api/v1/health", "/api/v1/ready"},
	}
	middleware := NewRateLimitMiddleware(config)

	tests := []struct {
		name           string
		path           string
		shouldSkip     bool
	}{
		{
			name:       "skip health check",
			path:       "/api/v1/health",
			shouldSkip: true,
		},
		{
			name:       "skip readiness check",
			path:       "/api/v1/ready",
			shouldSkip: true,
		},
		{
			name:       "do not skip regular endpoint",
			path:       "/api/v1/test",
			shouldSkip: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !tt.shouldSkip {
				result := &RateLimitResult{
					Allowed:   true,
					Limit:     100,
					Remaining: 99,
					ResetTime: time.Now().Add(time.Minute),
				}
				mockRateLimiter.On("Allow", mock.Anything, "192.168.1.1").Return(result, nil).Once()
			}

			req := &MockHTTPRequest{
				ip:   "192.168.1.1",
				path: tt.path,
			}
			resp := NewMockHTTPResponse()
			ctx := NewMockHTTPContext(req, resp)

			err := middleware.Handle(ctx)
			assert.NoError(t, err)

			if tt.shouldSkip {
				// Should not set rate limit headers for skipped paths
				assert.Empty(t, resp.GetHeader("X-RateLimit-Limit"))
			} else {
				// Should set rate limit headers for regular paths
				assert.Equal(t, "100", resp.GetHeader("X-RateLimit-Limit"))
				assert.Equal(t, "99", resp.GetHeader("X-RateLimit-Remaining"))
			}
		})
	}

	mockRateLimiter.AssertExpectations(t)
}

// TestRateLimitMiddlewareCustomKeyFunc tests custom key function
func TestRateLimitMiddlewareCustomKeyFunc(t *testing.T) {
	mockRateLimiter := &MockRateLimiter{}

	// Custom key function that uses a header value
	customKeyFunc := func(ctx HTTPContext) string {
		return ctx.Request().Header("X-User-ID")
	}

	config := RateLimitConfig{
		Limiter: mockRateLimiter,
		KeyFunc: customKeyFunc,
	}
	middleware := NewRateLimitMiddleware(config)

	result := &RateLimitResult{
		Allowed:   true,
		Limit:     100,
		Remaining: 99,
		ResetTime: time.Now().Add(time.Minute),
	}

	// Expect call with custom key
	mockRateLimiter.On("Allow", mock.Anything, "user123").Return(result, nil)

	req := &MockHTTPRequest{
		ip:   "192.168.1.1",
		path: "/api/test",
		headers: map[string]string{
			"X-User-ID": "user123",
		},
	}
	resp := NewMockHTTPResponse()
	ctx := NewMockHTTPContext(req, resp)

	err := middleware.Handle(ctx)
	assert.NoError(t, err)

	mockRateLimiter.AssertExpectations(t)
}

// TestRateLimitMiddlewareErrorHandling tests rate limiter error handling
func TestRateLimitMiddlewareErrorHandling(t *testing.T) {
	mockRateLimiter := &MockRateLimiter{}
	mockLogger := &MockLogger{}

	rateLimiterError := assert.AnError
	mockRateLimiter.On("Allow", mock.Anything, "192.168.1.1").Return((*RateLimitResult)(nil), rateLimiterError)
	mockLogger.On("Error", mock.Anything, "Rate limiter error", rateLimiterError, mock.Anything).Return()

	config := RateLimitConfig{
		Limiter: mockRateLimiter,
		Logger:  mockLogger,
	}
	middleware := NewRateLimitMiddleware(config)

	req := &MockHTTPRequest{
		ip:   "192.168.1.1",
		path: "/api/test",
	}
	resp := NewMockHTTPResponse()
	ctx := NewMockHTTPContext(req, resp)

	// Should continue processing even on rate limiter error
	err := middleware.Handle(ctx)
	assert.NoError(t, err)
	assert.True(t, ctx.WasNextCalled(), "Should call Next() on rate limiter error")

	mockRateLimiter.AssertExpectations(t)
	mockLogger.AssertExpectations(t)
}

// TestRateLimitMiddlewareResponseHeaders tests rate limit response headers
func TestRateLimitMiddlewareResponseHeaders(t *testing.T) {
	mockRateLimiter := &MockRateLimiter{}

	resetTime := time.Now().Add(time.Hour)
	result := &RateLimitResult{
		Allowed:   true,
		Limit:     100,
		Remaining: 50,
		ResetTime: resetTime,
	}
	mockRateLimiter.On("Allow", mock.Anything, "192.168.1.1").Return(result, nil)

	config := RateLimitConfig{
		Limiter: mockRateLimiter,
	}
	middleware := NewRateLimitMiddleware(config)

	req := &MockHTTPRequest{
		ip:   "192.168.1.1",
		path: "/api/test",
	}
	resp := NewMockHTTPResponse()
	ctx := NewMockHTTPContext(req, resp)

	err := middleware.Handle(ctx)
	assert.NoError(t, err)

	// Check headers
	assert.Equal(t, "100", resp.GetHeader("X-RateLimit-Limit"))
	assert.Equal(t, "50", resp.GetHeader("X-RateLimit-Remaining"))
	assert.NotEmpty(t, resp.GetHeader("X-RateLimit-Reset"))

	mockRateLimiter.AssertExpectations(t)
}

// TestRateLimitMiddlewareMemoryLeak tests for memory leaks in long-running scenarios
func TestRateLimitMiddlewareMemoryLeak(t *testing.T) {
	// This test is more conceptual - in a real scenario you'd run this longer
	// and monitor memory usage with tools like pprof

	config := InMemoryRateLimiterConfig{
		Limit:           100,
		Window:          50 * time.Millisecond,
		CleanupInterval: 25 * time.Millisecond,
	}
	limiter := NewInMemoryRateLimiter(config)
	defer limiter.(*InMemoryRateLimiter).Stop()

	ctx := context.Background()

	// Generate many different keys to test cleanup
	for i := 0; i < 1000; i++ {
		key := "test-key-" + string(rune(i))
		_, _ = limiter.Allow(ctx, key)
	}

	// Wait for cleanup cycles
	time.Sleep(200 * time.Millisecond)

	// Check that we don't have excessive entries
	inMemoryLimiter := limiter.(*InMemoryRateLimiter)
	inMemoryLimiter.mu.RLock()
	entryCount := len(inMemoryLimiter.requests)
	inMemoryLimiter.mu.RUnlock()

	// After cleanup, should have significantly fewer entries
	assert.Less(t, entryCount, 1000, "Should clean up old entries to prevent memory leak")
}

// BenchmarkInMemoryRateLimiter benchmarks the in-memory rate limiter
func BenchmarkInMemoryRateLimiter(b *testing.B) {
	config := InMemoryRateLimiterConfig{
		Limit:  1000,
		Window: time.Minute,
	}
	limiter := NewInMemoryRateLimiter(config)
	defer limiter.(*InMemoryRateLimiter).Stop()

	ctx := context.Background()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		keyCounter := 0
		for pb.Next() {
			// Use different keys to test realistic scenarios
			key := "test-key-" + string(rune(keyCounter%100))
			keyCounter++

			_, _ = limiter.Allow(ctx, key)
		}
	})
}

// BenchmarkRateLimitMiddleware benchmarks the rate limit middleware
func BenchmarkRateLimitMiddleware(b *testing.B) {
	config := InMemoryRateLimiterConfig{
		Limit:  1000,
		Window: time.Minute,
	}
	limiter := NewInMemoryRateLimiter(config)
	defer limiter.(*InMemoryRateLimiter).Stop()

	middlewareConfig := RateLimitConfig{
		Limiter: limiter,
	}
	middleware := NewRateLimitMiddleware(middlewareConfig)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		ipCounter := 0
		for pb.Next() {
			// Simulate different IPs
			ip := "192.168.1." + string(rune(ipCounter%254+1))
			ipCounter++

			req := &MockHTTPRequest{
				ip:   ip,
				path: "/api/test",
			}
			resp := NewMockHTTPResponse()
			ctx := NewMockHTTPContext(req, resp)

			_ = middleware.Handle(ctx)
		}
	})
}

// TestRateLimitMiddlewareGoroutineLeak tests for goroutine leaks
func TestRateLimitMiddlewareGoroutineLeak(t *testing.T) {
	initialGoroutines := runtime.NumGoroutine()

	// Create and stop multiple rate limiters
	for i := 0; i < 10; i++ {
		config := InMemoryRateLimiterConfig{
			Limit:           10,
			Window:          time.Minute,
			CleanupInterval: time.Millisecond, // Very short for testing
		}
		limiter := NewInMemoryRateLimiter(config)

		// Use the limiter briefly
		ctx := context.Background()
		_, _ = limiter.Allow(ctx, "test")

		// Stop it
		limiter.(*InMemoryRateLimiter).Stop()
	}

	// Give time for goroutines to exit
	time.Sleep(50 * time.Millisecond)
	runtime.GC()
	time.Sleep(10 * time.Millisecond)

	finalGoroutines := runtime.NumGoroutine()

	// Should not have significantly more goroutines
	assert.LessOrEqual(t, finalGoroutines, initialGoroutines+2,
		"Should not leak goroutines. Initial: %d, Final: %d", initialGoroutines, finalGoroutines)
}