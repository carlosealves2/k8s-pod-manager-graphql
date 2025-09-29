package services

import (
	"context"
	"strings"

	"github.com/carlosf/k8s-pod-manager/internal/kubernetes"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ControllerType represents different types of Kubernetes controllers
type ControllerType string

const (
	ControllerTypeDeployment  ControllerType = "Deployment"
	ControllerTypeStatefulSet ControllerType = "StatefulSet"
	ControllerTypeDaemonSet   ControllerType = "DaemonSet"
	ControllerTypeReplicaSet  ControllerType = "ReplicaSet"
	ControllerTypeJob         ControllerType = "Job"
	ControllerTypeCronJob     ControllerType = "CronJob"
	ControllerTypeNode        ControllerType = "Node"
	ControllerTypeStandalone  ControllerType = "Standalone"
)

// ControllerInfo holds information about a pod's controller
type ControllerInfo struct {
	Type ControllerType
	Name string
	Kind string
}

// ControllerDetector defines the interface for detecting pod controllers
type ControllerDetector interface {
	// DetectController identifies the controller type and name for a given pod
	DetectController(ctx context.Context, pod corev1.Pod) ControllerInfo

	// GetControllerType returns just the controller type (legacy compatibility)
	GetControllerType(pod corev1.Pod) string

	// GetController returns controller kind and name (legacy compatibility)
	GetController(pod corev1.Pod) (kind, name string)
}

// KubernetesControllerDetector implements ControllerDetector using Kubernetes API
type KubernetesControllerDetector struct {
	client kubernetes.ClientInterface
}

// NewKubernetesControllerDetector creates a new controller detector
func NewKubernetesControllerDetector(client kubernetes.ClientInterface) *KubernetesControllerDetector {
	return &KubernetesControllerDetector{
		client: client,
	}
}

// DetectController implements the ControllerDetector interface
func (d *KubernetesControllerDetector) DetectController(ctx context.Context, pod corev1.Pod) ControllerInfo {
	// Check direct owner references first
	for _, owner := range pod.OwnerReferences {
		if owner.Controller != nil && *owner.Controller {
			switch owner.Kind {
			case "ReplicaSet":
				// Check if ReplicaSet is owned by a Deployment
				if deploymentName := d.getDeploymentFromReplicaSet(ctx, pod.Namespace, owner.Name); deploymentName != "" {
					return ControllerInfo{
						Type: ControllerTypeDeployment,
						Name: deploymentName,
						Kind: "Deployment",
					}
				}
				return ControllerInfo{
					Type: ControllerTypeReplicaSet,
					Name: owner.Name,
					Kind: owner.Kind,
				}
			case "Job":
				// Check if Job is owned by a CronJob
				if cronJobName := d.getCronJobFromJob(ctx, pod.Namespace, owner.Name); cronJobName != "" {
					return ControllerInfo{
						Type: ControllerTypeCronJob,
						Name: cronJobName,
						Kind: "CronJob",
					}
				}
				return ControllerInfo{
					Type: ControllerTypeJob,
					Name: owner.Name,
					Kind: owner.Kind,
				}
			case "DaemonSet":
				return ControllerInfo{
					Type: ControllerTypeDaemonSet,
					Name: owner.Name,
					Kind: owner.Kind,
				}
			case "StatefulSet":
				return ControllerInfo{
					Type: ControllerTypeStatefulSet,
					Name: owner.Name,
					Kind: owner.Kind,
				}
			case "CronJob":
				return ControllerInfo{
					Type: ControllerTypeCronJob,
					Name: owner.Name,
					Kind: owner.Kind,
				}
			default:
				return ControllerInfo{
					Type: ControllerType(owner.Kind),
					Name: owner.Name,
					Kind: owner.Kind,
				}
			}
		}
	}

	// Check for other patterns in labels that might indicate controller type
	if controllerInfo := d.detectFromLabels(pod); controllerInfo.Type != "" {
		return controllerInfo
	}

	// If no controller found, check if it's a node-managed pod
	if d.isNodeManagedPod(pod) {
		return ControllerInfo{
			Type: ControllerTypeNode,
			Name: pod.Spec.NodeName,
			Kind: "Node",
		}
	}

	// Default to standalone
	return ControllerInfo{
		Type: ControllerTypeStandalone,
		Name: "",
		Kind: "",
	}
}

// GetControllerType provides legacy compatibility
func (d *KubernetesControllerDetector) GetControllerType(pod corev1.Pod) string {
	info := d.DetectController(context.Background(), pod)
	return string(info.Type)
}

// GetController provides legacy compatibility
func (d *KubernetesControllerDetector) GetController(pod corev1.Pod) (kind, name string) {
	info := d.DetectController(context.Background(), pod)
	return info.Kind, info.Name
}

// getDeploymentFromReplicaSet checks if a ReplicaSet is owned by a Deployment
func (d *KubernetesControllerDetector) getDeploymentFromReplicaSet(ctx context.Context, namespace, replicaSetName string) string {
	rs, err := d.client.AppsV1().ReplicaSets(namespace).Get(ctx, replicaSetName, metav1.GetOptions{})
	if err != nil {
		return ""
	}

	for _, owner := range rs.OwnerReferences {
		if owner.Controller != nil && *owner.Controller && owner.Kind == "Deployment" {
			return owner.Name
		}
	}
	return ""
}

// getCronJobFromJob checks if a Job is owned by a CronJob
func (d *KubernetesControllerDetector) getCronJobFromJob(ctx context.Context, namespace, jobName string) string {
	job, err := d.client.BatchV1().Jobs(namespace).Get(ctx, jobName, metav1.GetOptions{})
	if err != nil {
		return ""
	}

	for _, owner := range job.OwnerReferences {
		if owner.Controller != nil && *owner.Controller && owner.Kind == "CronJob" {
			return owner.Name
		}
	}
	return ""
}

// detectFromLabels attempts to detect controller type from common labels
func (d *KubernetesControllerDetector) detectFromLabels(pod corev1.Pod) ControllerInfo {
	if pod.Labels == nil {
		return ControllerInfo{}
	}

	// Check for common controller labels
	if managedBy, ok := pod.Labels["app.kubernetes.io/managed-by"]; ok && managedBy != "" {
		return ControllerInfo{
			Type: ControllerType(strings.Title(managedBy)),
			Name: "",
			Kind: strings.Title(managedBy),
		}
	}

	// Check for job-related labels
	if jobName, ok := pod.Labels["job-name"]; ok {
		return ControllerInfo{
			Type: ControllerTypeJob,
			Name: jobName,
			Kind: "Job",
		}
	}

	// Check for batch job labels
	if jobName, ok := pod.Labels["batch.kubernetes.io/job-name"]; ok {
		return ControllerInfo{
			Type: ControllerTypeJob,
			Name: jobName,
			Kind: "Job",
		}
	}

	return ControllerInfo{}
}

// isNodeManagedPod checks if the pod is managed directly by the node (like static pods)
func (d *KubernetesControllerDetector) isNodeManagedPod(pod corev1.Pod) bool {
	// Static pods typically have the node name in their owner references
	for _, owner := range pod.OwnerReferences {
		if owner.Kind == "Node" {
			return true
		}
	}

	// Check for static pod annotations
	if pod.Annotations != nil {
		if _, isStatic := pod.Annotations["kubernetes.io/config.source"]; isStatic {
			return true
		}
	}

	return false
}

// RestartStrategy defines how different controller types should be restarted
type RestartStrategy interface {
	// CanRestart returns true if the controller type supports restart operations
	CanRestart(controllerType ControllerType) bool

	// GetRestartMethod returns the recommended restart method for the controller
	GetRestartMethod(controllerType ControllerType) RestartMethod
}

// RestartMethod represents different ways to restart pods
type RestartMethod string

const (
	RestartMethodRollout RestartMethod = "rollout"  // For Deployments and StatefulSets
	RestartMethodDelete  RestartMethod = "delete"   // For DaemonSets, ReplicaSets
	RestartMethodWarning RestartMethod = "warning"  // For standalone pods
)

// DefaultRestartStrategy implements RestartStrategy with standard Kubernetes patterns
type DefaultRestartStrategy struct{}

// NewDefaultRestartStrategy creates a new default restart strategy
func NewDefaultRestartStrategy() *DefaultRestartStrategy {
	return &DefaultRestartStrategy{}
}

// CanRestart implements RestartStrategy interface
func (s *DefaultRestartStrategy) CanRestart(controllerType ControllerType) bool {
	switch controllerType {
	case ControllerTypeDeployment, ControllerTypeStatefulSet,
		 ControllerTypeDaemonSet, ControllerTypeReplicaSet,
		 ControllerTypeStandalone:
		return true
	case ControllerTypeJob, ControllerTypeCronJob:
		return false // Jobs shouldn't be restarted
	default:
		return false
	}
}

// GetRestartMethod implements RestartStrategy interface
func (s *DefaultRestartStrategy) GetRestartMethod(controllerType ControllerType) RestartMethod {
	switch controllerType {
	case ControllerTypeDeployment, ControllerTypeStatefulSet:
		return RestartMethodRollout
	case ControllerTypeDaemonSet, ControllerTypeReplicaSet:
		return RestartMethodDelete
	case ControllerTypeStandalone:
		return RestartMethodWarning
	default:
		return RestartMethodWarning
	}
}