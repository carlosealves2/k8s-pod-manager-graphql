package middleware

import (
	"log/slog"
	"time"

	"github.com/carlosf/k8s-pod-manager/internal/database"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// ExampleIntegration demonstrates how to integrate the new middleware with Gin
func ExampleIntegration() {
	// Example 1: Basic setup with default middleware stack
	basicSetup()

	// Example 2: Custom middleware configuration
	customSetup()

	// Example 3: Development vs Production configurations
	environmentSpecificSetup()
}

// basicSetup shows the simplest way to add middleware to a Gin router
func basicSetup() {
	router := gin.New()

	// Get database connection (assumes database.DB is available)
	var db *gorm.DB = database.DB

	// Use default middleware stack
	middlewares := DefaultMiddlewareStack(db, true) // Skip health path logging

	// Convert and add each middleware to Gin
	for _, mw := range middlewares {
		router.Use(ToGinHandler(mw))
	}

	// Your routes here...
	router.GET("/api/test", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "Hello World"})
	})
}

// customSetup shows how to build a custom middleware stack
func customSetup() {
	router := gin.New()

	// Build custom middleware stack
	middlewares := NewMiddlewareBuilder().
		WithSlogLogger(slog.LevelInfo).
		WithDatabaseAuditLogger(database.DB).
		WithInMemoryRateLimiter(50, time.Minute).
		WithDefaultErrorHandler(false, true).
		AddRequestID().
		AddRecovery(false).
		AddErrorHandling().
		AddLogging("/api/v1/health", "/api/v1/ready").
		AddRateLimit("/api/v1/health").
		Build()

	// Add to Gin router
	for _, mw := range middlewares {
		router.Use(ToGinHandler(mw))
	}

	// Your routes here...
}

// environmentSpecificSetup shows different configurations for development and production
func environmentSpecificSetup() {
	// Development setup
	devRouter := gin.New()
	devMiddlewares := DevelopmentMiddlewareStack(database.DB)
	for _, mw := range devMiddlewares {
		devRouter.Use(ToGinHandler(mw))
	}

	// Production setup
	prodRouter := gin.New()
	prodMiddlewares := ProductionMiddlewareStack(database.DB)
	for _, mw := range prodMiddlewares {
		prodRouter.Use(ToGinHandler(mw))
	}
}

// MigrationExample shows how to migrate from the old Fiber-based middleware
func MigrationExample() {
	router := gin.New()

	// OLD WAY (Fiber-based - won't work with Gin):
	// router.Use(middleware.ErrorHandler()) // This would fail
	// router.Use(middleware.RateLimiter(100))
	// router.Use(middleware.Logger())
	// router.Use(middleware.Recovery())
	// router.Use(middleware.RequestID())

	// NEW WAY (Framework-agnostic):
	middlewares := NewMiddlewareBuilder().
		WithSlogLogger(slog.LevelInfo).
		WithDatabaseAuditLogger(database.DB).
		WithInMemoryRateLimiter(100, time.Minute).
		WithDefaultErrorHandler(false, true).
		AddRequestID().
		AddRecovery(false).
		AddErrorHandling().
		AddLogging().
		AddRateLimit().
		Build()

	for _, mw := range middlewares {
		router.Use(ToGinHandler(mw))
	}
}

// CustomMiddlewareExample shows how to create custom middleware using the interfaces
func CustomMiddlewareExample() {
	// Custom middleware that adds a custom header
	customMiddleware := MiddlewareFunc(func(ctx HTTPContext) error {
		ctx.Response().SetHeader("X-Custom-Header", "MyValue")
		return ctx.Next()
	})

	// Custom rate limiting key function (e.g., rate limit by user ID)
	userBasedRateLimiter := func(ctx HTTPContext) string {
		userID := ctx.Request().Header("X-User-ID")
		if userID == "" {
			return ctx.Request().IP() // Fallback to IP
		}
		return "user:" + userID
	}

	// Build middleware stack with custom components
	middlewares := NewMiddlewareBuilder().
		WithSlogLogger(slog.LevelInfo).
		WithDatabaseAuditLogger(database.DB).
		WithInMemoryRateLimiter(10, time.Minute).
		WithDefaultErrorHandler(false, true).
		AddRequestID().
		AddCustom(customMiddleware). // Add custom middleware
		AddRecovery(false).
		AddErrorHandling().
		AddLogging().
		Build()

	// Add custom rate limiter with user-based key
	rateLimitConfig := RateLimitConfig{
		Limiter: NewInMemoryRateLimiter(InMemoryRateLimiterConfig{
			Limit:  10,
			Window: time.Minute,
		}),
		KeyFunc: userBasedRateLimiter,
	}
	customRateLimit := NewRateLimitMiddleware(rateLimitConfig)
	middlewares = append(middlewares, customRateLimit)

	// Use with Gin
	router := gin.New()
	for _, mw := range middlewares {
		router.Use(ToGinHandler(mw))
	}
}

// TestingExample shows how to test handlers with middleware
func TestingExample() {
	// This would be in a test file

	// Create mock dependencies
	mockLogger := &MockLogger{}
	mockAuditLogger := &MockAuditLogger{}

	// Set up expectations
	// mockLogger.On("Info", ...).Return()
	// mockAuditLogger.On("LogRequest", ...).Return(nil)

	// Build testable middleware stack
	middlewares := NewMiddlewareBuilder().
		WithLogger(mockLogger).
		WithAuditLogger(mockAuditLogger).
		WithInMemoryRateLimiter(100, time.Minute).
		WithDefaultErrorHandler(true, false). // Include details for testing
		AddRequestID().
		AddLogging().
		Build()

	// Create test router
	router := gin.New()
	for _, mw := range middlewares {
		router.Use(ToGinHandler(mw))
	}

	// Add test route
	router.GET("/test", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "test"})
	})

	// Test with httptest.ResponseRecorder
	// w := httptest.NewRecorder()
	// req, _ := http.NewRequest("GET", "/test", nil)
	// router.ServeHTTP(w, req)

	// Assert expectations
	// mockLogger.AssertExpectations(t)
	// mockAuditLogger.AssertExpectations(t)
}