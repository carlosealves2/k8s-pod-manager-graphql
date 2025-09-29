package services

import (
	"context"
	"fmt"
	"time"

	"github.com/carlosf/k8s-pod-manager/internal/kubernetes"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

// restartOrchestratorImpl implements RestartOrchestrator interface
type restartOrchestratorImpl struct {
	client             kubernetes.ClientInterface
	controllerDetector ControllerDetector
	restartStrategy    RestartStrategy
	deploymentService  DeploymentService
	statefulSetService StatefulSetService
}

// NewRestartOrchestrator creates a new restart orchestrator
func NewRestartOrchestrator(
	client kubernetes.ClientInterface,
	controllerDetector ControllerDetector,
	restartStrategy RestartStrategy,
	deploymentService DeploymentService,
	statefulSetService StatefulSetService,
) RestartOrchestrator {
	return &restartOrchestratorImpl{
		client:             client,
		controllerDetector: controllerDetector,
		restartStrategy:    restartStrategy,
		deploymentService:  deploymentService,
		statefulSetService: statefulSetService,
	}
}

// RestartPod implements RestartOrchestrator
func (r *restartOrchestratorImpl) RestartPod(ctx context.Context, namespace, podName, executedBy string) (*RestartResult, error) {
	// First, get the pod to determine its controller
	pod, err := r.client.CoreV1().Pods(namespace).Get(ctx, podName, metav1.GetOptions{})
	if err != nil {
		return nil, NewNotFoundError("pod", fmt.Sprintf("%s/%s", namespace, podName))
	}

	// Detect the controller
	controllerInfo := r.controllerDetector.DetectController(ctx, *pod)

	// Check if restart is supported
	if !r.CanRestart(controllerInfo.Type) {
		return &RestartResult{
			Success:        false,
			Message:        fmt.Sprintf("Restart not supported for controller type: %s", controllerInfo.Type),
			Method:         RestartMethodWarning,
			ControllerType: controllerInfo.Type,
			ControllerName: controllerInfo.Name,
			Warning:        "This controller type does not support restart operations",
		}, NewConflictError(fmt.Sprintf("restart not supported for controller type: %s", controllerInfo.Type))
	}

	// Determine restart method
	restartMethod := r.GetRestartMethod(controllerInfo.Type)

	// Execute the appropriate restart strategy
	result, err := r.executeRestart(ctx, namespace, podName, controllerInfo, restartMethod, executedBy)
	if err != nil {
		return &RestartResult{
			Success:        false,
			Message:        err.Error(),
			Method:         restartMethod,
			ControllerType: controllerInfo.Type,
			ControllerName: controllerInfo.Name,
		}, err
	}

	result.Success = true
	result.Method = restartMethod
	result.ControllerType = controllerInfo.Type
	result.ControllerName = controllerInfo.Name

	return result, nil
}

// CanRestart implements RestartOrchestrator
func (r *restartOrchestratorImpl) CanRestart(controllerType ControllerType) bool {
	return r.restartStrategy.CanRestart(controllerType)
}

// GetRestartMethod implements RestartOrchestrator
func (r *restartOrchestratorImpl) GetRestartMethod(controllerType ControllerType) RestartMethod {
	return r.restartStrategy.GetRestartMethod(controllerType)
}

// executeRestart handles the actual restart logic based on the method
func (r *restartOrchestratorImpl) executeRestart(
	ctx context.Context,
	namespace, podName string,
	controllerInfo ControllerInfo,
	method RestartMethod,
	executedBy string,
) (*RestartResult, error) {
	switch method {
	case RestartMethodRollout:
		return r.executeRolloutRestart(ctx, namespace, controllerInfo, executedBy)
	case RestartMethodDelete:
		return r.executeDeleteRestart(ctx, namespace, podName, controllerInfo)
	case RestartMethodWarning:
		return r.executeWarningRestart(ctx, namespace, podName, controllerInfo)
	default:
		return nil, NewInternalError(fmt.Sprintf("unsupported restart method: %s", method), nil)
	}
}

func (r *restartOrchestratorImpl) executeRolloutRestart(
	ctx context.Context,
	namespace string,
	controllerInfo ControllerInfo,
	executedBy string,
) (*RestartResult, error) {
	switch controllerInfo.Type {
	case ControllerTypeDeployment:
		err := r.rolloutRestartDeployment(ctx, namespace, controllerInfo.Name)
		if err != nil {
			return nil, NewPodOperationError("rollout-restart", controllerInfo.Name, namespace, "failed to restart deployment", err)
		}

		return &RestartResult{
			Message: "Rollout restart initiated for deployment",
			Details: map[string]interface{}{
				"deployment": controllerInfo.Name,
				"namespace":  namespace,
				"method":     "rollout-restart",
			},
		}, nil

	case ControllerTypeStatefulSet:
		err := r.rolloutRestartStatefulSet(ctx, namespace, controllerInfo.Name)
		if err != nil {
			return nil, NewPodOperationError("rollout-restart", controllerInfo.Name, namespace, "failed to restart statefulset", err)
		}

		return &RestartResult{
			Message: "Rollout restart initiated for statefulset",
			Details: map[string]interface{}{
				"statefulset": controllerInfo.Name,
				"namespace":   namespace,
				"method":      "rollout-restart",
			},
		}, nil

	default:
		return nil, NewConflictError(fmt.Sprintf("rollout restart not supported for controller type: %s", controllerInfo.Type))
	}
}

func (r *restartOrchestratorImpl) executeDeleteRestart(
	ctx context.Context,
	namespace, podName string,
	controllerInfo ControllerInfo,
) (*RestartResult, error) {
	err := r.client.CoreV1().Pods(namespace).Delete(ctx, podName, metav1.DeleteOptions{})
	if err != nil {
		return nil, NewPodOperationError("delete-restart", podName, namespace, "failed to delete pod", err)
	}

	return &RestartResult{
		Message: fmt.Sprintf("Pod deleted for recreation by %s", controllerInfo.Type),
		Details: map[string]interface{}{
			"pod":        podName,
			"namespace":  namespace,
			"controller": controllerInfo.Name,
			"type":       string(controllerInfo.Type),
			"method":     "delete",
		},
	}, nil
}

func (r *restartOrchestratorImpl) executeWarningRestart(
	ctx context.Context,
	namespace, podName string,
	controllerInfo ControllerInfo,
) (*RestartResult, error) {
	err := r.client.CoreV1().Pods(namespace).Delete(ctx, podName, metav1.DeleteOptions{})
	if err != nil {
		return nil, NewPodOperationError("warning-restart", podName, namespace, "failed to delete pod", err)
	}

	warning := "Pod deleted but will not be automatically recreated"
	if controllerInfo.Type == ControllerTypeStandalone {
		warning = "Standalone pod deleted - no controller will recreate it"
	}

	return &RestartResult{
		Message: "Pod deleted (manual intervention may be required)",
		Warning: warning,
		Details: map[string]interface{}{
			"pod":       podName,
			"namespace": namespace,
			"type":      string(controllerInfo.Type),
			"method":    "delete-with-warning",
		},
	}, nil
}

func (r *restartOrchestratorImpl) rolloutRestartDeployment(ctx context.Context, namespace, name string) error {
	// Add restart annotation to trigger rollout
	patch := []byte(fmt.Sprintf(`{
		"spec": {
			"template": {
				"metadata": {
					"annotations": {
						"kubectl.kubernetes.io/restartedAt": "%s"
					}
				}
			}
		}
	}`, time.Now().Format(time.RFC3339)))

	_, err := r.client.AppsV1().Deployments(namespace).Patch(
		ctx, name, types.StrategicMergePatchType, patch, metav1.PatchOptions{})
	return err
}

func (r *restartOrchestratorImpl) rolloutRestartStatefulSet(ctx context.Context, namespace, name string) error {
	// Add restart annotation to trigger rollout
	patch := []byte(fmt.Sprintf(`{
		"spec": {
			"template": {
				"metadata": {
					"annotations": {
						"kubectl.kubernetes.io/restartedAt": "%s"
					}
				}
			}
		}
	}`, time.Now().Format(time.RFC3339)))

	_, err := r.client.AppsV1().StatefulSets(namespace).Patch(
		ctx, name, types.StrategicMergePatchType, patch, metav1.PatchOptions{})
	return err
}