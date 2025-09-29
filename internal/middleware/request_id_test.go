package middleware

import (
	"regexp"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestUUIDGeneratorConcurrency tests UUID generation under concurrent access
func TestUUIDGeneratorConcurrency(t *testing.T) {
	generator := &UUIDGenerator{}
	const numGoroutines = 100
	const numIDsPerGoroutine = 50

	ids := make(map[string]bool)
	var mu sync.Mutex
	var wg sync.WaitGroup

	// Generate UUIDs concurrently
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < numIDsPerGoroutine; j++ {
				id := generator.Generate()

				mu.Lock()
				if ids[id] {
					t.Errorf("Duplicate UUID generated: %s", id)
				}
				ids[id] = true
				mu.Unlock()
			}
		}()
	}

	wg.Wait()

	// Verify we generated the expected number of unique IDs
	assert.Len(t, ids, numGoroutines*numIDsPerGoroutine)
}

// TestUUIDGeneratorFormat tests UUID format compliance
func TestUUIDGeneratorFormat(t *testing.T) {
	generator := &UUIDGenerator{}

	// UUID v4 regex pattern
	uuidPattern := regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)

	for i := 0; i < 100; i++ {
		id := generator.Generate()

		t.Run("UUID_"+string(rune(i)), func(t *testing.T) {
			// Test format
			assert.Len(t, id, 36, "UUID should be 36 characters long")
			assert.Regexp(t, uuidPattern, id, "UUID should match v4 format")

			// Test version bit (13th character should be '4')
			assert.Equal(t, '4', rune(id[14]), "UUID version should be 4")

			// Test variant bits (17th character should be 8, 9, a, or b)
			variantChar := rune(id[19])
			assert.Contains(t, []rune{'8', '9', 'a', 'b'}, variantChar, "UUID variant should be valid")
		})
	}
}

// TestUUIDGeneratorFallback tests fallback behavior when crypto/rand fails
func TestUUIDGeneratorFallback(t *testing.T) {
	// This test is conceptual since we can't easily mock crypto/rand failure
	// But we can verify the fallback format
	generator := &UUIDGenerator{}

	// Generate multiple IDs and verify they don't follow the req_timestamp pattern
	// (which would indicate fallback was used)
	for i := 0; i < 10; i++ {
		id := generator.Generate()
		assert.NotContains(t, id, "req_", "Should not use fallback format in normal operation")
	}
}

// TestRequestIDMiddlewareConfig tests configuration variations
func TestRequestIDMiddlewareConfig(t *testing.T) {
	tests := []struct {
		name           string
		config         RequestIDConfig
		expectedHeader string
		expectsDefault bool
	}{
		{
			name: "default configuration",
			config: RequestIDConfig{},
			expectedHeader: "X-Request-ID",
			expectsDefault: true,
		},
		{
			name: "custom header",
			config: RequestIDConfig{
				Header: "X-Custom-Request-ID",
			},
			expectedHeader: "X-Custom-Request-ID",
			expectsDefault: true,
		},
		{
			name: "custom generator",
			config: RequestIDConfig{
				Generator: &MockRequestIDGenerator{},
				Header:    "X-Request-ID",
			},
			expectedHeader: "X-Request-ID",
			expectsDefault: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mock if needed
			if mockGen, ok := tt.config.Generator.(*MockRequestIDGenerator); ok {
				mockGen.On("Generate").Return("mock-generated-id")
			}

			middleware := NewRequestIDMiddleware(tt.config)

			// Create test context
			req := &MockHTTPRequest{
				headers: make(map[string]string),
			}
			resp := NewMockHTTPResponse()
			ctx := NewMockHTTPContext(req, resp)

			// Execute middleware
			err := middleware.Handle(ctx)

			// Assertions
			assert.NoError(t, err)
			assert.True(t, ctx.WasNextCalled(), "Should call Next()")

			// Check header was set
			headerValue := resp.GetHeader(tt.expectedHeader)
			assert.NotEmpty(t, headerValue, "Should set request ID header")

			// Check context was updated
			requestID, exists := ctx.Get("requestID")
			assert.True(t, exists, "Should set requestID in context")
			assert.Equal(t, headerValue, requestID, "Header and context should match")

			// Check alternative context key
			altRequestID, exists := ctx.Get("request_id")
			assert.True(t, exists, "Should set request_id in context")
			assert.Equal(t, headerValue, altRequestID, "Alternative key should match")

			// Verify mock expectations
			if mockGen, ok := tt.config.Generator.(*MockRequestIDGenerator); ok {
				mockGen.AssertExpectations(t)
				assert.Equal(t, "mock-generated-id", headerValue)
			} else if tt.expectsDefault {
				// Should use UUID format
				assert.Len(t, headerValue, 36, "Should generate UUID format")
			}
		})
	}
}

