package middleware

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// Mock implementations for testing

type MockHTTPRequest struct {
	method    string
	path      string
	headers   map[string]string
	body      []byte
	ip        string
	userAgent string
	requestID string
	mu        sync.RWMutex
}

func (m *MockHTTPRequest) Method() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.method
}
func (m *MockHTTPRequest) Path() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.path
}
func (m *MockHTTPRequest) Header(key string) string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.headers[key]
}
func (m *MockHTTPRequest) Body() []byte {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.body
}
func (m *MockHTTPRequest) IP() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.ip
}
func (m *MockHTTPRequest) UserAgent() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.userAgent
}
func (m *MockHTTPRequest) RequestID() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.requestID
}

// Helper method to set headers thread-safely
func (m *MockHTTPRequest) SetHeader(key, value string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.headers == nil {
		m.headers = make(map[string]string)
	}
	m.headers[key] = value
}

// Helper method to set request ID thread-safely
func (m *MockHTTPRequest) SetRequestID(id string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.requestID = id
}

type MockHTTPResponse struct {
	headers    map[string]string
	statusCode int
	data       interface{}
	writeErr   error
	jsonErr    error
	mu         sync.RWMutex
}

func NewMockHTTPResponse() *MockHTTPResponse {
	return &MockHTTPResponse{
		headers: make(map[string]string),
	}
}

// NewMockHTTPResponseWithErrors creates a mock response that can simulate write/JSON errors
func NewMockHTTPResponseWithErrors(writeErr, jsonErr error) *MockHTTPResponse {
	return &MockHTTPResponse{
		headers:  make(map[string]string),
		writeErr: writeErr,
		jsonErr:  jsonErr,
	}
}

func (m *MockHTTPResponse) SetHeader(key, value string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.headers[key] = value
}
func (m *MockHTTPResponse) SetStatus(code int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.statusCode = code
}
func (m *MockHTTPResponse) Write(data []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.data = data
	return m.writeErr
}
func (m *MockHTTPResponse) JSON(data interface{}) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.data = data
	return m.jsonErr
}

// GetHeader returns a header value thread-safely
func (m *MockHTTPResponse) GetHeader(key string) string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.headers[key]
}

// GetStatusCode returns the status code thread-safely
func (m *MockHTTPResponse) GetStatusCode() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.statusCode
}

// GetData returns the response data thread-safely
func (m *MockHTTPResponse) GetData() interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.data
}

type MockHTTPContext struct {
	request    HTTPRequest
	response   HTTPResponse
	values     map[string]interface{}
	nextErr    error
	nextCalled bool
	mu         sync.RWMutex
}

func NewMockHTTPContext(req HTTPRequest, resp HTTPResponse) *MockHTTPContext {
	return &MockHTTPContext{
		request:  req,
		response: resp,
		values:   make(map[string]interface{}),
	}
}

func (m *MockHTTPContext) Request() HTTPRequest        { return m.request }
func (m *MockHTTPContext) Response() HTTPResponse     { return m.response }
func (m *MockHTTPContext) Set(key string, value interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.values[key] = value
}
func (m *MockHTTPContext) Get(key string) (interface{}, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	val, ok := m.values[key]
	return val, ok
}
func (m *MockHTTPContext) Next() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.nextCalled = true
	return m.nextErr
}

// WasNextCalled returns whether Next() was called
func (m *MockHTTPContext) WasNextCalled() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.nextCalled
}

// SetNextError sets the error to return from Next()
func (m *MockHTTPContext) SetNextError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.nextErr = err
}

type MockLogger struct {
	mock.Mock
}

func (m *MockLogger) Debug(ctx context.Context, msg string, fields ...Field) {
	m.Called(ctx, msg, fields)
}

func (m *MockLogger) Info(ctx context.Context, msg string, fields ...Field) {
	m.Called(ctx, msg, fields)
}

func (m *MockLogger) Warn(ctx context.Context, msg string, fields ...Field) {
	m.Called(ctx, msg, fields)
}

func (m *MockLogger) Error(ctx context.Context, msg string, err error, fields ...Field) {
	m.Called(ctx, msg, err, fields)
}

