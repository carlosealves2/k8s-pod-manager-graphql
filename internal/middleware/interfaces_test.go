package middleware

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestFieldStruct tests the Field struct
func TestFieldStruct(t *testing.T) {
	tests := []struct {
		name        string
		field       Field
		expectedKey string
		expectedVal interface{}
	}{
		{
			name:        "string field",
			field:       Field{Key: "message", Value: "test message"},
			expectedKey: "message",
			expectedVal: "test message",
		},
		{
			name:        "int field",
			field:       Field{Key: "status", Value: 200},
			expectedKey: "status",
			expectedVal: 200,
		},
		{
			name:        "time field",
			field:       Field{Key: "timestamp", Value: time.Now()},
			expectedKey: "timestamp",
		},
		{
			name:        "nil value field",
			field:       Field{Key: "optional", Value: nil},
			expectedKey: "optional",
			expectedVal: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expectedKey, tt.field.Key)
			if tt.name != "time field" {
				assert.Equal(t, tt.expectedVal, tt.field.Value)
			} else {
				assert.IsType(t, time.Time{}, tt.field.Value)
			}
		})
	}
}

// TestAuditEntry tests the AuditEntry struct
func TestAuditEntry(t *testing.T) {
	now := time.Now()
	entry := &AuditEntry{
		RequestID:    "test-request-id",
		Method:       "POST",
		Path:         "/api/v1/test",
		IP:           "192.168.1.1",
		UserAgent:    "test-agent",
		User:         "test-user",
		RequestBody:  `{"test": "data"}`,
		StatusCode:   200,
		Duration:     100 * time.Millisecond,
		Error:        "",
		Action:       "test_action",
		ResourceType: "test_resource",
		ResourceName: "test_name",
		Namespace:    "test-namespace",
		Timestamp:    now,
	}

	// Test all fields are set correctly
	assert.Equal(t, "test-request-id", entry.RequestID)
	assert.Equal(t, "POST", entry.Method)
	assert.Equal(t, "/api/v1/test", entry.Path)
	assert.Equal(t, "192.168.1.1", entry.IP)
	assert.Equal(t, "test-agent", entry.UserAgent)
	assert.Equal(t, "test-user", entry.User)
	assert.Equal(t, `{"test": "data"}`, entry.RequestBody)
	assert.Equal(t, 200, entry.StatusCode)
	assert.Equal(t, 100*time.Millisecond, entry.Duration)
	assert.Empty(t, entry.Error)
	assert.Equal(t, "test_action", entry.Action)
	assert.Equal(t, "test_resource", entry.ResourceType)
	assert.Equal(t, "test_name", entry.ResourceName)
	assert.Equal(t, "test-namespace", entry.Namespace)
	assert.Equal(t, now, entry.Timestamp)
}

// TestRateLimitResult tests the RateLimitResult struct
func TestRateLimitResult(t *testing.T) {
	tests := []struct {
		name   string
		result *RateLimitResult
	}{
		{
			name: "allowed request",
			result: &RateLimitResult{
				Allowed:    true,
				Limit:      100,
				Remaining:  50,
				ResetTime:  time.Now().Add(time.Minute),
				RetryAfter: 0,
			},
		},
		{
			name: "blocked request",
			result: &RateLimitResult{
				Allowed:    false,
				Limit:      100,
				Remaining:  0,
				ResetTime:  time.Now().Add(time.Minute),
				RetryAfter: time.Minute,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.result.Allowed {
				assert.True(t, tt.result.Allowed)
				assert.Greater(t, tt.result.Remaining, 0)
				assert.Equal(t, time.Duration(0), tt.result.RetryAfter)
			} else {
				assert.False(t, tt.result.Allowed)
				assert.Equal(t, 0, tt.result.Remaining)
				assert.Greater(t, tt.result.RetryAfter, time.Duration(0))
			}
			assert.Equal(t, 100, tt.result.Limit)
			assert.True(t, tt.result.ResetTime.After(time.Now()))
		})
	}
}

// TestMiddlewareFunc tests the MiddlewareFunc adapter
func TestMiddlewareFunc(t *testing.T) {
	t.Run("executes function correctly", func(t *testing.T) {
		// Arrange
		called := false
		fn := MiddlewareFunc(func(ctx HTTPContext) error {
			called = true
			return nil
		})

		req := &MockHTTPRequest{
			method: "GET",
			path:   "/test",
		}
		resp := NewMockHTTPResponse()
		ctx := NewMockHTTPContext(req, resp)

		// Act
		err := fn.Handle(ctx)

		// Assert
		assert.NoError(t, err)
		assert.True(t, called)
	})

	t.Run("propagates error from function", func(t *testing.T) {
		// Arrange
		expectedErr := assert.AnError
		fn := MiddlewareFunc(func(ctx HTTPContext) error {
			return expectedErr
		})

		req := &MockHTTPRequest{
			method: "GET",
			path:   "/test",
		}
		resp := NewMockHTTPResponse()
		ctx := NewMockHTTPContext(req, resp)

		// Act
		err := fn.Handle(ctx)

		// Assert
		assert.Error(t, err)
		assert.Equal(t, expectedErr, err)
	})
}

