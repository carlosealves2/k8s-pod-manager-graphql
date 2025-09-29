package middleware

import (
	"context"
	"encoding/json"

	"github.com/carlosf/k8s-pod-manager/internal/models"
	"gorm.io/gorm"
)

// DatabaseAuditLogger implements AuditLogger using GORM database
type DatabaseAuditLogger struct {
	db     *gorm.DB
	logger Logger
}

// DatabaseAuditLoggerConfig configures the database audit logger
type DatabaseAuditLoggerConfig struct {
	DB     *gorm.DB
	Logger Logger
}

// NewDatabaseAuditLogger creates a new database audit logger
func NewDatabaseAuditLogger(config DatabaseAuditLoggerConfig) AuditLogger {
	return &DatabaseAuditLogger{
		db:     config.DB,
		logger: config.Logger,
	}
}

// LogRequest implements the AuditLogger interface
func (a *DatabaseAuditLogger) LogRequest(ctx context.Context, entry *AuditEntry) error {
	if a.db == nil {
		return nil // Skip if database is not configured
	}

	// Convert to GORM model
	auditLog := &models.AuditLog{
		Method:       entry.Method,
		Path:         entry.Path,
		IP:           entry.IP,
		UserAgent:    entry.UserAgent,
		User:         entry.User,
		RequestBody:  entry.RequestBody,
		StatusCode:   entry.StatusCode,
		Duration:     entry.Duration.Milliseconds(),
		Error:        entry.Error,
		Action:       entry.Action,
		ResourceType: entry.ResourceType,
		ResourceName: entry.ResourceName,
		Namespace:    entry.Namespace,
		CreatedAt:    entry.Timestamp,
	}

	// Create the audit log entry
	if err := a.db.Create(auditLog).Error; err != nil {
		if a.logger != nil {
			a.logger.Error(ctx, "Failed to create audit log entry", err,
				Field{Key: "request_id", Value: entry.RequestID},
				Field{Key: "path", Value: entry.Path},
				Field{Key: "method", Value: entry.Method},
			)
		}
		return err
	}

	return nil
}

// NoOpAuditLogger implements AuditLogger with no-op operations
type NoOpAuditLogger struct{}

// NewNoOpAuditLogger creates a new no-op audit logger
func NewNoOpAuditLogger() AuditLogger {
	return &NoOpAuditLogger{}
}

// LogRequest implements the AuditLogger interface with no-op
func (a *NoOpAuditLogger) LogRequest(ctx context.Context, entry *AuditEntry) error {
	return nil
}

// JSONAuditLogger implements AuditLogger by writing JSON to a logger
type JSONAuditLogger struct {
	logger Logger
}

// NewJSONAuditLogger creates a new JSON audit logger
func NewJSONAuditLogger(logger Logger) AuditLogger {
	return &JSONAuditLogger{
		logger: logger,
	}
}

// LogRequest implements the AuditLogger interface
func (a *JSONAuditLogger) LogRequest(ctx context.Context, entry *AuditEntry) error {
	if a.logger == nil {
		return nil
	}

	// Convert entry to JSON-serializable format
	logEntry := map[string]interface{}{
		"request_id":    entry.RequestID,
		"method":        entry.Method,
		"path":          entry.Path,
		"ip":            entry.IP,
		"user_agent":    entry.UserAgent,
		"user":          entry.User,
		"status_code":   entry.StatusCode,
		"duration_ms":   entry.Duration.Milliseconds(),
		"action":        entry.Action,
		"resource_type": entry.ResourceType,
		"resource_name": entry.ResourceName,
		"namespace":     entry.Namespace,
		"timestamp":     entry.Timestamp,
	}

	if entry.RequestBody != "" {
		logEntry["request_body"] = entry.RequestBody
	}

	if entry.Error != "" {
		logEntry["error"] = entry.Error
	}

	// Marshal to JSON for pretty formatting
	jsonData, err := json.Marshal(logEntry)
	if err != nil {
		a.logger.Error(ctx, "Failed to marshal audit entry to JSON", err,
			Field{Key: "request_id", Value: entry.RequestID},
		)
		return err
	}

	// Log as structured data
	a.logger.Info(ctx, "Audit log entry",
		Field{Key: "audit_entry", Value: string(jsonData)},
		Field{Key: "request_id", Value: entry.RequestID},
		Field{Key: "action", Value: entry.Action},
		Field{Key: "resource", Value: entry.ResourceType + "/" + entry.ResourceName},
	)

	return nil
}

// CompositeAuditLogger combines multiple audit loggers
type CompositeAuditLogger struct {
	loggers []AuditLogger
}

// NewCompositeAuditLogger creates a new composite audit logger
func NewCompositeAuditLogger(loggers ...AuditLogger) AuditLogger {
	return &CompositeAuditLogger{
		loggers: loggers,
	}
}

// LogRequest implements the AuditLogger interface
func (c *CompositeAuditLogger) LogRequest(ctx context.Context, entry *AuditEntry) error {
	var lastErr error

	for _, logger := range c.loggers {
		if err := logger.LogRequest(ctx, entry); err != nil {
			lastErr = err
		}
	}

	return lastErr
}