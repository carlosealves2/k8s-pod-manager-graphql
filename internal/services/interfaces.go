package services

import (
	"context"

	"github.com/carlosf/k8s-pod-manager/internal/kubernetes"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
)

// PodService defines the interface for pod-related operations
// Following Single Responsibility Principle - only handles pod operations
type PodService interface {
	// Core pod operations
	ListPods(ctx context.Context, namespace string, opts *ListOptions) ([]PodInfo, error)
	ListAllPods(ctx context.Context, opts *ListOptions) ([]PodInfo, error)
	GetPod(ctx context.Context, namespace, name string) (*PodInfo, error)
	DeletePod(ctx context.Context, namespace, podName, executedBy string) error
	RestartPod(ctx context.Context, namespace, podName, executedBy string) (*RestartResult, error)

	// Pod watching
	WatchPods(ctx context.Context, namespace string, opts *WatchOptions, eventsChan chan<- PodWatchEvent) error
	WatchAllPods(ctx context.Context, opts *WatchOptions, eventsChan chan<- PodWatchEvent) error

	// Pod logs streaming
	StreamLogs(ctx context.Context, namespace, podName string, opts *LogStreamOptions, logsChan chan<- PodLogLine) error
}

// DeploymentService defines the interface for deployment-related operations
type DeploymentService interface {
	GetDeployment(ctx context.Context, namespace, name string) (*appsv1.Deployment, error)
	ListDeployments(ctx context.Context, namespace string, opts *ListOptions) (*appsv1.DeploymentList, error)
	ScaleDeployment(ctx context.Context, namespace, deploymentName string, replicas int32, executedBy string) (*ScaleResult, error)
	RolloutRestart(ctx context.Context, namespace, deploymentName string, executedBy string) error
}

// StatefulSetService defines the interface for statefulset-related operations
type StatefulSetService interface {
	GetStatefulSet(ctx context.Context, namespace, name string) (*appsv1.StatefulSet, error)
	ListStatefulSets(ctx context.Context, namespace string, opts *ListOptions) (*appsv1.StatefulSetList, error)
	ScaleStatefulSet(ctx context.Context, namespace, statefulSetName string, replicas int32, executedBy string) (*ScaleResult, error)
	RolloutRestart(ctx context.Context, namespace, statefulSetName string, executedBy string) error
}

// NamespaceService defines the interface for namespace-related operations
type NamespaceService interface {
	ListNamespaces(ctx context.Context, opts *ListOptions) ([]NamespaceInfo, error)
	GetNamespace(ctx context.Context, name string) (*NamespaceInfo, error)
}

// AuditService defines the interface for audit operations
type AuditService interface {
	LogPodOperation(ctx context.Context, operation PodOperationAudit) error
	LogScaleOperation(ctx context.Context, operation ScaleOperationAudit) error
	GetOperationHistory(ctx context.Context, resourceType, namespace, name string, limit int) ([]OperationAudit, error)
}

// PodConverter handles conversion between Kubernetes pods and service types
type PodConverter interface {
	ConvertPod(pod corev1.Pod, controllerInfo ControllerInfo) PodInfo
	ConvertPods(pods []corev1.Pod) []PodInfo
	ExtractContainerInfo(pod corev1.Pod) []ContainerInfo
	CalculateResourceUsage(pod corev1.Pod) *ResourceUsageInfo
}

// RestartOrchestrator coordinates pod restart operations across different controller types
type RestartOrchestrator interface {
	RestartPod(ctx context.Context, namespace, podName, executedBy string) (*RestartResult, error)
	CanRestart(controllerType ControllerType) bool
	GetRestartMethod(controllerType ControllerType) RestartMethod
}

// ServiceRegistry holds all service dependencies for dependency injection
type ServiceRegistry struct {
	PodService         PodService
	DeploymentService  DeploymentService
	StatefulSetService StatefulSetService
	NamespaceService   NamespaceService
	AuditService       AuditService

	// Utilities
	ControllerDetector  ControllerDetector
	RestartOrchestrator RestartOrchestrator
	PodConverter        PodConverter

	// Infrastructure
	KubernetesClient kubernetes.ClientInterface
}

// ServiceOptions defines configuration for service creation
type ServiceOptions struct {
	EnableMetrics      bool
	EnableAudit        bool
	DefaultTimeout     int
	MaxRetries         int
	CacheEnabled       bool
	CacheTTL           int
}

// Audit types for operations
type PodOperationAudit struct {
	PodName        string
	Namespace      string
	Operation      string
	Controller     string
	ControllerName string
	ExecutedBy     string
	Status         string
	Message        string
	Details        map[string]interface{}
}

type ScaleOperationAudit struct {
	ResourceType     string
	ResourceName     string
	Namespace        string
	PreviousReplicas int32
	NewReplicas      int32
	ExecutedBy       string
	Status           string
	Reason           string
}

type OperationAudit struct {
	ID           string
	Type         string
	ResourceType string
	ResourceName string
	Namespace    string
	ExecutedBy   string
	Status       string
	Message      string
	Details      map[string]interface{}
	Timestamp    string
}