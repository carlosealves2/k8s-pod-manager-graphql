package services

import (
	"context"

	"github.com/carlosf/k8s-pod-manager/internal/kubernetes"
)

// ServiceFactory creates and configures all services with proper dependency injection
type ServiceFactory struct {
	client  kubernetes.ClientInterface
	options ServiceOptions
}

// NewServiceFactory creates a new service factory
func NewServiceFactory(client kubernetes.ClientInterface, options ServiceOptions) *ServiceFactory {
	return &ServiceFactory{
		client:  client,
		options: options,
	}
}

// CreateServiceRegistry creates and wires up all services
func (f *ServiceFactory) CreateServiceRegistry() *ServiceRegistry {
	// Create utilities first
	controllerDetector := NewKubernetesControllerDetector(f.client)
	restartStrategy := NewDefaultRestartStrategy()
	podConverter := NewPodConverter(f.options.EnableMetrics)

	// Create audit service (stub implementation - would be replaced with real implementation)
	auditService := &stubAuditService{}

	// Create individual services
	deploymentService := NewDeploymentService(f.client, auditService, f.options)
	statefulSetService := NewStatefulSetService(f.client, auditService, f.options)
	namespaceService := NewNamespaceService(f.client, f.options)

	// Create restart orchestrator with dependencies
	restartOrchestrator := NewRestartOrchestrator(
		f.client,
		controllerDetector,
		restartStrategy,
		deploymentService,
		statefulSetService,
	)

	// Create pod service with all dependencies
	podService := NewPodService(
		f.client,
		controllerDetector,
		restartOrchestrator,
		podConverter,
		auditService,
		f.options,
	)

	return &ServiceRegistry{
		PodService:         podService,
		DeploymentService:  deploymentService,
		StatefulSetService: statefulSetService,
		NamespaceService:   namespaceService,
		AuditService:       auditService,

		// Utilities
		ControllerDetector:  controllerDetector,
		RestartOrchestrator: restartOrchestrator,
		PodConverter:        podConverter,

		// Infrastructure
		KubernetesClient: f.client,
	}
}

// DefaultServiceOptions returns sensible default options
func DefaultServiceOptions() ServiceOptions {
	return ServiceOptions{
		EnableMetrics:      true,
		EnableAudit:        true,
		DefaultTimeout:     30,
		MaxRetries:         3,
		CacheEnabled:       false,
		CacheTTL:           300, // 5 minutes
	}
}

// ProductionServiceOptions returns production-ready options
func ProductionServiceOptions() ServiceOptions {
	return ServiceOptions{
		EnableMetrics:      true,
		EnableAudit:        true,
		DefaultTimeout:     60,
		MaxRetries:         5,
		CacheEnabled:       true,
		CacheTTL:           300,
	}
}

// TestServiceOptions returns options suitable for testing
func TestServiceOptions() ServiceOptions {
	return ServiceOptions{
		EnableMetrics:      false,
		EnableAudit:        false,
		DefaultTimeout:     5,
		MaxRetries:         1,
		CacheEnabled:       false,
		CacheTTL:           0,
	}
}

// stubAuditService is a stub implementation for demonstration
// In practice, this would be replaced with the real audit service from refactored modules
type stubAuditService struct{}

func (s *stubAuditService) LogPodOperation(ctx context.Context, operation PodOperationAudit) error {
	// TODO: Replace with real audit service implementation
	return nil
}

func (s *stubAuditService) LogScaleOperation(ctx context.Context, operation ScaleOperationAudit) error {
	// TODO: Replace with real audit service implementation
	return nil
}

func (s *stubAuditService) GetOperationHistory(ctx context.Context, resourceType, namespace, name string, limit int) ([]OperationAudit, error) {
	// TODO: Replace with real audit service implementation
	return nil, nil
}