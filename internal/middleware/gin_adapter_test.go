package middleware

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() {
	// Set Gin to test mode to reduce noise in test output
	gin.SetMode(gin.TestMode)
}

// TestGinHTTPRequest tests the Gin request adapter
func TestGinHTTPRequest(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		path           string
		headers        map[string]string
		body           string
		expectedMethod string
		expectedPath   string
		expectedIP     string
	}{
		{
			name:           "GET request with headers",
			method:         "GET",
			path:           "/api/v1/test",
			headers:        map[string]string{"Authorization": "Bearer token", "X-Request-ID": "test-id"},
			expectedMethod: "GET",
			expectedPath:   "/api/v1/test",
		},
		{
			name:           "POST request with body",
			method:         "POST",
			path:           "/api/v1/create",
			headers:        map[string]string{"Content-Type": "application/json"},
			body:           `{"test": "data"}`,
			expectedMethod: "POST",
			expectedPath:   "/api/v1/create",
		},
		{
			name:           "request with user agent",
			method:         "PUT",
			path:           "/api/v1/update",
			headers:        map[string]string{"User-Agent": "test-client/1.0"},
			expectedMethod: "PUT",
			expectedPath:   "/api/v1/update",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a Gin context with the test request
			gin.SetMode(gin.TestMode)
			router := gin.New()

			var capturedRequest *GinHTTPRequest
			router.Any(tt.path, func(c *gin.Context) {
				capturedRequest = &GinHTTPRequest{ctx: c}
				c.Status(http.StatusOK)
			})

			// Create HTTP request
			var body *bytes.Buffer
			if tt.body != "" {
				body = bytes.NewBufferString(tt.body)
			} else {
				body = bytes.NewBuffer(nil)
			}

			req := httptest.NewRequest(tt.method, tt.path, body)
			for k, v := range tt.headers {
				req.Header.Set(k, v)
			}

			// Execute request
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			// Test the adapter methods
			require.NotNil(t, capturedRequest)
			assert.Equal(t, tt.expectedMethod, capturedRequest.Method())
			assert.Equal(t, tt.expectedPath, capturedRequest.Path())

			// Test headers
			for k, v := range tt.headers {
				assert.Equal(t, v, capturedRequest.Header(k))
			}

			// Test other methods
			assert.NotEmpty(t, capturedRequest.IP()) // Should have some IP (even if localhost)
			if userAgent, exists := tt.headers["User-Agent"]; exists {
				assert.Equal(t, userAgent, capturedRequest.UserAgent())
			}
			if requestID, exists := tt.headers["X-Request-ID"]; exists {
				assert.Equal(t, requestID, capturedRequest.RequestID())
			}

			// Test body (note: Gin reads body once, so this tests the caching mechanism)
			if tt.body != "" {
				// For this test, we can't easily read the body after Gin has processed it
				// But we can verify the method doesn't panic
				assert.NotPanics(t, func() {
					capturedRequest.Body()
				})
			}
		})
	}
}