// TestRequestIDMiddlewarePreservesExisting tests preserving existing request IDs
func TestRequestIDMiddlewarePreservesExisting(t *testing.T) {
	tests := []struct {
		name        string
		existingID  string
		header      string
		shouldUseExisting bool
	}{
		{
			name:       "preserve existing request ID",
			existingID: "existing-request-123",
			header:     "X-Request-ID",
			shouldUseExisting: true,
		},
		{
			name:       "preserve existing custom header",
			existingID: "custom-id-456",
			header:     "X-Custom-Request-ID",
			shouldUseExisting: true,
		},
		{
			name:       "empty existing ID should generate new",
			existingID: "",
			header:     "X-Request-ID",
			shouldUseExisting: false,
		},
		{
			name:       "whitespace only ID should be preserved",
			existingID: "   ",
			header:     "X-Request-ID",
			shouldUseExisting: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := RequestIDConfig{
				Header: tt.header,
			}
			middleware := NewRequestIDMiddleware(config)

			req := &MockHTTPRequest{
				headers: map[string]string{
					tt.header: tt.existingID,
				},
			}
			resp := NewMockHTTPResponse()
			ctx := NewMockHTTPContext(req, resp)

			err := middleware.Handle(ctx)
			assert.NoError(t, err)

			if tt.shouldUseExisting && tt.existingID != "" {
				// Should preserve existing ID
				requestID, exists := ctx.Get("requestID")
				assert.True(t, exists)
				assert.Equal(t, tt.existingID, requestID)

				// Header should not be overwritten
				assert.Empty(t, resp.GetHeader(tt.header), "Should not overwrite existing header")
			} else {
				// Should generate new ID
				requestID, exists := ctx.Get("requestID")
				assert.True(t, exists)
				assert.NotEqual(t, tt.existingID, requestID)
				assert.NotEmpty(t, requestID)

				// New header should be set
				assert.Equal(t, requestID, resp.GetHeader(tt.header))
			}
		})
	}
}

// TestRequestIDMiddlewareChaining tests middleware chaining behavior
func TestRequestIDMiddlewareChaining(t *testing.T) {
	config := RequestIDConfig{
		Header: "X-Request-ID",
	}
	middleware := NewRequestIDMiddleware(config)

	req := &MockHTTPRequest{
		headers: make(map[string]string),
	}
	resp := NewMockHTTPResponse()
	ctx := NewMockHTTPContext(req, resp)
	ctx.SetNextError(nil) // Ensure Next() succeeds

	err := middleware.Handle(ctx)
	assert.NoError(t, err)
	assert.True(t, ctx.WasNextCalled(), "Next() should be called")

	// Verify request ID was set and is available for chained middleware
	requestID, exists := ctx.Get("requestID")
	assert.True(t, exists, "Request ID should be available to chained middleware")
	assert.NotEmpty(t, requestID, "Request ID should not be empty")
}

