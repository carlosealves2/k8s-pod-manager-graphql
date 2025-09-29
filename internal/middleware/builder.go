package middleware

import (
	"log/slog"
	"time"

	"gorm.io/gorm"
)

// MiddlewareBuilder provides a fluent interface for building middleware chains
type MiddlewareBuilder struct {
	middlewares []Middleware
	logger      Logger
	auditLogger AuditLogger
	rateLimiter RateLimiter
	errorHandler ErrorHandler
}

// NewMiddlewareBuilder creates a new middleware builder
func NewMiddlewareBuilder() *MiddlewareBuilder {
	return &MiddlewareBuilder{
		middlewares: make([]Middleware, 0),
	}
}

// WithLogger sets the logger for the middleware chain
func (b *MiddlewareBuilder) WithLogger(logger Logger) *MiddlewareBuilder {
	b.logger = logger
	return b
}

// WithSlogLogger sets a structured logger using slog
func (b *MiddlewareBuilder) WithSlogLogger(level slog.Level) *MiddlewareBuilder {
	b.logger = NewSlogLogger(level)
	return b
}

// WithAuditLogger sets the audit logger for the middleware chain
func (b *MiddlewareBuilder) WithAuditLogger(auditLogger AuditLogger) *MiddlewareBuilder {
	b.auditLogger = auditLogger
	return b
}

// WithDatabaseAuditLogger sets a database audit logger
func (b *MiddlewareBuilder) WithDatabaseAuditLogger(db *gorm.DB) *MiddlewareBuilder {
	config := DatabaseAuditLoggerConfig{
		DB:     db,
		Logger: b.logger,
	}
	b.auditLogger = NewDatabaseAuditLogger(config)
	return b
}

// WithJSONAuditLogger sets a JSON audit logger
func (b *MiddlewareBuilder) WithJSONAuditLogger() *MiddlewareBuilder {
	b.auditLogger = NewJSONAuditLogger(b.logger)
	return b
}

// WithCompositeAuditLogger sets multiple audit loggers
func (b *MiddlewareBuilder) WithCompositeAuditLogger(loggers ...AuditLogger) *MiddlewareBuilder {
	b.auditLogger = NewCompositeAuditLogger(loggers...)
	return b
}

// WithRateLimiter sets the rate limiter for the middleware chain
func (b *MiddlewareBuilder) WithRateLimiter(rateLimiter RateLimiter) *MiddlewareBuilder {
	b.rateLimiter = rateLimiter
	return b
}

// WithInMemoryRateLimiter sets an in-memory rate limiter
func (b *MiddlewareBuilder) WithInMemoryRateLimiter(limit int, window time.Duration) *MiddlewareBuilder {
	config := InMemoryRateLimiterConfig{
		Limit:  limit,
		Window: window,
	}
	b.rateLimiter = NewInMemoryRateLimiter(config)
	return b
}

// WithErrorHandler sets the error handler for the middleware chain
func (b *MiddlewareBuilder) WithErrorHandler(errorHandler ErrorHandler) *MiddlewareBuilder {
	b.errorHandler = errorHandler
	return b
}

// WithDefaultErrorHandler sets a default error handler
func (b *MiddlewareBuilder) WithDefaultErrorHandler(includeStackTrace, maskInternalErrs bool) *MiddlewareBuilder {
	config := ErrorHandlerConfig{
		Logger:            b.logger,
		IncludeStackTrace: includeStackTrace,
		MaskInternalErrs:  maskInternalErrs,
	}
	b.errorHandler = NewDefaultErrorHandler(config)
	return b
}

// AddRequestID adds request ID middleware to the chain
func (b *MiddlewareBuilder) AddRequestID() *MiddlewareBuilder {
	config := RequestIDConfig{
		Generator: &UUIDGenerator{},
		Header:    "X-Request-ID",
	}
	b.middlewares = append(b.middlewares, NewRequestIDMiddleware(config))
	return b
}

// AddLogging adds logging middleware to the chain
func (b *MiddlewareBuilder) AddLogging(skipPaths ...string) *MiddlewareBuilder {
	config := LoggingConfig{
		Logger:      b.logger,
		AuditLogger: b.auditLogger,
		SkipPaths:   skipPaths,
	}
	b.middlewares = append(b.middlewares, NewLoggingMiddleware(config))
	return b
}

// AddRateLimit adds rate limiting middleware to the chain
func (b *MiddlewareBuilder) AddRateLimit(skipPaths ...string) *MiddlewareBuilder {
	if b.rateLimiter == nil {
		// Default rate limiter if none specified
		b.WithInMemoryRateLimiter(100, time.Minute)
	}

	config := RateLimitConfig{
		Limiter:   b.rateLimiter,
		SkipPaths: skipPaths,
		Logger:    b.logger,
	}
	b.middlewares = append(b.middlewares, NewRateLimitMiddleware(config))
	return b
}