// TestGinHTTPResponse tests the Gin response adapter
func TestGinHTTPResponse(t *testing.T) {
	tests := []struct {
		name           string
		setHeaders     map[string]string
		statusCode     int
		jsonData       interface{}
		writeData      []byte
		expectError    bool
	}{
		{
			name:       "set headers and status",
			setHeaders: map[string]string{"X-Custom": "value", "Content-Type": "application/json"},
			statusCode: 201,
		},
		{
			name:      "write JSON response",
			statusCode: 200,
			jsonData:  map[string]string{"message": "success"},
		},
		{
			name:      "write raw data",
			statusCode: 200,
			writeData: []byte("raw response data"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			router := gin.New()

			var capturedResponse *GinHTTPResponse
			router.GET("/test", func(c *gin.Context) {
				capturedResponse = &GinHTTPResponse{ctx: c}

				// Set headers
				for k, v := range tt.setHeaders {
					capturedResponse.SetHeader(k, v)
				}

				// Set status
				if tt.statusCode != 0 {
					capturedResponse.SetStatus(tt.statusCode)
				}

				// Write JSON or raw data
				if tt.jsonData != nil {
					err := capturedResponse.JSON(tt.jsonData)
					assert.NoError(t, err)
				} else if tt.writeData != nil {
					err := capturedResponse.Write(tt.writeData)
					assert.NoError(t, err)
				}
			})

			// Execute request
			req := httptest.NewRequest("GET", "/test", nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			// Verify response
			require.NotNil(t, capturedResponse)

			// Check headers
			for k, v := range tt.setHeaders {
				assert.Equal(t, v, w.Header().Get(k))
			}

			// Check status code
			if tt.statusCode != 0 {
				assert.Equal(t, tt.statusCode, w.Code)
			}

			// Check response body
			if tt.jsonData != nil {
				var responseData interface{}
				err := json.Unmarshal(w.Body.Bytes(), &responseData)
				assert.NoError(t, err)
			} else if tt.writeData != nil {
				assert.Equal(t, string(tt.writeData), w.Body.String())
			}
		})
	}
}

// TestGinHTTPContext tests the complete Gin context adapter
func TestGinHTTPContext(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	var capturedContext HTTPContext
	router.POST("/test", func(c *gin.Context) {
		capturedContext = NewGinHTTPContext(c)

		// Test setting and getting values
		capturedContext.Set("test-key", "test-value")
		capturedContext.Set("number", 42)

		// Test Next() call
		err := capturedContext.Next()
		assert.NoError(t, err)

		c.Status(http.StatusOK)
	})

	// Create request with body and headers
	body := `{"data": "test"}`
	req := httptest.NewRequest("POST", "/test", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Request-ID", "test-request-id")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Verify context functionality
	require.NotNil(t, capturedContext)

	// Test Request() method
	request := capturedContext.Request()
	assert.Equal(t, "POST", request.Method())
	assert.Equal(t, "/test", request.Path())
	assert.Equal(t, "application/json", request.Header("Content-Type"))
	assert.Equal(t, "test-request-id", request.RequestID())

	// Test Response() method
	response := capturedContext.Response()
	assert.NotNil(t, response)

	// Test Get/Set methods
	value, exists := capturedContext.Get("test-key")
	assert.True(t, exists)
	assert.Equal(t, "test-value", value)

	numberValue, exists := capturedContext.Get("number")
	assert.True(t, exists)
	assert.Equal(t, 42, numberValue)

	// Test non-existent key
	_, exists = capturedContext.Get("non-existent")
	assert.False(t, exists)
}

// TestToGinHandler tests converting middleware to Gin handler
func TestToGinHandler(t *testing.T) {
	tests := []struct {
		name               string
		middleware         Middleware
		expectError        bool
		expectedStatus     int
		expectedResponse   map[string]interface{}
	}{
		{
			name: "successful middleware",
			middleware: MiddlewareFunc(func(ctx HTTPContext) error {
				ctx.Set("middleware-called", true)
				return ctx.Next()
			}),
			expectError:    false,
			expectedStatus: http.StatusOK,
		},
		{
			name: "middleware with error",
			middleware: MiddlewareFunc(func(ctx HTTPContext) error {
				return errors.New("middleware error")
			}),
			expectError:    true,
			expectedStatus: http.StatusInternalServerError,
			expectedResponse: map[string]interface{}{
				"error":   "Middleware error",
				"details": "middleware error",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			router := gin.New()

			// Convert middleware to Gin handler
			ginHandler := ToGinHandler(tt.middleware)
			router.Use(ginHandler)

			// Add a test route
			router.GET("/test", func(c *gin.Context) {
				// Check if middleware was called successfully
				if !tt.expectError {
					value, exists := c.Get("middleware-called")
					assert.True(t, exists)
					assert.True(t, value.(bool))
				}
				c.JSON(http.StatusOK, gin.H{"success": true})
			})

			// Execute request
			req := httptest.NewRequest("GET", "/test", nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			// Verify response
			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectError {
				var response map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedResponse["error"], response["error"])
				assert.Equal(t, tt.expectedResponse["details"], response["details"])
			}
		})
	}
}

// TestGinMiddlewareAdapter tests the adapter for Gin middleware
func TestGinMiddlewareAdapter(t *testing.T) {
	tests := []struct {
		name            string
		ginMiddleware   gin.HandlerFunc
		expectError     bool
		contextType     HTTPContext
	}{
		{
			name: "valid Gin middleware",
			ginMiddleware: func(c *gin.Context) {
				c.Set("gin-middleware-called", true)
				c.Next()
			},
			expectError: false,
			contextType: &GinHTTPContext{},
		},
		{
			name: "incompatible context type",
			ginMiddleware: func(c *gin.Context) {
				c.Next()
			},
			expectError: true,
			contextType: &MockHTTPContext{}, // Not a GinHTTPContext
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			adapter := NewGinMiddlewareAdapter(tt.ginMiddleware)

			if tt.expectError {
				// Test with incompatible context
				err := adapter.Handle(tt.contextType)
				assert.Error(t, err)
				assert.Equal(t, ErrIncompatibleContext, err)
			} else {
				gin.SetMode(gin.TestMode)
				router := gin.New()

				router.GET("/test", func(c *gin.Context) {
					ginCtx := NewGinHTTPContext(c)
					err := adapter.Handle(ginCtx)
					assert.NoError(t, err)

					// Verify the Gin middleware was called
					value, exists := c.Get("gin-middleware-called")
					assert.True(t, exists)
					assert.True(t, value.(bool))

					c.JSON(http.StatusOK, gin.H{"success": true})
				})

				req := httptest.NewRequest("GET", "/test", nil)
				w := httptest.NewRecorder()
				router.ServeHTTP(w, req)

				assert.Equal(t, http.StatusOK, w.Code)
			}
		})
	}
}

// TestMiddlewareError tests the MiddlewareError type
func TestMiddlewareError(t *testing.T) {
	tests := []struct {
		name     string
		err      *MiddlewareError
		expected string
	}{
		{
			name: "error without cause",
			err: &MiddlewareError{
				Code:    "TEST_ERROR",
				Message: "Test error message",
			},
			expected: "Test error message",
		},
		{
			name: "error with cause",
			err: &MiddlewareError{
				Code:    "TEST_ERROR",
				Message: "Test error message",
				Cause:   errors.New("root cause"),
			},
			expected: "Test error message: root cause",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.err.Error())

			// Test Unwrap
			if tt.err.Cause != nil {
				assert.Equal(t, tt.err.Cause, tt.err.Unwrap())
			} else {
				assert.Nil(t, tt.err.Unwrap())
			}
		})
	}
}