// TestRequestIDMiddlewareError tests error handling
func TestRequestIDMiddlewareError(t *testing.T) {
	tests := []struct {
		name        string
		nextError   error
		expectError bool
	}{
		{
			name:        "no error from Next()",
			nextError:   nil,
			expectError: false,
		},
		{
			name:        "error from Next()",
			nextError:   assert.AnError,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := RequestIDConfig{}
			middleware := NewRequestIDMiddleware(config)

			req := &MockHTTPRequest{
				headers: make(map[string]string),
			}
			resp := NewMockHTTPResponse()
			ctx := NewMockHTTPContext(req, resp)
			ctx.SetNextError(tt.nextError)

			err := middleware.Handle(ctx)

			// Request ID should still be set even if Next() fails
			requestID, exists := ctx.Get("requestID")
			assert.True(t, exists, "Request ID should be set even on Next() error")
			assert.NotEmpty(t, requestID)

			if tt.expectError {
				assert.Error(t, err)
				assert.Equal(t, tt.nextError, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestRequestIDMiddlewareThreadSafety tests thread safety
func TestRequestIDMiddlewareThreadSafety(t *testing.T) {
	config := RequestIDConfig{}
	middleware := NewRequestIDMiddleware(config)

	const numGoroutines = 50
	var wg sync.WaitGroup
	results := make([]string, numGoroutines)

	// Process requests concurrently
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()

			req := &MockHTTPRequest{
				headers: make(map[string]string),
			}
			resp := NewMockHTTPResponse()
			ctx := NewMockHTTPContext(req, resp)

			err := middleware.Handle(ctx)
			assert.NoError(t, err)

			requestID, exists := ctx.Get("requestID")
			assert.True(t, exists)
			results[index] = requestID.(string)
		}(i)
	}

	wg.Wait()

	// Verify all request IDs are unique
	seen := make(map[string]bool)
	for i, id := range results {
		assert.NotEmpty(t, id, "Request ID %d should not be empty", i)
		assert.False(t, seen[id], "Request ID should be unique: %s", id)
		seen[id] = true
	}
}

// BenchmarkRequestIDMiddleware benchmarks the request ID middleware
func BenchmarkRequestIDMiddleware(b *testing.B) {
	config := RequestIDConfig{}
	middleware := NewRequestIDMiddleware(config)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			req := &MockHTTPRequest{
				headers: make(map[string]string),
			}
			resp := NewMockHTTPResponse()
			ctx := NewMockHTTPContext(req, resp)

			_ = middleware.Handle(ctx)
		}
	})
}

// BenchmarkUUIDGenerator benchmarks the UUID generator
func BenchmarkUUIDGenerator(b *testing.B) {
	generator := &UUIDGenerator{}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = generator.Generate()
		}
	})
}

// TestRequestIDMiddlewareIntegration tests integration with the full middleware stack
func TestRequestIDMiddlewareIntegration(t *testing.T) {
	// Create a simple middleware chain
	requestIDMiddleware := NewRequestIDMiddleware(RequestIDConfig{})

	// Mock logging middleware that depends on request ID
	loggingCalled := false
	loggingMiddleware := MiddlewareFunc(func(ctx HTTPContext) error {
		loggingCalled = true

		// Verify request ID is available
		requestID, exists := ctx.Get("requestID")
		assert.True(t, exists, "Request ID should be available to logging middleware")
		assert.NotEmpty(t, requestID, "Request ID should not be empty")

		return ctx.Next()
	})

	req := &MockHTTPRequest{
		headers: make(map[string]string),
	}
	resp := NewMockHTTPResponse()
	ctx := NewMockHTTPContext(req, resp)

	// Execute chain: RequestID -> Logging -> Next
	chainCtx := &chainHTTPContext{
		HTTPContext: ctx,
		middlewares: []Middleware{loggingMiddleware},
		index:       0,
	}

	err := requestIDMiddleware.Handle(chainCtx)
	assert.NoError(t, err)
	assert.True(t, loggingCalled, "Logging middleware should be called")

	// Verify request ID was set
	requestID, exists := ctx.Get("requestID")
	assert.True(t, exists)
	assert.NotEmpty(t, requestID)
}