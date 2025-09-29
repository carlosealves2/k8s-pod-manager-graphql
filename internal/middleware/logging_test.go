package middleware

import (
	"bytes"
	"context"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// TestSlogLogger tests the slog-based logger implementation
func TestSlogLogger(t *testing.T) {
	tests := []struct {
		name     string
		level    slog.Level
		logFunc  func(Logger, context.Context)
		expected bool // whether log should be written
	}{
		{
			name:  "debug level with debug log",
			level: slog.LevelDebug,
			logFunc: func(l Logger, ctx context.Context) {
				l.Debug(ctx, "debug message", Field{Key: "test", Value: "value"})
			},
			expected: true,
		},
		{
			name:  "info level with debug log",
			level: slog.LevelInfo,
			logFunc: func(l Logger, ctx context.Context) {
				l.Debug(ctx, "debug message", Field{Key: "test", Value: "value"})
			},
			expected: false,
		},
		{
			name:  "info level with info log",
			level: slog.LevelInfo,
			logFunc: func(l Logger, ctx context.Context) {
				l.Info(ctx, "info message", Field{Key: "test", Value: "value"})
			},
			expected: true,
		},
		{
			name:  "warn level with warn log",
			level: slog.LevelWarn,
			logFunc: func(l Logger, ctx context.Context) {
				l.Warn(ctx, "warn message", Field{Key: "test", Value: "value"})
			},
			expected: true,
		},
		{
			name:  "error level with error log",
			level: slog.LevelError,
			logFunc: func(l Logger, ctx context.Context) {
				l.Error(ctx, "error message", assert.AnError, Field{Key: "test", Value: "value"})
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			opts := &slog.HandlerOptions{Level: tt.level}
			handler := slog.NewJSONHandler(&buf, opts)
			logger := &SlogLogger{logger: slog.New(handler)}

			ctx := context.Background()
			tt.logFunc(logger, ctx)

			if tt.expected {
				assert.Greater(t, buf.Len(), 0, "Should write log output")
			} else {
				assert.Equal(t, 0, buf.Len(), "Should not write log output")
			}
		})
	}
}

// TestConvertFields tests the field conversion utility
func TestConvertFields(t *testing.T) {
	tests := []struct {
		name     string
		fields   []Field
		expected []interface{}
	}{
		{
			name:     "empty fields",
			fields:   []Field{},
			expected: []interface{}{},
		},
		{
			name: "single field",
			fields: []Field{
				{Key: "test", Value: "value"},
			},
			expected: []interface{}{"test", "value"},
		},
		{
			name: "multiple fields",
			fields: []Field{
				{Key: "key1", Value: "value1"},
				{Key: "key2", Value: 42},
				{Key: "key3", Value: true},
			},
			expected: []interface{}{"key1", "value1", "key2", 42, "key3", true},
		},
		{
			name: "nil value field",
			fields: []Field{
				{Key: "nil_field", Value: nil},
			},
			expected: []interface{}{"nil_field", nil},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := convertFields(tt.fields)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestLoggingMiddlewareBasicOperation tests basic logging middleware functionality
func TestLoggingMiddlewareBasicOperation(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		path           string
		requestID      string
		nextError      error
		expectError    bool
		skipPaths      []string
		shouldLog      bool
	}{
		{
			name:        "successful request",
			method:      "GET",
			path:        "/api/v1/test",
			requestID:   "test-request-id",
			nextError:   nil,
			expectError: false,
			shouldLog:   true,
		},
		{
			name:        "failed request",
			method:      "POST",
			path:        "/api/v1/create",
			requestID:   "test-request-id-2",
			nextError:   assert.AnError,
			expectError: true,
			shouldLog:   true,
		},
		{
			name:        "skipped path",
			method:      "GET",
			path:        "/api/v1/health",
			requestID:   "test-request-id-3",
			nextError:   nil,
			expectError: false,
			skipPaths:   []string{"/api/v1/health"},
			shouldLog:   false,
		},
		{
			name:        "no request ID",
			method:      "GET",
			path:        "/api/v1/test",
			requestID:   "",
			nextError:   nil,
			expectError: false,
			shouldLog:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockLogger := &MockLogger{}
			mockAuditLogger := &MockAuditLogger{}

			// Setup expectations based on shouldLog
			if tt.shouldLog {
				if tt.expectError {
					mockLogger.On("Info", mock.Anything, "Request started", mock.AnythingOfType("[]middleware.Field")).Return()
					mockLogger.On("Error", mock.Anything, "Request failed", tt.nextError, mock.AnythingOfType("[]middleware.Field")).Return()
				} else {
					mockLogger.On("Info", mock.Anything, "Request started", mock.AnythingOfType("[]middleware.Field")).Return()
					mockLogger.On("Info", mock.Anything, "Request completed", mock.AnythingOfType("[]middleware.Field")).Return()
				}

				// Audit logger should always be called if configured
				mockAuditLogger.On("LogRequest", mock.Anything, mock.AnythingOfType("*middleware.AuditEntry")).Return(nil)
			}

			config := LoggingConfig{
				Logger:      mockLogger,
				AuditLogger: mockAuditLogger,
				SkipPaths:   tt.skipPaths,
			}
			middleware := NewLoggingMiddleware(config)

			req := &MockHTTPRequest{
				method:    tt.method,
				path:      tt.path,
				ip:        "192.168.1.1",
				userAgent: "test-agent",
				headers:   map[string]string{"X-User": "testuser"},
			}
			resp := NewMockHTTPResponse()
			ctx := NewMockHTTPContext(req, resp)

			if tt.requestID != "" {
				ctx.Set("requestID", tt.requestID)
			}
			ctx.SetNextError(tt.nextError)

			// Execute middleware
			err := middleware.Handle(ctx)

			// Verify error handling
			if tt.expectError {
				assert.Error(t, err)
				assert.Equal(t, tt.nextError, err)
			} else {
				assert.NoError(t, err)
			}

			// Wait a bit for async audit logging
			time.Sleep(10 * time.Millisecond)

			// Verify mock expectations
			if tt.shouldLog {
				mockLogger.AssertExpectations(t)
				mockAuditLogger.AssertExpectations(t)
			} else {
				mockLogger.AssertNotCalled(t, "Info")
				mockLogger.AssertNotCalled(t, "Error")
			}
		})
	}
}

// TestLoggingMiddlewareAuditEntry tests audit entry creation
func TestLoggingMiddlewareAuditEntry(t *testing.T) {
	mockLogger := &MockLogger{}
	mockAuditLogger := &MockAuditLogger{}

	// Capture the audit entry
	var capturedEntry *AuditEntry
	mockAuditLogger.On("LogRequest", mock.Anything, mock.AnythingOfType("*middleware.AuditEntry")).
		Run(func(args mock.Arguments) {
			capturedEntry = args.Get(1).(*AuditEntry)
		}).Return(nil)

	mockLogger.On("Info", mock.Anything, mock.Anything, mock.Anything).Return()

	config := LoggingConfig{
		Logger:      mockLogger,
		AuditLogger: mockAuditLogger,
	}
	middleware := NewLoggingMiddleware(config)

	req := &MockHTTPRequest{
		method:    "POST",
		path:      "/api/v1/create",
		ip:        "192.168.1.1",
		userAgent: "test-agent/1.0",
		headers:   map[string]string{"X-User": "testuser"},
		body:      []byte(`{"test": "data"}`),
	}
	resp := NewMockHTTPResponse()
	ctx := NewMockHTTPContext(req, resp)
	ctx.Set("requestID", "test-request-id")

	err := middleware.Handle(ctx)
	assert.NoError(t, err)

	// Wait for async audit logging
	time.Sleep(50 * time.Millisecond)

	// Verify audit entry
	require.NotNil(t, capturedEntry, "Audit entry should be captured")
	assert.Equal(t, "test-request-id", capturedEntry.RequestID)
	assert.Equal(t, "POST", capturedEntry.Method)
	assert.Equal(t, "/api/v1/create", capturedEntry.Path)
	assert.Equal(t, "192.168.1.1", capturedEntry.IP)
	assert.Equal(t, "test-agent/1.0", capturedEntry.UserAgent)
	assert.Equal(t, "testuser", capturedEntry.User)
	assert.Equal(t, `{"test": "data"}`, capturedEntry.RequestBody)
	assert.Equal(t, 200, capturedEntry.StatusCode) // Default status
	assert.Greater(t, capturedEntry.Duration, time.Duration(0))
	assert.Empty(t, capturedEntry.Error)
	assert.False(t, capturedEntry.Timestamp.IsZero())
}

// TestLoggingMiddlewareActionParsing tests action parsing functionality
func TestLoggingMiddlewareActionParsing(t *testing.T) {
	tests := []struct {
		name               string
		method             string
		path               string
		expectedAction     string
		expectedResource   string
		expectedName       string
	}{
		{
			name:             "GraphQL request",
			method:           "POST",
			path:             "/graphql",
			expectedAction:   "graphql_request",
			expectedResource: "graphql",
			expectedName:     "query",
		},
		{
			name:             "health check",
			method:           "GET",
			path:             "/api/v1/health",
			expectedAction:   "health_check",
			expectedResource: "system",
			expectedName:     "health",
		},
		{
			name:             "readiness check",
			method:           "GET",
			path:             "/api/v1/ready",
			expectedAction:   "readiness_check",
			expectedResource: "system",
			expectedName:     "readiness",
		},
		{
			name:             "metrics",
			method:           "GET",
			path:             "/api/v1/metrics",
			expectedAction:   "metrics",
			expectedResource: "system",
			expectedName:     "metrics",
		},
		{
			name:             "info endpoint",
			method:           "GET",
			path:             "/api/v1/info",
			expectedAction:   "info",
			expectedResource: "system",
			expectedName:     "info",
		},
		{
			name:             "GraphQL playground",
			method:           "GET",
			path:             "/",
			expectedAction:   "playground",
			expectedResource: "graphql",
			expectedName:     "playground",
		},
		{
			name:             "unknown endpoint",
			method:           "GET",
			path:             "/api/v1/unknown",
			expectedAction:   "unknown",
			expectedResource: "unknown",
			expectedName:     "/api/v1/unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockLogger := &MockLogger{}
			mockAuditLogger := &MockAuditLogger{}

			var capturedEntry *AuditEntry
			mockAuditLogger.On("LogRequest", mock.Anything, mock.AnythingOfType("*middleware.AuditEntry")).
				Run(func(args mock.Arguments) {
					capturedEntry = args.Get(1).(*AuditEntry)
				}).Return(nil)

			mockLogger.On("Info", mock.Anything, mock.Anything, mock.Anything).Return()

			config := LoggingConfig{
				Logger:      mockLogger,
				AuditLogger: mockAuditLogger,
			}
			middleware := NewLoggingMiddleware(config)

			req := &MockHTTPRequest{
				method: tt.method,
				path:   tt.path,
			}
			resp := NewMockHTTPResponse()
			ctx := NewMockHTTPContext(req, resp)

			err := middleware.Handle(ctx)
			assert.NoError(t, err)

			// Wait for async audit logging
			time.Sleep(50 * time.Millisecond)

			require.NotNil(t, capturedEntry)
			assert.Equal(t, tt.expectedAction, capturedEntry.Action)
			assert.Equal(t, tt.expectedResource, capturedEntry.ResourceType)
			assert.Equal(t, tt.expectedName, capturedEntry.ResourceName)
		})
	}
}

// TestExtractRequestBody tests request body extraction
func TestExtractRequestBody(t *testing.T) {
	tests := []struct {
		name     string
		method   string
		body     []byte
		expected string
	}{
		{
			name:     "GET request - no body extraction",
			method:   "GET",
			body:     []byte(`{"test": "data"}`),
			expected: "",
		},
		{
			name:     "POST request with JSON",
			method:   "POST",
			body:     []byte(`{"test": "data"}`),
			expected: `{"test": "data"}`,
		},
		{
			name:     "PUT request with body",
			method:   "PUT",
			body:     []byte(`{"update": "data"}`),
			expected: `{"update": "data"}`,
		},
		{
			name:     "PATCH request with body",
			method:   "PATCH",
			body:     []byte(`{"patch": "data"}`),
			expected: `{"patch": "data"}`,
		},
		{
			name:     "empty body",
			method:   "POST",
			body:     []byte{},
			expected: "",
		},
		{
			name:     "large body - should be truncated",
			method:   "POST",
			body:     bytes.Repeat([]byte("a"), 15000), // Larger than 10KB limit
			expected: "[BODY_TOO_LARGE]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &MockHTTPRequest{
				method: tt.method,
				body:   tt.body,
			}

			result := extractRequestBody(req)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestLoggingMiddlewareAuditErrorHandling tests audit logger error handling
func TestLoggingMiddlewareAuditErrorHandling(t *testing.T) {
	mockLogger := &MockLogger{}
	mockAuditLogger := &MockAuditLogger{}

	// Setup audit logger to return an error
	auditError := assert.AnError
	mockAuditLogger.On("LogRequest", mock.Anything, mock.AnythingOfType("*middleware.AuditEntry")).Return(auditError)

	// Logger should log the start and completion, plus the audit error
	mockLogger.On("Info", mock.Anything, "Request started", mock.Anything).Return()
	mockLogger.On("Info", mock.Anything, "Request completed", mock.Anything).Return()
	mockLogger.On("Error", mock.Anything, "Failed to log audit entry", auditError, mock.Anything).Return()

	config := LoggingConfig{
		Logger:      mockLogger,
		AuditLogger: mockAuditLogger,
	}
	middleware := NewLoggingMiddleware(config)

	req := &MockHTTPRequest{
		method: "GET",
		path:   "/api/v1/test",
	}
	resp := NewMockHTTPResponse()
	ctx := NewMockHTTPContext(req, resp)

	err := middleware.Handle(ctx)
	assert.NoError(t, err)

	// Wait for async audit logging
	time.Sleep(50 * time.Millisecond)

	mockLogger.AssertExpectations(t)
	mockAuditLogger.AssertExpectations(t)
}

// TestLoggingMiddlewareWithoutAuditLogger tests logging without audit logger
func TestLoggingMiddlewareWithoutAuditLogger(t *testing.T) {
	mockLogger := &MockLogger{}

	mockLogger.On("Info", mock.Anything, "Request started", mock.Anything).Return()
	mockLogger.On("Info", mock.Anything, "Request completed", mock.Anything).Return()

	config := LoggingConfig{
		Logger:      mockLogger,
		AuditLogger: nil, // No audit logger
	}
	middleware := NewLoggingMiddleware(config)

	req := &MockHTTPRequest{
		method: "GET",
		path:   "/api/v1/test",
	}
	resp := NewMockHTTPResponse()
	ctx := NewMockHTTPContext(req, resp)

	err := middleware.Handle(ctx)
	assert.NoError(t, err)

	mockLogger.AssertExpectations(t)
}

// TestLoggingMiddlewareConcurrency tests concurrent access
func TestLoggingMiddlewareConcurrency(t *testing.T) {
	mockLogger := &MockLogger{}
	mockAuditLogger := &MockAuditLogger{}

	// Allow multiple calls
	mockLogger.On("Info", mock.Anything, mock.Anything, mock.Anything).Return()
	mockAuditLogger.On("LogRequest", mock.Anything, mock.Anything).Return(nil)

	config := LoggingConfig{
		Logger:      mockLogger,
		AuditLogger: mockAuditLogger,
	}
	middleware := NewLoggingMiddleware(config)

	const numGoroutines = 50
	var wg sync.WaitGroup

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()

			req := &MockHTTPRequest{
				method: "GET",
				path:   "/api/v1/test",
			}
			resp := NewMockHTTPResponse()
			ctx := NewMockHTTPContext(req, resp)
			ctx.Set("requestID", "test-request-"+string(rune(index)))

			err := middleware.Handle(ctx)
			assert.NoError(t, err)
		}(i)
	}

	wg.Wait()

	// Wait for all async audit logging to complete
	time.Sleep(100 * time.Millisecond)

	// Verify that all calls were made (2 info calls per request)
	mockLogger.AssertNumberOfCalls(t, "Info", numGoroutines*2)
	mockAuditLogger.AssertNumberOfCalls(t, "LogRequest", numGoroutines)
}

// BenchmarkLoggingMiddleware benchmarks the logging middleware
func BenchmarkLoggingMiddleware(b *testing.B) {
	mockLogger := &MockLogger{}
	mockAuditLogger := &MockAuditLogger{}

	mockLogger.On("Info", mock.Anything, mock.Anything, mock.Anything).Return()
	mockAuditLogger.On("LogRequest", mock.Anything, mock.Anything).Return(nil)

	config := LoggingConfig{
		Logger:      mockLogger,
		AuditLogger: mockAuditLogger,
	}
	middleware := NewLoggingMiddleware(config)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			req := &MockHTTPRequest{
				method: "GET",
				path:   "/api/v1/test",
				ip:     "192.168.1.1",
			}
			resp := NewMockHTTPResponse()
			ctx := NewMockHTTPContext(req, resp)
			ctx.Set("requestID", "test-request-id")

			_ = middleware.Handle(ctx)
		}
	})
}

// BenchmarkSlogLogger benchmarks the slog logger
func BenchmarkSlogLogger(b *testing.B) {
	logger := NewSlogLogger(slog.LevelInfo)
	ctx := context.Background()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			logger.Info(ctx, "Benchmark message",
				Field{Key: "key1", Value: "value1"},
				Field{Key: "key2", Value: 42},
			)
		}
	})
}