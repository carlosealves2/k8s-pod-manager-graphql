package middleware

import (
	"context"
	"time"
)

// HTTPRequest represents an abstracted HTTP request
type HTTPRequest interface {
	Method() string
	Path() string
	Header(key string) string
	Body() []byte
	IP() string
	UserAgent() string
	RequestID() string
}

// HTTPResponse represents an abstracted HTTP response
type HTTPResponse interface {
	SetHeader(key, value string)
	SetStatus(code int)
	Write(data []byte) error
	JSON(data interface{}) error
}

// HTTPContext represents an abstracted HTTP context
type HTTPContext interface {
	Request() HTTPRequest
	Response() HTTPResponse
	Set(key string, value interface{})
	Get(key string) (interface{}, bool)
	Next() error
}

// Logger defines the logging interface for middleware
type Logger interface {
	Debug(ctx context.Context, msg string, fields ...Field)
	Info(ctx context.Context, msg string, fields ...Field)
	Warn(ctx context.Context, msg string, fields ...Field)
	Error(ctx context.Context, msg string, err error, fields ...Field)
}

// Field represents a structured log field
type Field struct {
	Key   string
	Value interface{}
}

// AuditLogger defines the interface for audit logging
type AuditLogger interface {
	LogRequest(ctx context.Context, entry *AuditEntry) error
}

// AuditEntry represents an audit log entry
type AuditEntry struct {
	RequestID    string
	Method       string
	Path         string
	IP           string
	UserAgent    string
	User         string
	RequestBody  string
	StatusCode   int
	Duration     time.Duration
	Error        string
	Action       string
	ResourceType string
	ResourceName string
	Namespace    string
	Timestamp    time.Time
}

// RateLimiter defines the interface for rate limiting
type RateLimiter interface {
	Allow(ctx context.Context, key string) (*RateLimitResult, error)
	Reset(ctx context.Context, key string) error
}

// RateLimitResult contains rate limiting information
type RateLimitResult struct {
	Allowed   bool
	Limit     int
	Remaining int
	ResetTime time.Time
	RetryAfter time.Duration
}

// ErrorHandler defines the interface for error handling
type ErrorHandler interface {
	HandleError(ctx HTTPContext, err error) error
}

// Middleware defines the common interface for all middleware
type Middleware interface {
	Handle(ctx HTTPContext) error
}

// MiddlewareFunc is a function adapter for the Middleware interface
type MiddlewareFunc func(ctx HTTPContext) error

// Handle implements the Middleware interface
func (f MiddlewareFunc) Handle(ctx HTTPContext) error {
	return f(ctx)
}

// RequestIDGenerator defines the interface for generating request IDs
type RequestIDGenerator interface {
	Generate() string
}