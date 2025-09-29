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

// statefulSetServiceImpl implements StatefulSetService interface
type statefulSetServiceImpl struct {
	client       kubernetes.ClientInterface
	auditService AuditService
	options      ServiceOptions
}

// NewStatefulSetService creates a new statefulset service
func NewStatefulSetService(
	client kubernetes.ClientInterface,
	auditService AuditService,
	options ServiceOptions,
) StatefulSetService {
	return &statefulSetServiceImpl{
		client:       client,
		auditService: auditService,
		options:      options,
	}
}

// GetStatefulSet implements StatefulSetService
func (s *statefulSetServiceImpl) GetStatefulSet(ctx context.Context, namespace, name string) (*appsv1.StatefulSet, error) {
	if namespace == "" || name == "" {
		return nil, NewValidationError("namespace and name are required", nil)
	}

	statefulSet, err := s.client.AppsV1().StatefulSets(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, NewNotFoundError("statefulset", fmt.Sprintf("%s/%s", namespace, name))
	}

	return statefulSet, nil
}

// ListStatefulSets implements StatefulSetService
func (s *statefulSetServiceImpl) ListStatefulSets(ctx context.Context, namespace string, opts *ListOptions) (*appsv1.StatefulSetList, error) {
	listOpts := metav1.ListOptions{}
	if opts != nil {
		listOpts = s.buildListOptions(opts)
	}

	statefulSets, err := s.client.AppsV1().StatefulSets(namespace).List(ctx, listOpts)
	if err != nil {
		return nil, NewInternalError(
			fmt.Sprintf("failed to list statefulsets in namespace '%s'", namespace),
			err,
		)
	}

	return statefulSets, nil
}

// ScaleStatefulSet implements StatefulSetService
func (s *statefulSetServiceImpl) ScaleStatefulSet(ctx context.Context, namespace, statefulSetName string, replicas int32, executedBy string) (*ScaleResult, error) {
	if err := s.validateScaleParams(namespace, statefulSetName, executedBy); err != nil {
		return nil, err
	}

	if replicas < 0 {
		return nil, NewValidationError("replicas must be non-negative", nil)
	}

	// Get current statefulset
	statefulSet, err := s.client.AppsV1().StatefulSets(namespace).Get(ctx, statefulSetName, metav1.GetOptions{})
	if err != nil {
		return nil, NewNotFoundError("statefulset", fmt.Sprintf("%s/%s", namespace, statefulSetName))
	}

	previousReplicas := int32(0)
	if statefulSet.Spec.Replicas != nil {
		previousReplicas = *statefulSet.Spec.Replicas
	}

	// No change needed
	if previousReplicas == replicas {
		return &ScaleResult{
			Success:          true,
			Message:          "No scaling required - already at desired replicas",
			ResourceType:     "statefulset",
			ResourceName:     statefulSetName,
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
		ResourceType:     "statefulset",
		ResourceName:     statefulSetName,
		Namespace:        namespace,
		PreviousReplicas: previousReplicas,
		NewReplicas:      replicas,
		ExecutedBy:       executedBy,
		Status:           "in-progress",
		Reason:           action,
	}

	// Perform the scaling
	statefulSet.Spec.Replicas = &replicas
	updatedStatefulSet, err := s.client.AppsV1().StatefulSets(namespace).Update(ctx, statefulSet, metav1.UpdateOptions{})
	if err != nil {
		scaleAudit.Status = "failed"
		if s.options.EnableAudit {
			s.auditService.LogScaleOperation(ctx, scaleAudit)
		}
		return nil, NewInternalError(fmt.Sprintf("failed to scale statefulset %s/%s", namespace, statefulSetName), err)
	}

	// Success audit
	scaleAudit.Status = "success"
	if s.options.EnableAudit {
		s.auditService.LogScaleOperation(ctx, scaleAudit)
	}

	return &ScaleResult{
		Success:          true,
		Message:          fmt.Sprintf("StatefulSet scaled successfully from %d to %d replicas", previousReplicas, replicas),
		ResourceType:     "statefulset",
		ResourceName:     statefulSetName,
		Namespace:        namespace,
		PreviousReplicas: previousReplicas,
		NewReplicas:      *updatedStatefulSet.Spec.Replicas,
		Action:           action,
	}, nil
}

// RolloutRestart implements StatefulSetService
func (s *statefulSetServiceImpl) RolloutRestart(ctx context.Context, namespace, statefulSetName string, executedBy string) error {
	if err := s.validateParams(namespace, statefulSetName, executedBy); err != nil {
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

	_, err := s.client.AppsV1().StatefulSets(namespace).Patch(
		ctx, statefulSetName, types.StrategicMergePatchType, patch, metav1.PatchOptions{})
	if err != nil {
		return NewInternalError(
			fmt.Sprintf("failed to restart statefulset %s/%s", namespace, statefulSetName),
			err,
		)
	}

	// Audit the restart
	if s.options.EnableAudit {
		restartAudit := PodOperationAudit{
			PodName:        "statefulset-restart",
			Namespace:      namespace,
			Operation:      "rollout-restart",
			Controller:     "StatefulSet",
			ControllerName: statefulSetName,
			ExecutedBy:     executedBy,
			Status:         "success",
			Message:        "StatefulSet rollout restart initiated",
		}
		s.auditService.LogPodOperation(ctx, restartAudit)
	}

	return nil
}

// Helper methods

func (s *statefulSetServiceImpl) validateScaleParams(namespace, statefulSetName, executedBy string) error {
	if namespace == "" {
		return NewValidationError("namespace is required", nil)
	}
	if statefulSetName == "" {
		return NewValidationError("statefulset name is required", nil)
	}
	if executedBy == "" {
		return NewValidationError("executedBy is required for audit trail", nil)
	}
	return nil
}

func (s *statefulSetServiceImpl) validateParams(namespace, statefulSetName, executedBy string) error {
	return s.validateScaleParams(namespace, statefulSetName, executedBy)
}

func (s *statefulSetServiceImpl) determineScaleAction(previous, new int32) string {
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

func (s *statefulSetServiceImpl) buildListOptions(opts *ListOptions) metav1.ListOptions {
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