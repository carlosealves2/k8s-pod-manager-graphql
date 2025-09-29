package services

import (
	"context"
	"fmt"
	"time"

	"github.com/carlosf/k8s-pod-manager/internal/kubernetes"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

// deploymentServiceImpl implements DeploymentService interface
type deploymentServiceImpl struct {
	client       kubernetes.ClientInterface
	auditService AuditService
	options      ServiceOptions
}

// NewDeploymentService creates a new deployment service
func NewDeploymentService(
	client kubernetes.ClientInterface,
	auditService AuditService,
	options ServiceOptions,
) DeploymentService {
	return &deploymentServiceImpl{
		client:       client,
		auditService: auditService,
		options:      options,
	}
}

// GetDeployment implements DeploymentService
func (s *deploymentServiceImpl) GetDeployment(ctx context.Context, namespace, name string) (*appsv1.Deployment, error) {
	if namespace == "" || name == "" {
		return nil, NewValidationError("namespace and name are required", nil)
	}

	deployment, err := s.client.AppsV1().Deployments(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, NewNotFoundError("deployment", fmt.Sprintf("%s/%s", namespace, name))
	}

	return deployment, nil
}

// ListDeployments implements DeploymentService
func (s *deploymentServiceImpl) ListDeployments(ctx context.Context, namespace string, opts *ListOptions) (*appsv1.DeploymentList, error) {
	listOpts := metav1.ListOptions{}
	if opts != nil {
		listOpts = s.buildListOptions(opts)
	}

	deployments, err := s.client.AppsV1().Deployments(namespace).List(ctx, listOpts)
	if err != nil {
		return nil, NewInternalError(
			fmt.Sprintf("failed to list deployments in namespace '%s'", namespace),
			err,
		)
	}

	return deployments, nil
}

// ScaleDeployment implements DeploymentService
func (s *deploymentServiceImpl) ScaleDeployment(ctx context.Context, namespace, deploymentName string, replicas int32, executedBy string) (*ScaleResult, error) {
	if err := s.validateScaleParams(namespace, deploymentName, executedBy); err != nil {
		return nil, err
	}

	if replicas < 0 {
		return nil, NewValidationError("replicas must be non-negative", nil)
	}

	// Get current deployment
	deployment, err := s.client.AppsV1().Deployments(namespace).Get(ctx, deploymentName, metav1.GetOptions{})
	if err != nil {
		return nil, NewNotFoundError("deployment", fmt.Sprintf("%s/%s", namespace, deploymentName))
	}

	previousReplicas := int32(0)
	if deployment.Spec.Replicas != nil {
		previousReplicas = *deployment.Spec.Replicas
	}

	// No change needed
	if previousReplicas == replicas {
		return &ScaleResult{
			Success:          true,
			Message:          "No scaling required - already at desired replicas",
			ResourceType:     "deployment",
			ResourceName:     deploymentName,
			Namespace:        namespace,
			PreviousReplicas: previousReplicas,
			NewReplicas:      replicas,
			Action:           "no-change",
		}, nil
	}

	// Determine action type
	action := s.determineScaleAction(previousReplicas, replicas)

	// Create audit record
	scaleAudit := ScaleOperationAudit{
		ResourceType:     "deployment",
		ResourceName:     deploymentName,
		Namespace:        namespace,
		PreviousReplicas: previousReplicas,
		NewReplicas:      replicas,
		ExecutedBy:       executedBy,
		Status:           "in-progress",
		Reason:           action,
	}

	// Perform the scaling
	deployment.Spec.Replicas = &replicas
	updatedDeployment, err := s.client.AppsV1().Deployments(namespace).Update(ctx, deployment, metav1.UpdateOptions{})
	if err != nil {
		scaleAudit.Status = "failed"
		if s.options.EnableAudit {
			s.auditService.LogScaleOperation(ctx, scaleAudit)
		}
		return nil, NewInternalError(fmt.Sprintf("failed to scale deployment %s/%s", namespace, deploymentName), err)
	}

	// Success audit
	scaleAudit.Status = "success"
	if s.options.EnableAudit {
		s.auditService.LogScaleOperation(ctx, scaleAudit)
	}

	return &ScaleResult{
		Success:          true,
		Message:          fmt.Sprintf("Deployment scaled successfully from %d to %d replicas", previousReplicas, replicas),
		ResourceType:     "deployment",
		ResourceName:     deploymentName,
		Namespace:        namespace,
		PreviousReplicas: previousReplicas,
		NewReplicas:      *updatedDeployment.Spec.Replicas,
		Action:           action,
	}, nil
}

// RolloutRestart implements DeploymentService
func (s *deploymentServiceImpl) RolloutRestart(ctx context.Context, namespace, deploymentName string, executedBy string) error {
	if err := s.validateParams(namespace, deploymentName, executedBy); err != nil {
		return err
	}

	// Create restart annotation patch
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

	_, err := s.client.AppsV1().Deployments(namespace).Patch(
		ctx, deploymentName, types.StrategicMergePatchType, patch, metav1.PatchOptions{})
	if err != nil {
		return NewInternalError(
			fmt.Sprintf("failed to restart deployment %s/%s", namespace, deploymentName),
			err,
		)
	}

	// Audit the restart
	if s.options.EnableAudit {
		restartAudit := PodOperationAudit{
			PodName:        "deployment-restart",
			Namespace:      namespace,
			Operation:      "rollout-restart",
			Controller:     "Deployment",
			ControllerName: deploymentName,
			ExecutedBy:     executedBy,
			Status:         "success",
			Message:        "Deployment rollout restart initiated",
		}
		s.auditService.LogPodOperation(ctx, restartAudit)
	}

	return nil
}

// Helper methods

func (s *deploymentServiceImpl) validateScaleParams(namespace, deploymentName, executedBy string) error {
	if namespace == "" {
		return NewValidationError("namespace is required", nil)
	}
	if deploymentName == "" {
		return NewValidationError("deployment name is required", nil)
	}
	if executedBy == "" {
		return NewValidationError("executedBy is required for audit trail", nil)
	}
	return nil
}

func (s *deploymentServiceImpl) validateParams(namespace, deploymentName, executedBy string) error {
	return s.validateScaleParams(namespace, deploymentName, executedBy)
}

func (s *deploymentServiceImpl) determineScaleAction(previous, new int32) string {
	switch {
	case new == 0:
		return "stopping"
	case previous == 0:
		return "starting"
	case new > previous:
		return fmt.Sprintf("scaling-up-from-%d-to-%d", previous, new)
	case new < previous:
		return fmt.Sprintf("scaling-down-from-%d-to-%d", previous, new)
	default:
		return "no-change"
	}
}

func (s *deploymentServiceImpl) buildListOptions(opts *ListOptions) metav1.ListOptions {
	listOpts := metav1.ListOptions{}

	if opts.Limit > 0 {
		listOpts.Limit = opts.Limit
	}

	if opts.Continue != "" {
		listOpts.Continue = opts.Continue
	}

	// Build label selector
	if len(opts.LabelFilter) > 0 {
		labelSelector := ""
		for key, value := range opts.LabelFilter {
			if labelSelector != "" {
				labelSelector += ","
			}
			labelSelector += fmt.Sprintf("%s=%s", key, value)
		}
		listOpts.LabelSelector = labelSelector
	}

	// Build field selector
	if len(opts.FieldFilter) > 0 {
		fieldSelector := ""
		for key, value := range opts.FieldFilter {
			if fieldSelector != "" {
				fieldSelector += ","
			}
			fieldSelector += fmt.Sprintf("%s=%s", key, value)
		}
		listOpts.FieldSelector = fieldSelector
	}

	return listOpts
}