type MockAuditLogger struct {
	mock.Mock
}

func (m *MockAuditLogger) LogRequest(ctx context.Context, entry *AuditEntry) error {
	args := m.Called(ctx, entry)
	return args.Error(0)
}

type MockRateLimiter struct {
	mock.Mock
}

func (m *MockRateLimiter) Allow(ctx context.Context, key string) (*RateLimitResult, error) {
	args := m.Called(ctx, key)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*RateLimitResult), args.Error(1)
}

func (m *MockRateLimiter) Reset(ctx context.Context, key string) error {
	args := m.Called(ctx, key)
	return args.Error(0)
}

// MockRequestIDGenerator provides a testable implementation of RequestIDGenerator
type MockRequestIDGenerator struct {
	mock.Mock
}

func (m *MockRequestIDGenerator) Generate() string {
	args := m.Called()
	return args.String(0)
}

// MockErrorHandler provides a testable implementation of ErrorHandler
type MockErrorHandler struct {
	mock.Mock
}

func (m *MockErrorHandler) HandleError(ctx HTTPContext, err error) error {
	args := m.Called(ctx, err)
	return args.Error(0)
}

// Test RequestIDMiddleware
func TestRequestIDMiddleware(t *testing.T) {
	t.Run("generates request ID when none exists", func(t *testing.T) {
		// Arrange
		req := &MockHTTPRequest{
			headers: make(map[string]string),
		}
		resp := NewMockHTTPResponse()
		ctx := NewMockHTTPContext(req, resp)

		config := RequestIDConfig{
			Generator: &UUIDGenerator{},
			Header:    "X-Request-ID",
		}
		middleware := NewRequestIDMiddleware(config)

		// Act
		err := middleware.Handle(ctx)

		// Assert
		assert.NoError(t, err)
		assert.NotEmpty(t, resp.GetHeader("X-Request-ID"))

		requestID, exists := ctx.Get("requestID")
		assert.True(t, exists)
		assert.Equal(t, resp.GetHeader("X-Request-ID"), requestID)
	})

	t.Run("preserves existing request ID", func(t *testing.T) {
		// Arrange
		existingID := "existing-request-id"
		req := &MockHTTPRequest{
			headers: map[string]string{
				"X-Request-ID": existingID,
			},
		}
		resp := NewMockHTTPResponse()
		ctx := NewMockHTTPContext(req, resp)

		config := RequestIDConfig{
			Generator: &UUIDGenerator{},
			Header:    "X-Request-ID",
		}
		middleware := NewRequestIDMiddleware(config)

		// Act
		err := middleware.Handle(ctx)

		// Assert
		assert.NoError(t, err)

		requestID, exists := ctx.Get("requestID")
		assert.True(t, exists)
		assert.Equal(t, existingID, requestID)
	})
}

