package middleware

import (
	"context"
	"log/slog"
	"os"
	"time"
)

// SlogLogger implements Logger interface using slog
type SlogLogger struct {
	logger *slog.Logger
}

// NewSlogLogger creates a new structured logger using slog
func NewSlogLogger(level slog.Level) Logger {
	opts := &slog.HandlerOptions{
		Level: level,
	}

	handler := slog.NewJSONHandler(os.Stdout, opts)
	logger := slog.New(handler)

	return &SlogLogger{logger: logger}
}

func (l *SlogLogger) Debug(ctx context.Context, msg string, fields ...Field) {
	l.logger.DebugContext(ctx, msg, convertFields(fields)...)
}

func (l *SlogLogger) Info(ctx context.Context, msg string, fields ...Field) {
	l.logger.InfoContext(ctx, msg, convertFields(fields)...)
}

func (l *SlogLogger) Warn(ctx context.Context, msg string, fields ...Field) {
	l.logger.WarnContext(ctx, msg, convertFields(fields)...)
}

func (l *SlogLogger) Error(ctx context.Context, msg string, err error, fields ...Field) {
	allFields := append([]Field{{Key: "error", Value: err.Error()}}, fields...)
	l.logger.ErrorContext(ctx, msg, convertFields(allFields)...)
}

func convertFields(fields []Field) []interface{} {
	args := make([]interface{}, 0, len(fields)*2)
	for _, field := range fields {
		args = append(args, field.Key, field.Value)
	}
	return args
}

// LoggingMiddleware handles HTTP request/response logging
type LoggingMiddleware struct {
	logger      Logger
	auditLogger AuditLogger
	skipPaths   map[string]bool
}

// LoggingConfig configures the logging middleware
type LoggingConfig struct {
	Logger      Logger
	AuditLogger AuditLogger
	SkipPaths   []string // Paths to skip logging (e.g., health checks)
}

// NewLoggingMiddleware creates a new logging middleware
func NewLoggingMiddleware(config LoggingConfig) Middleware {
	skipPaths := make(map[string]bool)
	for _, path := range config.SkipPaths {
		skipPaths[path] = true
	}

	return &LoggingMiddleware{
		logger:      config.Logger,
		auditLogger: config.AuditLogger,
		skipPaths:   skipPaths,
	}
}

// Handle implements the Middleware interface
func (m *LoggingMiddleware) Handle(ctx HTTPContext) error {
	start := time.Now()
	req := ctx.Request()

	// Skip logging for certain paths
	if m.skipPaths[req.Path()] {
		return ctx.Next()
	}

	// Extract request ID from context
	requestID := ""
	if id, exists := ctx.Get("requestID"); exists {
		if idStr, ok := id.(string); ok {
			requestID = idStr
		}
	}

	// Create base context with request ID
	logCtx := context.Background()
	if requestID != "" {
		logCtx = context.WithValue(logCtx, "request_id", requestID)
	}

	// Log request start
	m.logger.Info(logCtx, "Request started",
		Field{Key: "method", Value: req.Method()},
		Field{Key: "path", Value: req.Path()},
		Field{Key: "ip", Value: req.IP()},
		Field{Key: "user_agent", Value: req.UserAgent()},
		Field{Key: "request_id", Value: requestID},
	)

	// Process request
	err := ctx.Next()

	// Calculate duration
	duration := time.Since(start)

	// Prepare audit entry
	auditEntry := &AuditEntry{
		RequestID:   requestID,
		Method:      req.Method(),
		Path:        req.Path(),
		IP:          req.IP(),
		UserAgent:   req.UserAgent(),
		User:        req.Header("X-User"),
		Duration:    duration,
		Timestamp:   start,
		RequestBody: extractRequestBody(req),
	}

	// Set default user if empty
	if auditEntry.User == "" {
		auditEntry.User = "anonymous"
	}

	// Extract status code from response (this is framework-specific)
	// For now, we'll set a default and let the audit logger implementation handle it
	auditEntry.StatusCode = 200

	if err != nil {
		auditEntry.Error = err.Error()
		auditEntry.StatusCode = 500

		m.logger.Error(logCtx, "Request failed",
			err,
			Field{Key: "method", Value: req.Method()},
			Field{Key: "path", Value: req.Path()},
			Field{Key: "duration_ms", Value: duration.Milliseconds()},
			Field{Key: "request_id", Value: requestID},
		)
	} else {
		m.logger.Info(logCtx, "Request completed",
			Field{Key: "method", Value: req.Method()},
			Field{Key: "path", Value: req.Path()},
			Field{Key: "duration_ms", Value: duration.Milliseconds()},
			Field{Key: "request_id", Value: requestID},
		)
	}

	// Parse action from request
	parseAction(auditEntry)

	// Log audit entry asynchronously if audit logger is configured
	if m.auditLogger != nil {
		go func() {
			if auditErr := m.auditLogger.LogRequest(logCtx, auditEntry); auditErr != nil {
				m.logger.Error(logCtx, "Failed to log audit entry", auditErr,
					Field{Key: "request_id", Value: requestID},
				)
			}
		}()
	}

	return err
}

func extractRequestBody(req HTTPRequest) string {
	if req.Method() != "POST" && req.Method() != "PUT" && req.Method() != "PATCH" {
		return ""
	}

	body := req.Body()
	if len(body) == 0 {
		return ""
	}

	// Basic JSON validation and formatting
	if len(body) > 10000 { // Limit body size for logging
		return "[BODY_TOO_LARGE]"
	}

	return string(body)
}

func parseAction(entry *AuditEntry) {
	path := entry.Path
	method := entry.Method

	// Set defaults
	entry.Action = "unknown"
	entry.ResourceType = "unknown"
	entry.ResourceName = path

	switch {
	case method == "POST" && path == "/graphql":
		entry.Action = "graphql_request"
		entry.ResourceType = "graphql"
		entry.ResourceName = "query"
	case path == "/api/v1/health":
		entry.Action = "health_check"
		entry.ResourceType = "system"
		entry.ResourceName = "health"
	case path == "/api/v1/ready":
		entry.Action = "readiness_check"
		entry.ResourceType = "system"
		entry.ResourceName = "readiness"
	case path == "/api/v1/metrics":
		entry.Action = "metrics"
		entry.ResourceType = "system"
		entry.ResourceName = "metrics"
	case path == "/api/v1/info":
		entry.Action = "info"
		entry.ResourceType = "system"
		entry.ResourceName = "info"
	case path == "/":
		entry.Action = "playground"
		entry.ResourceType = "graphql"
		entry.ResourceName = "playground"
	}
}