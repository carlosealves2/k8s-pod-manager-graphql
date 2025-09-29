package graph

import (
	"context"

	"github.com/carlosf/k8s-pod-manager/graph/model"
	"github.com/carlosf/k8s-pod-manager/internal/services"
	appsv1 "k8s.io/api/apps/v1"
)

// PodServiceInterface defines the contract for pod operations
// This interface follows the Interface Segregation Principle by grouping related operations
type PodServiceInterface interface {
	// Pod operations
	ListPods(ctx context.Context, namespace string) ([]services.PodInfo, error)
	ListAllPods(ctx context.Context, namespace *string) ([]services.PodInfo, error)
	GetPod(ctx context.Context, namespace, name string) (*services.PodInfo, error)
	RestartPod(ctx context.Context, namespace, name, user string) (map[string]interface{}, error)
	DeletePod(ctx context.Context, namespace, name, user string) error

	// Namespace operations
	ListNamespaces(ctx context.Context) ([]services.NamespaceInfo, error)

	// Deployment operations
	ListDeployments(ctx context.Context, namespace string) (*appsv1.DeploymentList, error)
	ScaleDeployment(ctx context.Context, namespace, name string, replicas int32, user string) (map[string]interface{}, error)

	// StatefulSet operations
	ListStatefulSets(ctx context.Context, namespace string) (*appsv1.StatefulSetList, error)
	ScaleStatefulSet(ctx context.Context, namespace, name string, replicas int32, user string) (map[string]interface{}, error)

	// Watch operations
	WatchPods(ctx context.Context, namespace string, eventsChan chan<- services.PodWatchEvent) error
	WatchAllPods(ctx context.Context, eventsChan chan<- services.PodWatchEvent) error
}

// HealthServiceInterface defines the contract for health check operations
// Separated from pod operations following Single Responsibility Principle
type HealthServiceInterface interface {
	GetHealth() *model.HealthResponse
	GetReadiness() *model.ReadinessResponse
	GetSystemInfo() *model.InfoResponse
}

// InputValidator defines the contract for validating GraphQL inputs
// This interface allows for different validation strategies (Strategy Pattern)
type InputValidator interface {
	ValidateNamespace(namespace string) error
	ValidatePodName(name string) error
	ValidateScaleInput(input model.ScaleInput) error
	ValidateUserInput(user *string) error
}

// TypeConverter defines the contract for converting between service and GraphQL types
// This interface encapsulates conversion logic and makes it easily testable
type TypeConverter interface {
	ConvertPodInfo(servicePod services.PodInfo) *model.PodInfo
	ConvertNamespaceInfo(serviceNS services.NamespaceInfo) *model.NamespaceInfo
	ConvertDeploymentInfo(deployment appsv1.Deployment) *model.DeploymentInfo
	ConvertStatefulSetInfo(statefulSet appsv1.StatefulSet) *model.StatefulSetInfo
	ConvertPodWatchEvent(serviceEvent services.PodWatchEvent) *model.PodWatchEvent
	MapToKeyValuePairs(m map[string]string) []*model.KeyValue
}

// ErrorHandler defines the contract for handling and formatting GraphQL errors
// This provides consistent error responses across the API
type ErrorHandler interface {
	HandleServiceError(err error, operation string) error
	HandleValidationError(err error, field string) error
	HandleNotFoundError(resource, namespace, name string) error
	HandleInternalError(err error, context string) error
}

// AuditLogger defines the contract for audit logging GraphQL operations
// This interface allows for different audit logging implementations
type AuditLogger interface {
	LogMutation(ctx context.Context, operation, namespace, resource, user string, success bool, error string)
	LogQuery(ctx context.Context, operation, namespace string, resourceCount int)
	LogSubscription(ctx context.Context, operation, namespace string, started bool)
}