// AddRecovery adds panic recovery middleware to the chain
func (b *MiddlewareBuilder) AddRecovery(includeStackTrace bool) *MiddlewareBuilder {
	config := RecoveryConfig{
		ErrorHandler:      b.errorHandler,
		Logger:            b.logger,
		IncludeStackTrace: includeStackTrace,
	}
	b.middlewares = append(b.middlewares, NewRecoveryMiddleware(config))
	return b
}

// AddErrorHandling adds error handling middleware to the chain
func (b *MiddlewareBuilder) AddErrorHandling() *MiddlewareBuilder {
	if b.errorHandler == nil {
		// Default error handler if none specified
		b.WithDefaultErrorHandler(false, true)
	}

	b.middlewares = append(b.middlewares, NewErrorMiddleware(b.errorHandler))
	return b
}

// AddCustom adds a custom middleware to the chain
func (b *MiddlewareBuilder) AddCustom(middleware Middleware) *MiddlewareBuilder {
	b.middlewares = append(b.middlewares, middleware)
	return b
}

// Build returns the complete middleware chain
func (b *MiddlewareBuilder) Build() []Middleware {
	return b.middlewares
}

// BuildChain returns a single middleware that executes all middlewares in order
func (b *MiddlewareBuilder) BuildChain() Middleware {
	middlewares := b.middlewares
	return MiddlewareFunc(func(ctx HTTPContext) error {
		return executeChain(ctx, middlewares)
	})
}

// executeChain executes a chain of middlewares
func executeChain(ctx HTTPContext, middlewares []Middleware) error {
	if len(middlewares) == 0 {
		return ctx.Next()
	}

	// Create a new context that continues with the next middleware in the chain
	chainCtx := &chainHTTPContext{
		HTTPContext: ctx,
		middlewares: middlewares[1:],
		index:       0,
	}

	return middlewares[0].Handle(chainCtx)
}

// chainHTTPContext wraps HTTPContext to handle middleware chaining
type chainHTTPContext struct {
	HTTPContext
	middlewares []Middleware
	index       int
}

func (c *chainHTTPContext) Next() error {
	if c.index < len(c.middlewares) {
		middleware := c.middlewares[c.index]
		c.index++
		return middleware.Handle(c)
	}
	return c.HTTPContext.Next()
}

// DefaultMiddlewareStack creates a default middleware stack for common use cases
func DefaultMiddlewareStack(db *gorm.DB, skipHealthPaths bool) []Middleware {
	builder := NewMiddlewareBuilder().
		WithSlogLogger(slog.LevelInfo).
		WithDatabaseAuditLogger(db).
		WithInMemoryRateLimiter(100, time.Minute).
		WithDefaultErrorHandler(false, true).
		AddRequestID().
		AddRecovery(false).
		AddErrorHandling()

	if skipHealthPaths {
		builder.AddLogging("/api/v1/health", "/api/v1/ready").
			AddRateLimit("/api/v1/health", "/api/v1/ready")
	} else {
		builder.AddLogging().
			AddRateLimit()
	}

	return builder.Build()
}

// DevelopmentMiddlewareStack creates a middleware stack suitable for development
func DevelopmentMiddlewareStack(db *gorm.DB) []Middleware {
	return NewMiddlewareBuilder().
		WithSlogLogger(slog.LevelDebug).
		WithCompositeAuditLogger(
			NewDatabaseAuditLogger(DatabaseAuditLoggerConfig{DB: db}),
			NewJSONAuditLogger(NewSlogLogger(slog.LevelDebug)),
		).
		WithInMemoryRateLimiter(1000, time.Minute). // Higher limit for development
		WithDefaultErrorHandler(true, false).       // Include stack traces, don't mask errors
		AddRequestID().
		AddRecovery(true). // Include stack traces in recovery
		AddErrorHandling().
		AddLogging().
		AddRateLimit().
		Build()
}

// ProductionMiddlewareStack creates a middleware stack suitable for production
func ProductionMiddlewareStack(db *gorm.DB) []Middleware {
	return NewMiddlewareBuilder().
		WithSlogLogger(slog.LevelWarn).
		WithDatabaseAuditLogger(db).
		WithInMemoryRateLimiter(60, time.Minute). // Conservative limit for production
		WithDefaultErrorHandler(false, true).     // No stack traces, mask internal errors
		AddRequestID().
		AddRecovery(false). // No stack traces in recovery
		AddErrorHandling().
		AddLogging("/api/v1/health", "/api/v1/ready"). // Skip health check logging
		AddRateLimit("/api/v1/health", "/api/v1/ready").
		Build()
}