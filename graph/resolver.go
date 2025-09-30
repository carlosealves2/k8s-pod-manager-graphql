package graph

import (
	"context"
	"time"

	"github.com/carlosf/k8s-pod-manager/internal/kubernetes"
	"github.com/carlosf/k8s-pod-manager/internal/services"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// This file will not be regenerated automatically.
//
// It serves as dependency injection for your app, add any dependencies you require here.

// Resolver struct now follows Dependency Injection principle
// All dependencies are injected via interfaces, making it testable and flexible
type Resolver struct {
	// Core services injected as interfaces following Dependency Inversion Principle
	podService    PodServiceInterface
	healthService HealthServiceInterface

	// Cross-cutting concerns injected as interfaces
	validator     InputValidator
	converter     TypeConverter
	errorHandler  ErrorHandler
	auditLogger   AuditLogger
}

// ResolverConfig holds configuration for creating a new resolver
// Using the Builder pattern to make resolver creation more flexible
type ResolverConfig struct {
	PodService    PodServiceInterface
	HealthService HealthServiceInterface
	Validator     InputValidator
	Converter     TypeConverter
	ErrorHandler  ErrorHandler
	AuditLogger   AuditLogger
}

// NewResolver creates a new resolver with injected dependencies
// This constructor enforces dependency injection and makes testing easier
func NewResolver(config ResolverConfig) *Resolver {
	return &Resolver{
		podService:    config.PodService,
		healthService: config.HealthService,
		validator:     config.Validator,
		converter:     config.Converter,
		errorHandler:  config.ErrorHandler,
		auditLogger:   config.AuditLogger,
	}
}

// NewDefaultResolver creates a resolver with default implementations
// Useful for production environments where you want sensible defaults
func NewDefaultResolver() *Resolver {
	// Create default implementations
	validator := NewDefaultInputValidator()
	converter := NewDefaultTypeConverter()
	errorHandler := NewDefaultErrorHandler()
	auditLogger := NewDefaultAuditLogger("k8s-pod-manager", true)
	healthService := NewCachedHealthService(
		NewDefaultHealthService("1.0.0"),
		30*time.Second, // Cache health checks for 30 seconds
	)

	// Initialize Kubernetes client from global client
	client := kubernetes.Client

	// Create service factory and registry
	factory := services.NewServiceFactory(client, services.ProductionServiceOptions())
	registry := factory.CreateServiceRegistry()

	// Create adapter with full registry to match interface
	podService := NewPodServiceAdapter(registry)

	return NewResolver(ResolverConfig{
		PodService:    podService,
		HealthService: healthService,
		Validator:     validator,
		Converter:     converter,
		ErrorHandler:  errorHandler,
		AuditLogger:   auditLogger,
	})
}

// NewTestResolver creates a resolver suitable for testing
// Uses no-op implementations to avoid side effects during testing
func NewTestResolver(podService PodServiceInterface) *Resolver {
	return NewResolver(ResolverConfig{
		PodService:    podService,
		HealthService: NewDefaultHealthService("test"),
		Validator:     NewDefaultInputValidator(),
		Converter:     NewDefaultTypeConverter(),
		ErrorHandler:  NewDefaultErrorHandler(),
		AuditLogger:   NewNoOpAuditLogger(),
	})
}

// Helper methods to access injected dependencies
// These provide controlled access to dependencies within resolver methods

func (r *Resolver) getPodService() PodServiceInterface {
	return r.podService
}

func (r *Resolver) getHealthService() HealthServiceInterface {
	return r.healthService
}

func (r *Resolver) getValidator() InputValidator {
	return r.validator
}

func (r *Resolver) getConverter() TypeConverter {
	return r.converter
}

func (r *Resolver) getErrorHandler() ErrorHandler {
	return r.errorHandler
}

func (r *Resolver) getAuditLogger() AuditLogger {
	return r.auditLogger
}

// PodServiceAdapter adapts services to PodServiceInterface
type PodServiceAdapter struct {
	podService         services.PodService
	namespaceService   services.NamespaceService
	deploymentService  services.DeploymentService
	statefulSetService services.StatefulSetService
	client             kubernetes.ClientInterface
}

// NewPodServiceAdapter creates a new adapter with full service registry
func NewPodServiceAdapter(registry *services.ServiceRegistry) PodServiceInterface {
	return &PodServiceAdapter{
		podService:         registry.PodService,
		namespaceService:   registry.NamespaceService,
		deploymentService:  registry.DeploymentService,
		statefulSetService: registry.StatefulSetService,
		client:             registry.KubernetesClient,
	}
}

// Implement PodServiceInterface methods by delegating to the wrapped services
func (a *PodServiceAdapter) ListPods(ctx context.Context, namespace string) ([]services.PodInfo, error) {
	opts := &services.ListOptions{} // Use default list options
	return a.podService.ListPods(ctx, namespace, opts)
}

func (a *PodServiceAdapter) ListAllPods(ctx context.Context, namespace *string) ([]services.PodInfo, error) {
	opts := &services.ListOptions{} // Use default list options
	if namespace != nil {
		opts.Namespace = *namespace
	}
	return a.podService.ListAllPods(ctx, opts)
}

func (a *PodServiceAdapter) GetPod(ctx context.Context, namespace, name string) (*services.PodInfo, error) {
	return a.podService.GetPod(ctx, namespace, name)
}

func (a *PodServiceAdapter) RestartPod(ctx context.Context, namespace, name, user string) (map[string]interface{}, error) {
	result, err := a.podService.RestartPod(ctx, namespace, name, user)
	if err != nil {
		return nil, err
	}

	// Convert RestartResult to map[string]interface{}
	resultMap := map[string]interface{}{
		"message":           result.Message,
		"success":           result.Success,
		"method":            result.Method,
		"controller_type":   result.ControllerType,
	}

	if result.ControllerName != "" {
		resultMap["controller_name"] = result.ControllerName
	}
	if result.Details != nil {
		resultMap["details"] = result.Details
	}
	if result.Warning != "" {
		resultMap["warning"] = result.Warning
	}

	return resultMap, nil
}

func (a *PodServiceAdapter) DeletePod(ctx context.Context, namespace, name, user string) error {
	return a.podService.DeletePod(ctx, namespace, name, user)
}

func (a *PodServiceAdapter) ListNamespaces(ctx context.Context) ([]services.NamespaceInfo, error) {
	opts := &services.ListOptions{} // Use default list options
	return a.namespaceService.ListNamespaces(ctx, opts)
}

func (a *PodServiceAdapter) ListDeployments(ctx context.Context, namespace string) (*appsv1.DeploymentList, error) {
	return a.client.AppsV1().Deployments(namespace).List(ctx, metav1.ListOptions{})
}

func (a *PodServiceAdapter) ScaleDeployment(ctx context.Context, namespace, name string, replicas int32, user string) (map[string]interface{}, error) {
	result, err := a.deploymentService.ScaleDeployment(ctx, namespace, name, replicas, user)
	if err != nil {
		return nil, err
	}

	// Convert ScaleResult to map
	return map[string]interface{}{
		"message":           result.Message,
		"success":           result.Success,
		"previous_replicas": result.PreviousReplicas,
		"new_replicas":      result.NewReplicas,
		"resource_name":     result.ResourceName,
		"resource_type":     result.ResourceType,
	}, nil
}

func (a *PodServiceAdapter) ListStatefulSets(ctx context.Context, namespace string) (*appsv1.StatefulSetList, error) {
	return a.client.AppsV1().StatefulSets(namespace).List(ctx, metav1.ListOptions{})
}

func (a *PodServiceAdapter) ScaleStatefulSet(ctx context.Context, namespace, name string, replicas int32, user string) (map[string]interface{}, error) {
	result, err := a.statefulSetService.ScaleStatefulSet(ctx, namespace, name, replicas, user)
	if err != nil {
		return nil, err
	}

	// Convert ScaleResult to map
	return map[string]interface{}{
		"message":           result.Message,
		"success":           result.Success,
		"previous_replicas": result.PreviousReplicas,
		"new_replicas":      result.NewReplicas,
		"resource_name":     result.ResourceName,
		"resource_type":     result.ResourceType,
	}, nil
}

func (a *PodServiceAdapter) WatchPods(ctx context.Context, namespace string, eventsChan chan<- services.PodWatchEvent) error {
	opts := &services.WatchOptions{Namespace: namespace}
	return a.podService.WatchPods(ctx, namespace, opts, eventsChan)
}

func (a *PodServiceAdapter) WatchAllPods(ctx context.Context, eventsChan chan<- services.PodWatchEvent) error {
	opts := &services.WatchOptions{}
	return a.podService.WatchAllPods(ctx, opts, eventsChan)
}

func (a *PodServiceAdapter) StreamLogs(ctx context.Context, namespace, podName string, opts *services.LogStreamOptions, logsChan chan<- services.PodLogLine) error {
	return a.podService.StreamLogs(ctx, namespace, podName, opts, logsChan)
}
