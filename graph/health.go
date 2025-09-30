package graph

import (
	"time"

	"github.com/carlosf/k8s-pod-manager/graph/model"
	"github.com/carlosf/k8s-pod-manager/graph/scalar"
	"github.com/carlosf/k8s-pod-manager/internal/database"
	"github.com/carlosf/k8s-pod-manager/internal/kubernetes"
)

// DefaultHealthService implements HealthServiceInterface
// Handles health check operations with proper separation of concerns
type DefaultHealthService struct {
	startTime time.Time
	version   string
}

// NewDefaultHealthService creates a new instance of DefaultHealthService
func NewDefaultHealthService(version string) *DefaultHealthService {
	return &DefaultHealthService{
		startTime: time.Now(),
		version:   version,
	}
}

// GetHealth returns the basic health status of the service
func (h *DefaultHealthService) GetHealth() *model.HealthResponse {
	return &model.HealthResponse{
		Status:    "healthy",
		Timestamp: scalar.Time(time.Now()),
		Uptime:    time.Since(h.startTime).String(),
		Version:   h.version,
	}
}

// GetReadiness returns the readiness status including dependency checks
func (h *DefaultHealthService) GetReadiness() *model.ReadinessResponse {
	checks := &model.ReadinessCheck{}
	allHealthy := true

	// Check database connectivity
	if database.DB != nil {
		sqlDB, err := database.DB.DB()
		if err != nil || sqlDB.Ping() != nil {
			checks.Database = "unhealthy"
			allHealthy = false
		} else {
			checks.Database = "healthy"
		}
	} else {
		checks.Database = "not initialized"
		allHealthy = false
	}

	// Check Kubernetes connectivity
	if kubernetes.Client != nil {
		_, err := kubernetes.Client.Discovery().ServerVersion()
		if err != nil {
			checks.Kubernetes = "unhealthy"
			allHealthy = false
		} else {
			checks.Kubernetes = "healthy"
		}
	} else {
		checks.Kubernetes = "not initialized"
		allHealthy = false
	}

	status := "ready"
	if !allHealthy {
		status = "not ready"
	}

	return &model.ReadinessResponse{
		Status:    status,
		Checks:    checks,
		Timestamp: scalar.Time(time.Now()),
	}
}

// GetSystemInfo returns detailed system information
func (h *DefaultHealthService) GetSystemInfo() *model.InfoResponse {
	var k8sVersion, k8sPlatform string = "unknown", "unknown"

	// Get Kubernetes version information
	if kubernetes.Client != nil {
		if version, err := kubernetes.Client.Discovery().ServerVersion(); err == nil {
			k8sVersion = version.String()
			k8sPlatform = version.Platform
		}
	}

	// Get database statistics
	var dbStats model.DatabaseInfo
	if database.DB != nil {
		if sqlDB, err := database.DB.DB(); err == nil {
			stats := sqlDB.Stats()
			dbStats = model.DatabaseInfo{
				OpenConnections: stats.OpenConnections,
				InUse:          stats.InUse,
				Idle:           stats.Idle,
			}
		}
	}

	return &model.InfoResponse{
		Service: &model.SystemInfo{
			Name:    "K8s Pod Manager API",
			Version: h.version,
			Uptime:  time.Since(h.startTime).String(),
		},
		Kubernetes: &model.KubernetesInfo{
			Version:  k8sVersion,
			Platform: k8sPlatform,
		},
		Database:  &dbStats,
		Timestamp: scalar.Time(time.Now()),
	}
}

// CachedHealthService wraps a HealthService with caching (Decorator Pattern)
// Useful for expensive health checks that don't need to run on every request
type CachedHealthService struct {
	wrapped     HealthServiceInterface
	cacheTTL    time.Duration
	lastHealth  *model.HealthResponse
	lastHealthTime time.Time
	lastReadiness *model.ReadinessResponse
	lastReadinessTime time.Time
	lastInfo    *model.InfoResponse
	lastInfoTime time.Time
}

// NewCachedHealthService creates a new cached health service
func NewCachedHealthService(wrapped HealthServiceInterface, cacheTTL time.Duration) *CachedHealthService {
	return &CachedHealthService{
		wrapped:  wrapped,
		cacheTTL: cacheTTL,
	}
}

// GetHealth returns cached health status or refreshes if cache is stale
func (c *CachedHealthService) GetHealth() *model.HealthResponse {
	now := time.Now()
	if c.lastHealth == nil || now.Sub(c.lastHealthTime) > c.cacheTTL {
		c.lastHealth = c.wrapped.GetHealth()
		c.lastHealthTime = now
	}
	return c.lastHealth
}

// GetReadiness returns cached readiness status or refreshes if cache is stale
func (c *CachedHealthService) GetReadiness() *model.ReadinessResponse {
	now := time.Now()
	if c.lastReadiness == nil || now.Sub(c.lastReadinessTime) > c.cacheTTL {
		c.lastReadiness = c.wrapped.GetReadiness()
		c.lastReadinessTime = now
	}
	return c.lastReadiness
}

// GetSystemInfo returns cached system info or refreshes if cache is stale
func (c *CachedHealthService) GetSystemInfo() *model.InfoResponse {
	now := time.Now()
	if c.lastInfo == nil || now.Sub(c.lastInfoTime) > c.cacheTTL {
		c.lastInfo = c.wrapped.GetSystemInfo()
		c.lastInfoTime = now
	}
	return c.lastInfo
}