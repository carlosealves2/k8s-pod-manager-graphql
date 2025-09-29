package handlers

import (
	"context"
	"time"
)

// HealthChecker defines interface for health checking dependencies
type HealthChecker interface {
	CheckHealth(ctx context.Context) error
}

// DatabaseHealthChecker provides database health checking
type DatabaseHealthChecker interface {
	HealthChecker
	GetStats() (*DatabaseStats, error)
}

// KubernetesHealthChecker provides Kubernetes health checking
type KubernetesHealthChecker interface {
	HealthChecker
	GetServerVersion() (*ServerVersionInfo, error)
}

// AppInfoProvider provides application information
type AppInfoProvider interface {
	GetAppInfo() *AppInfo
}

// MetricsProvider provides application metrics
type MetricsProvider interface {
	GetMetrics(ctx context.Context) (*MetricsData, error)
}

// HTTPHandler defines common HTTP handler interface
type HTTPHandler interface {
	RegisterRoutes(router Router)
}

// Router abstracts routing functionality (Gin/Fiber agnostic)
type Router interface {
	GET(path string, handler func(Context))
	POST(path string, handler func(Context))
	PUT(path string, handler func(Context))
	DELETE(path string, handler func(Context))
}

// Context abstracts HTTP context (Gin/Fiber agnostic)
type Context interface {
	JSON(statusCode int, data interface{})
	String(statusCode int, data string)
	Header(key, value string)
	Param(key string) string
	Query(key string) string
	Get(key string) string
	BodyParser(out interface{}) error
	Context() context.Context
}

// DatabaseStats represents database connection statistics
type DatabaseStats struct {
	OpenConnections int
	InUse          int
	Idle           int
	WaitCount      int64
	WaitDuration   time.Duration
}

// ServerVersionInfo represents Kubernetes server version information
type ServerVersionInfo struct {
	GitVersion string
	Platform   string
	BuildDate  string
	GitCommit  string
	GoVersion  string
	Compiler   string
	Major      string
	Minor      string
}

// AppInfo represents application information
type AppInfo struct {
	Name      string
	Version   string
	StartTime time.Time
}

// MetricsData represents application metrics
type MetricsData struct {
	DatabaseMetrics []Metric
	AppMetrics      []Metric
}

// Metric represents a single metric entry
type Metric struct {
	Name   string
	Value  float64
	Help   string
	Labels map[string]string
}