// TestErrIncompatibleContext tests the predefined error
func TestErrIncompatibleContext(t *testing.T) {
	assert.Equal(t, "INCOMPATIBLE_CONTEXT", ErrIncompatibleContext.Code)
	assert.Equal(t, "Context type is not compatible with this adapter", ErrIncompatibleContext.Message)
	assert.Nil(t, ErrIncompatibleContext.Cause)
}

// BenchmarkGinHTTPContext benchmarks the Gin context adapter
func BenchmarkGinHTTPContext(b *testing.B) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	router.GET("/benchmark", func(c *gin.Context) {
		httpCtx := NewGinHTTPContext(c)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			// Simulate typical middleware operations
			httpCtx.Set("key", "value")
			_, _ = httpCtx.Get("key")
			_ = httpCtx.Request().Method()
			_ = httpCtx.Request().Path()
		}
	})

	req := httptest.NewRequest("GET", "/benchmark", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
}

// BenchmarkToGinHandler benchmarks the middleware to Gin handler conversion
func BenchmarkToGinHandler(b *testing.B) {
	middleware := MiddlewareFunc(func(ctx HTTPContext) error {
		ctx.Set("test", "value")
		return ctx.Next()
	})

	ginHandler := ToGinHandler(middleware)
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(ginHandler)
	router.GET("/benchmark", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/benchmark", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
	}
}