// Test RateLimitMiddleware
func TestRateLimitMiddleware(t *testing.T) {
	t.Run("allows request when under limit", func(t *testing.T) {
		// Arrange
		mockLimiter := &MockRateLimiter{}
		result := &RateLimitResult{
			Allowed:   true,
			Limit:     100,
			Remaining: 99,
			ResetTime: time.Now().Add(time.Minute),
		}
		mockLimiter.On("Allow", mock.Anything, "192.168.1.1").Return(result, nil)

		req := &MockHTTPRequest{
			ip:   "192.168.1.1",
			path: "/api/test",
		}
		resp := NewMockHTTPResponse()
		ctx := NewMockHTTPContext(req, resp)

		config := RateLimitConfig{
			Limiter: mockLimiter,
		}
		middleware := NewRateLimitMiddleware(config)

		// Act
		err := middleware.Handle(ctx)

		// Assert
		assert.NoError(t, err)
		assert.Equal(t, "100", resp.GetHeader("X-RateLimit-Limit"))
		assert.Equal(t, "99", resp.GetHeader("X-RateLimit-Remaining"))
		mockLimiter.AssertExpectations(t)
	})

	t.Run("blocks request when over limit", func(t *testing.T) {
		// Arrange
		mockLimiter := &MockRateLimiter{}
		result := &RateLimitResult{
			Allowed:    false,
			Limit:      100,
			Remaining:  0,
			ResetTime:  time.Now().Add(time.Minute),
			RetryAfter: time.Minute,
		}
		mockLimiter.On("Allow", mock.Anything, "192.168.1.1").Return(result, nil)

		req := &MockHTTPRequest{
			ip:   "192.168.1.1",
			path: "/api/test",
		}
		resp := NewMockHTTPResponse()
		ctx := NewMockHTTPContext(req, resp)

		config := RateLimitConfig{
			Limiter: mockLimiter,
		}
		middleware := NewRateLimitMiddleware(config)

		// Act
		err := middleware.Handle(ctx)

		// Assert
		assert.NoError(t, err)
		assert.Equal(t, 429, resp.GetStatusCode())
		assert.Equal(t, "60", resp.GetHeader("Retry-After"))
		mockLimiter.AssertExpectations(t)
	})

	t.Run("skips rate limiting for configured paths", func(t *testing.T) {
		// Arrange
		req := &MockHTTPRequest{
			ip:   "192.168.1.1",
			path: "/api/v1/health",
		}
		resp := NewMockHTTPResponse()
		ctx := NewMockHTTPContext(req, resp)

		config := RateLimitConfig{
			Limiter:   &MockRateLimiter{}, // Should not be called
			SkipPaths: []string{"/api/v1/health"},
		}
		middleware := NewRateLimitMiddleware(config)

		// Act
		err := middleware.Handle(ctx)

		// Assert
		assert.NoError(t, err)
		// No rate limit headers should be set
		assert.Empty(t, resp.GetHeader("X-RateLimit-Limit"))
	})
}

// Test LoggingMiddleware
func TestLoggingMiddleware(t *testing.T) {
	t.Run("logs request and calls audit logger", func(t *testing.T) {
		// Arrange
		mockLogger := &MockLogger{}
		mockAuditLogger := &MockAuditLogger{}

		mockLogger.On("Info", mock.Anything, "Request started", mock.Anything).Return()
		mockLogger.On("Info", mock.Anything, "Request completed", mock.Anything).Return()
		mockAuditLogger.On("LogRequest", mock.Anything, mock.AnythingOfType("*middleware.AuditEntry")).Return(nil)

		req := &MockHTTPRequest{
			method:    "GET",
			path:      "/api/test",
			ip:        "192.168.1.1",
			userAgent: "test-agent",
			headers:   map[string]string{"X-User": "testuser"},
		}
		resp := NewMockHTTPResponse()
		ctx := NewMockHTTPContext(req, resp)
		ctx.Set("requestID", "test-request-id")

		config := LoggingConfig{
			Logger:      mockLogger,
			AuditLogger: mockAuditLogger,
		}
		middleware := NewLoggingMiddleware(config)

		// Act
		err := middleware.Handle(ctx)

		// Assert
		assert.NoError(t, err)
		mockLogger.AssertExpectations(t)

		// Give audit logger goroutine time to complete
		time.Sleep(10 * time.Millisecond)
		mockAuditLogger.AssertExpectations(t)
	})

	t.Run("skips logging for configured paths", func(t *testing.T) {
		// Arrange
		mockLogger := &MockLogger{}

		req := &MockHTTPRequest{
			path: "/api/v1/health",
		}
		resp := NewMockHTTPResponse()
		ctx := NewMockHTTPContext(req, resp)

		config := LoggingConfig{
			Logger:    mockLogger,
			SkipPaths: []string{"/api/v1/health"},
		}
		middleware := NewLoggingMiddleware(config)

		// Act
		err := middleware.Handle(ctx)

		// Assert
		assert.NoError(t, err)
		// No logging methods should be called
		mockLogger.AssertNotCalled(t, "Info")
		mockLogger.AssertNotCalled(t, "Error")
	})
}

