package middleware

import (
	"crypto/rand"
	"fmt"
	"time"
)

// UUIDGenerator implements RequestIDGenerator using UUID v4 format
type UUIDGenerator struct{}

// Generate creates a new UUID v4 format request ID
func (g *UUIDGenerator) Generate() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		// Fallback to timestamp-based ID
		return fmt.Sprintf("req_%d", time.Now().UnixNano())
	}

	// Set version (4) and variant bits
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80

	return fmt.Sprintf("%x-%x-%x-%x-%x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}

// RequestIDMiddleware handles request ID generation and propagation
type RequestIDMiddleware struct {
	generator RequestIDGenerator
	header    string
}

// RequestIDConfig configures the request ID middleware
type RequestIDConfig struct {
	Generator RequestIDGenerator
	Header    string // Header name for request ID (default: "X-Request-ID")
}

// NewRequestIDMiddleware creates a new request ID middleware
func NewRequestIDMiddleware(config RequestIDConfig) Middleware {
	if config.Generator == nil {
		config.Generator = &UUIDGenerator{}
	}
	if config.Header == "" {
		config.Header = "X-Request-ID"
	}

	return &RequestIDMiddleware{
		generator: config.Generator,
		header:    config.Header,
	}
}

// Handle implements the Middleware interface
func (m *RequestIDMiddleware) Handle(ctx HTTPContext) error {
	requestID := ctx.Request().Header(m.header)

	if requestID == "" {
		requestID = m.generator.Generate()
		ctx.Response().SetHeader(m.header, requestID)
	}

	// Store in context for other middleware and handlers
	ctx.Set("requestID", requestID)
	ctx.Set("request_id", requestID) // Alternative key for compatibility

	return ctx.Next()
}