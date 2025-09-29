//go:build ignore
// +build ignore

package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// HealthHandler provides health check endpoints with dependency injection
type HealthHandler struct {
	dbChecker         DatabaseHealthChecker
	k8sChecker        KubernetesHealthChecker
	appInfo          AppInfoProvider
	metricsProvider  MetricsProvider
	metricsFormatter *PrometheusFormatter
}

// NewHealthHandler creates a new health handler with injected dependencies
func NewHealthHandler(
	dbChecker DatabaseHealthChecker,
	k8sChecker KubernetesHealthChecker,
	appInfo AppInfoProvider,
	metricsProvider MetricsProvider,
) *HealthHandler {
	return &HealthHandler{
		dbChecker:        dbChecker,
		k8sChecker:       k8sChecker,
		appInfo:         appInfo,
		metricsProvider: metricsProvider,
		metricsFormatter: NewPrometheusFormatter(),
	}
}

// Health provides a basic health check endpoint
func (h *HealthHandler) Health(c *gin.Context) {
	ctx := NewGinContextAdapter(c)
	h.handleHealth(ctx)
}

// handleHealth handles the health check logic
func (h *HealthHandler) handleHealth(ctx Context) {
	appInfo := h.appInfo.GetAppInfo()

	response := &HealthResponse{
		Status:    "healthy",
		Uptime:    time.Since(appInfo.StartTime).String(),
		Version:   appInfo.Version,
		Timestamp: time.Now(),
	}

	ctx.JSON(http.StatusOK, response)
}

// Readiness provides a readiness check endpoint
func (h *HealthHandler) Readiness(c *gin.Context) {
	ctx := NewGinContextAdapter(c)
	h.handleReadiness(ctx)
}

// handleReadiness handles the readiness check logic
func (h *HealthHandler) handleReadiness(ctx Context) {
	checks := make(map[string]interface{})
	allHealthy := true
	requestCtx := ctx.Context()

	// Check database health
	if h.dbChecker != nil {
		if err := h.dbChecker.CheckHealth(requestCtx); err != nil {
			checks["database"] = "unhealthy"
			allHealthy = false
		} else {
			checks["database"] = "healthy"
		}
	} else {
		checks["database"] = "not configured"
	}

	// Check Kubernetes health
	if h.k8sChecker != nil {
		if err := h.k8sChecker.CheckHealth(requestCtx); err != nil {
			checks["kubernetes"] = "unhealthy"
			allHealthy = false
		} else {
			checks["kubernetes"] = "healthy"
		}
	} else {
		checks["kubernetes"] = "not configured"
	}

	appInfo := h.appInfo.GetAppInfo()
	status := http.StatusOK
	statusText := "ready"
	if !allHealthy {
		status = http.StatusServiceUnavailable
		statusText = "not ready"
	}

	response := &HealthResponse{
		Status:    statusText,
		Checks:    checks,
		Uptime:    time.Since(appInfo.StartTime).String(),
		Version:   appInfo.Version,
		Timestamp: time.Now(),
	}

	ctx.JSON(status, response)
}

// Info provides detailed service information endpoint
func (h *HealthHandler) Info(c *gin.Context) {
	ctx := NewGinContextAdapter(c)
	h.handleInfo(ctx)
}

// handleInfo handles the service information logic
func (h *HealthHandler) handleInfo(ctx Context) {
	appInfo := h.appInfo.GetAppInfo()

	response := &InfoResponse{
		Service: &ServiceInfo{
			Name:    appInfo.Name,
			Version: appInfo.Version,
			Uptime:  time.Since(appInfo.StartTime).String(),
		},
		Timestamp: time.Now(),
	}

	// Add Kubernetes information if available
	if h.k8sChecker != nil {
		if serverVersion, err := h.k8sChecker.GetServerVersion(); err == nil {
			response.Kubernetes = serverVersion
		}
	}

	// Add database information if available
	if h.dbChecker != nil {
		if stats, err := h.dbChecker.GetStats(); err == nil {
			response.Database = stats
		}
	}

	ctx.JSON(http.StatusOK, response)
}

// Metrics provides Prometheus-formatted metrics endpoint
func (h *HealthHandler) Metrics(c *gin.Context) {
	ctx := NewGinContextAdapter(c)
	h.handleMetrics(ctx)
}

// handleMetrics handles the metrics logic
func (h *HealthHandler) handleMetrics(ctx Context) {
	requestCtx := ctx.Context()

	metricsData, err := h.metricsProvider.GetMetrics(requestCtx)
	if err != nil {
		// Return error response in JSON format for consistency
		builder := NewResponseBuilder("")
		errorResp := builder.InternalError("Failed to gather metrics")
		ctx.JSON(http.StatusInternalServerError, errorResp)
		return
	}

	// Format metrics in Prometheus text format
	metricsText := h.metricsFormatter.Format(metricsData)

	ctx.Header("Content-Type", "text/plain; version=0.0.4")
	ctx.String(http.StatusOK, metricsText)
}

// RegisterRoutes registers health check routes on the provided router
func (h *HealthHandler) RegisterRoutes(router Router) {
	router.GET("/health", h.handleHealth)
	router.GET("/ready", h.handleReadiness)
	router.GET("/info", h.handleInfo)
	router.GET("/metrics", h.handleMetrics)
}