// Test ErrorHandler
func TestDefaultErrorHandler(t *testing.T) {
	t.Run("handles app error correctly", func(t *testing.T) {
		// Arrange
		mockLogger := &MockLogger{}
		mockLogger.On("Warn", mock.Anything, "Client error", mock.Anything).Return()

		config := ErrorHandlerConfig{
			Logger:            mockLogger,
			IncludeStackTrace: false,
			MaskInternalErrs:  false,
		}
		handler := NewDefaultErrorHandler(config)

		req := &MockHTTPRequest{
			method: "GET",
			path:   "/api/test",
			ip:     "192.168.1.1",
		}
		resp := NewMockHTTPResponse()
		ctx := NewMockHTTPContext(req, resp)
		ctx.Set("requestID", "test-request-id")

		appErr := &AppError{
			Type:       ErrorTypeValidation,
			Code:       "VALIDATION_FAILED",
			Message:    "Invalid input",
			StatusCode: 400,
			Timestamp:  time.Now(),
		}

		// Act
		err := handler.HandleError(ctx, appErr)

		// Assert
		assert.NoError(t, err)
		assert.Equal(t, 400, resp.GetStatusCode())

		errorResp, ok := resp.GetData().(*ErrorResponse)
		require.True(t, ok)
		assert.Equal(t, "Invalid input", errorResp.Error)
		assert.Equal(t, ErrorTypeValidation, errorResp.Type)
		assert.Equal(t, "test-request-id", errorResp.RequestID)

		mockLogger.AssertExpectations(t)
	})

	t.Run("masks internal errors in production", func(t *testing.T) {
		// Arrange
		config := ErrorHandlerConfig{
			IncludeStackTrace: false,
			MaskInternalErrs:  true, // Production mode
		}
		handler := NewDefaultErrorHandler(config)

		req := &MockHTTPRequest{
			method: "GET",
			path:   "/api/test",
		}
		resp := NewMockHTTPResponse()
		ctx := NewMockHTTPContext(req, resp)

		internalErr := errors.New("database connection failed")

		// Act
		err := handler.HandleError(ctx, internalErr)

		// Assert
		assert.NoError(t, err)
		assert.Equal(t, 500, resp.GetStatusCode())

		errorResp, ok := resp.GetData().(*ErrorResponse)
		require.True(t, ok)
		assert.Equal(t, "Internal server error", errorResp.Error)
		assert.Equal(t, "An unexpected error occurred", errorResp.Message)
		assert.Empty(t, errorResp.Details) // Should be masked
	})
}

// Test InMemoryRateLimiter
func TestInMemoryRateLimiter(t *testing.T) {
	t.Run("allows requests under limit", func(t *testing.T) {
		// Arrange
		config := InMemoryRateLimiterConfig{
			Limit:  2,
			Window: time.Minute,
		}
		limiter := NewInMemoryRateLimiter(config)

		// Act & Assert
		result1, err := limiter.Allow(context.Background(), "test-key")
		assert.NoError(t, err)
		assert.True(t, result1.Allowed)
		assert.Equal(t, 1, result1.Remaining)

		result2, err := limiter.Allow(context.Background(), "test-key")
		assert.NoError(t, err)
		assert.True(t, result2.Allowed)
		assert.Equal(t, 0, result2.Remaining)

		result3, err := limiter.Allow(context.Background(), "test-key")
		assert.NoError(t, err)
		assert.False(t, result3.Allowed)
		assert.Equal(t, 0, result3.Remaining)
	})

	t.Run("resets counter after window", func(t *testing.T) {
		// Arrange
		config := InMemoryRateLimiterConfig{
			Limit:  1,
			Window: 50 * time.Millisecond,
		}
		limiter := NewInMemoryRateLimiter(config)

		// Act & Assert
		result1, err := limiter.Allow(context.Background(), "test-key")
		assert.NoError(t, err)
		assert.True(t, result1.Allowed)

		result2, err := limiter.Allow(context.Background(), "test-key")
		assert.NoError(t, err)
		assert.False(t, result2.Allowed)

		// Wait for window to expire
		time.Sleep(100 * time.Millisecond)

		result3, err := limiter.Allow(context.Background(), "test-key")
		assert.NoError(t, err)
		assert.True(t, result3.Allowed)
	})
}