// TestUUIDGenerator tests the UUID generator implementation
func TestUUIDGenerator(t *testing.T) {
	generator := &UUIDGenerator{}

	t.Run("generates valid UUID format", func(t *testing.T) {
		id := generator.Generate()

		// UUID v4 format: xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx
		assert.Len(t, id, 36) // 32 hex chars + 4 hyphens
		assert.Contains(t, id, "-")

		// Split by hyphens and check segment lengths
		parts := []string{}
		current := ""
		for _, char := range id {
			if char == '-' {
				parts = append(parts, current)
				current = ""
			} else {
				current += string(char)
			}
		}
		parts = append(parts, current) // Add the last part

		require.Len(t, parts, 5)
		assert.Len(t, parts[0], 8)  // xxxxxxxx
		assert.Len(t, parts[1], 4)  // xxxx
		assert.Len(t, parts[2], 4)  // 4xxx
		assert.Len(t, parts[3], 4)  // yxxx
		assert.Len(t, parts[4], 12) // xxxxxxxxxxxx

		// Check version bit (4th character of 3rd segment should be '4')
		assert.Equal(t, byte('4'), parts[2][0])
	})

	t.Run("generates unique IDs", func(t *testing.T) {
		ids := make(map[string]bool)
		for i := 0; i < 100; i++ {
			id := generator.Generate()
			assert.False(t, ids[id], "Generated duplicate ID: %s", id)
			ids[id] = true
		}
	})

	t.Run("fallback generation when crypto/rand fails", func(t *testing.T) {
		// This test is harder to implement without modifying the code,
		// but we can at least verify the function doesn't panic
		assert.NotPanics(t, func() {
			generator.Generate()
		})
	})
}

// TestInterfaceCompliance tests that our mock types properly implement interfaces
func TestInterfaceCompliance(t *testing.T) {
	t.Run("MockHTTPRequest implements HTTPRequest", func(t *testing.T) {
		var _ HTTPRequest = &MockHTTPRequest{}
	})

	t.Run("MockHTTPResponse implements HTTPResponse", func(t *testing.T) {
		var _ HTTPResponse = &MockHTTPResponse{}
	})

	t.Run("MockHTTPContext implements HTTPContext", func(t *testing.T) {
		var _ HTTPContext = &MockHTTPContext{}
	})

	t.Run("MockLogger implements Logger", func(t *testing.T) {
		var _ Logger = &MockLogger{}
	})

	t.Run("MockAuditLogger implements AuditLogger", func(t *testing.T) {
		var _ AuditLogger = &MockAuditLogger{}
	})

	t.Run("MockRateLimiter implements RateLimiter", func(t *testing.T) {
		var _ RateLimiter = &MockRateLimiter{}
	})

	t.Run("MockRequestIDGenerator implements RequestIDGenerator", func(t *testing.T) {
		var _ RequestIDGenerator = &MockRequestIDGenerator{}
	})

	t.Run("MockErrorHandler implements ErrorHandler", func(t *testing.T) {
		var _ ErrorHandler = &MockErrorHandler{}
	})

	t.Run("UUIDGenerator implements RequestIDGenerator", func(t *testing.T) {
		var _ RequestIDGenerator = &UUIDGenerator{}
	})

	t.Run("MiddlewareFunc implements Middleware", func(t *testing.T) {
		var _ Middleware = MiddlewareFunc(func(ctx HTTPContext) error { return nil })
	})
}

// TestContextValuesPropagation tests that values are properly set and retrieved from context
func TestContextValuesPropagation(t *testing.T) {
	req := &MockHTTPRequest{
		method: "GET",
		path:   "/test",
	}
	resp := NewMockHTTPResponse()
	ctx := NewMockHTTPContext(req, resp)

	t.Run("set and get string values", func(t *testing.T) {
		ctx.Set("key1", "value1")
		value, exists := ctx.Get("key1")
		assert.True(t, exists)
		assert.Equal(t, "value1", value)
	})

	t.Run("set and get complex values", func(t *testing.T) {
		complexValue := map[string]interface{}{
			"nested": "data",
			"number": 42,
		}
		ctx.Set("complex", complexValue)
		value, exists := ctx.Get("complex")
		assert.True(t, exists)
		assert.Equal(t, complexValue, value)
	})

	t.Run("get non-existent key", func(t *testing.T) {
		value, exists := ctx.Get("non-existent")
		assert.False(t, exists)
		assert.Nil(t, value)
	})

	t.Run("overwrite existing value", func(t *testing.T) {
		ctx.Set("overwrite", "original")
		ctx.Set("overwrite", "new")
		value, exists := ctx.Get("overwrite")
		assert.True(t, exists)
		assert.Equal(t, "new", value)
	})
}