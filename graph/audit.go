package graph

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/carlosf/k8s-pod-manager/internal/database"
	"github.com/carlosf/k8s-pod-manager/internal/models"
)

// DefaultAuditLogger implements AuditLogger interface
// Provides audit logging functionality for GraphQL operations
type DefaultAuditLogger struct {
	serviceName string
	enabled     bool
}

// NewDefaultAuditLogger creates a new instance of DefaultAuditLogger
func NewDefaultAuditLogger(serviceName string, enabled bool) *DefaultAuditLogger {
	return &DefaultAuditLogger{
		serviceName: serviceName,
		enabled:     enabled,
	}
}

// LogMutation logs mutation operations for audit purposes
func (a *DefaultAuditLogger) LogMutation(ctx context.Context, operation, namespace, resource, user string, success bool, errorMsg string) {
	if !a.enabled {
		return
	}

	auditLog := models.AuditLog{
		Action:       operation,
		ResourceType: "pod",
		ResourceName: resource,
		Namespace:    namespace,
		User:         user,
		UserAgent:    a.extractUserAgent(ctx),
		IP:           a.extractRemoteAddr(ctx),
		Method:       "POST",
		Path:         fmt.Sprintf("/graphql/mutation.%s", operation),
		StatusCode:   200,
		Error:        errorMsg,
		RequestBody:  "{}",  // Valid JSON for JSONB field
		Response:     "{}",  // Valid JSON for JSONB field
		CreatedAt:    time.Now(),
	}

	if !success {
		auditLog.StatusCode = 500
	}

	// Log to database if available
	if database.DB != nil {
		if err := database.DB.Create(&auditLog).Error; err != nil {
			log.Printf("Failed to save audit log to database: %v", err)
		}
	}

	// Always log to stdout for immediate visibility
	logLevel := "INFO"
	if !success {
		logLevel = "ERROR"
	}

	log.Printf("[%s] GraphQL Mutation - Operation: %s, Namespace: %s, Resource: %s, User: %s, Success: %t, Error: %s",
		logLevel, operation, namespace, resource, user, success, errorMsg)
}

// LogQuery logs query operations for audit purposes
func (a *DefaultAuditLogger) LogQuery(ctx context.Context, operation, namespace string, resourceCount int) {
	if !a.enabled {
		return
	}

	auditLog := models.AuditLog{
		Action:       operation,
		ResourceType: "query",
		ResourceName: fmt.Sprintf("%d_resources", resourceCount),
		Namespace:    namespace,
		User:         a.extractUser(ctx),
		UserAgent:    a.extractUserAgent(ctx),
		IP:           a.extractRemoteAddr(ctx),
		Method:       "POST",
		Path:         fmt.Sprintf("/graphql/query.%s", operation),
		StatusCode:   200,
		RequestBody:  "{}",  // Valid JSON for JSONB field
		Response:     "{}",  // Valid JSON for JSONB field
		CreatedAt:    time.Now(),
	}

	// Log to database if available
	if database.DB != nil {
		if err := database.DB.Create(&auditLog).Error; err != nil {
			log.Printf("Failed to save audit log to database: %v", err)
		}
	}

	log.Printf("[INFO] GraphQL Query - Operation: %s, Namespace: %s, ResourceCount: %d, User: %s",
		operation, namespace, resourceCount, a.extractUser(ctx))
}

// LogSubscription logs subscription operations for audit purposes
func (a *DefaultAuditLogger) LogSubscription(ctx context.Context, operation, namespace string, started bool) {
	if !a.enabled {
		return
	}

	action := "stopped"
	if started {
		action = "started"
	}

	auditLog := models.AuditLog{
		Action:       fmt.Sprintf("%s_%s", operation, action),
		ResourceType: "subscription",
		ResourceName: operation,
		Namespace:    namespace,
		User:         a.extractUser(ctx),
		UserAgent:    a.extractUserAgent(ctx),
		IP:           a.extractRemoteAddr(ctx),
		Method:       "GET",
		Path:         fmt.Sprintf("/graphql/subscription.%s", operation),
		StatusCode:   200,
		RequestBody:  "{}",  // Valid JSON for JSONB field
		Response:     "{}",  // Valid JSON for JSONB field
		CreatedAt:    time.Now(),
	}

	// Log to database if available
	if database.DB != nil {
		if err := database.DB.Create(&auditLog).Error; err != nil {
			log.Printf("Failed to save audit log to database: %v", err)
		}
	}

	log.Printf("[INFO] GraphQL Subscription - Operation: %s, Namespace: %s, Action: %s, User: %s",
		operation, namespace, action, a.extractUser(ctx))
}

// extractUser extracts user information from context
func (a *DefaultAuditLogger) extractUser(ctx context.Context) string {
	if user := ctx.Value("user"); user != nil {
		if userStr, ok := user.(string); ok {
			return userStr
		}
	}
	return "system"
}

// extractUserAgent extracts user agent from context
func (a *DefaultAuditLogger) extractUserAgent(ctx context.Context) string {
	if ua := ctx.Value("user-agent"); ua != nil {
		if uaStr, ok := ua.(string); ok {
			return uaStr
		}
	}
	return "unknown"
}

// extractRemoteAddr extracts remote address from context
func (a *DefaultAuditLogger) extractRemoteAddr(ctx context.Context) string {
	if addr := ctx.Value("remote-addr"); addr != nil {
		if addrStr, ok := addr.(string); ok {
			return addrStr
		}
	}
	return "unknown"
}

// NoOpAuditLogger implements AuditLogger interface with no-op methods
// Useful for testing or when audit logging is disabled
type NoOpAuditLogger struct{}

// NewNoOpAuditLogger creates a new no-op audit logger
func NewNoOpAuditLogger() *NoOpAuditLogger {
	return &NoOpAuditLogger{}
}

// LogMutation does nothing
func (n *NoOpAuditLogger) LogMutation(ctx context.Context, operation, namespace, resource, user string, success bool, errorMsg string) {
	// No-op
}

// LogQuery does nothing
func (n *NoOpAuditLogger) LogQuery(ctx context.Context, operation, namespace string, resourceCount int) {
	// No-op
}

// LogSubscription does nothing
func (n *NoOpAuditLogger) LogSubscription(ctx context.Context, operation, namespace string, started bool) {
	// No-op
}

// CompositeAuditLogger allows multiple audit loggers (Composite Pattern)
type CompositeAuditLogger struct {
	loggers []AuditLogger
}

// NewCompositeAuditLogger creates a new composite audit logger
func NewCompositeAuditLogger(loggers ...AuditLogger) *CompositeAuditLogger {
	return &CompositeAuditLogger{
		loggers: loggers,
	}
}

// LogMutation logs to all composed loggers
func (c *CompositeAuditLogger) LogMutation(ctx context.Context, operation, namespace, resource, user string, success bool, errorMsg string) {
	for _, logger := range c.loggers {
		logger.LogMutation(ctx, operation, namespace, resource, user, success, errorMsg)
	}
}

// LogQuery logs to all composed loggers
func (c *CompositeAuditLogger) LogQuery(ctx context.Context, operation, namespace string, resourceCount int) {
	for _, logger := range c.loggers {
		logger.LogQuery(ctx, operation, namespace, resourceCount)
	}
}

// LogSubscription logs to all composed loggers
func (c *CompositeAuditLogger) LogSubscription(ctx context.Context, operation, namespace string, started bool) {
	for _, logger := range c.loggers {
		logger.LogSubscription(ctx, operation, namespace, started)
	}
}