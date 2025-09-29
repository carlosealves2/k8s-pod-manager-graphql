//go:build ignore
// +build ignore

package handlers

import (
	"time"

	"github.com/carlosf/k8s-pod-manager/internal/database"
	"github.com/carlosf/k8s-pod-manager/internal/kubernetes"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"k8s.io/client-go/kubernetes"
)

// SetupHealthHandlerWithDI demonstrates how to set up the health handler with dependency injection
// This function should be called from main.go after initializing dependencies
func SetupHealthHandlerWithDI(router *gin.RouterGroup, db *gorm.DB, k8sClient kubernetes.Interface, startTime time.Time) *HealthHandler {
	// Create dependency implementations
	dbChecker := NewGormDatabaseHealthChecker(db)
	k8sChecker := NewKubernetesClientHealthChecker(k8sClient)
	appInfo := NewDefaultAppInfoProvider("k8s-pod-manager", "1.0.0", startTime)
	metricsProvider := NewDefaultMetricsProvider(dbChecker, appInfo)

	// Create health handler with injected dependencies
	healthHandler := NewHealthHandler(dbChecker, k8sChecker, appInfo, metricsProvider)

	// Register routes using the adapter pattern
	routerAdapter := NewGinRouterAdapter(router)
	healthHandler.RegisterRoutes(routerAdapter)

	return healthHandler
}

// Alternative setup for existing main.go with minimal changes
func SetupHealthHandlerLegacy(router *gin.RouterGroup, startTime time.Time) *HealthHandler {
	// Create dependency implementations using global variables (legacy approach)
	var dbChecker DatabaseHealthChecker
	var k8sChecker KubernetesHealthChecker

	if database.DB != nil {
		dbChecker = NewGormDatabaseHealthChecker(database.DB)
	}

	if kubernetes.Client != nil {
		k8sChecker = NewKubernetesClientHealthChecker(kubernetes.Client)
	}

	appInfo := NewDefaultAppInfoProvider("k8s-pod-manager", "1.0.0", startTime)
	metricsProvider := NewDefaultMetricsProvider(dbChecker, appInfo)

	// Create health handler with injected dependencies
	healthHandler := NewHealthHandler(dbChecker, k8sChecker, appInfo, metricsProvider)

	// Register individual routes (compatible with existing main.go)
	router.GET("/health", healthHandler.Health)
	router.GET("/ready", healthHandler.Readiness)
	router.GET("/info", healthHandler.Info)
	router.GET("/metrics", healthHandler.Metrics)

	return healthHandler
}