package services

import (
	"fmt"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

// podConverterImpl implements PodConverter interface
type podConverterImpl struct {
	enableResourceInfo bool
}

// NewPodConverter creates a new pod converter
func NewPodConverter(enableResourceInfo bool) PodConverter {
	return &podConverterImpl{
		enableResourceInfo: enableResourceInfo,
	}
}

// ConvertPod implements PodConverter
func (c *podConverterImpl) ConvertPod(pod corev1.Pod, controllerInfo ControllerInfo) PodInfo {
	now := time.Now()

	// Extract owner references
	owners := make([]string, 0, len(pod.OwnerReferences))
	for _, owner := range pod.OwnerReferences {
		owners = append(owners, fmt.Sprintf("%s/%s", owner.Kind, owner.Name))
	}

	// Process containers
	containers := c.ExtractContainerInfo(pod)

	// Calculate totals
	totalRestarts := int32(0)
	readyContainers := 0
	totalContainers := len(pod.Spec.Containers)

	for _, container := range containers {
		totalRestarts += container.RestartCount
		if container.Ready {
			readyContainers++
		}
	}

	// Calculate age
	age := now.Sub(pod.CreationTimestamp.Time)
	ageStr := c.formatDuration(age)
	ready := fmt.Sprintf("%d/%d", readyContainers, totalContainers)

	// Extract conditions
	conditions := c.extractPodConditions(pod)

	// Extract resource usage if enabled
	var resourceUsage *ResourceUsageInfo
	if c.enableResourceInfo {
		resourceUsage = c.CalculateResourceUsage(pod)
	}

	return PodInfo{
		Name:           pod.Name,
		Namespace:      pod.Namespace,
		Phase:          pod.Status.Phase,
		PodIP:          pod.Status.PodIP,
		HostIP:         pod.Status.HostIP,
		NodeName:       pod.Spec.NodeName,
		Age:            ageStr,
		Ready:          ready,
		Restarts:       totalRestarts,
		Owners:         owners,
		ControllerType: string(controllerInfo.Type),
		Containers:     containers,
		Labels:         pod.Labels,
		Annotations:    pod.Annotations,
		CreatedAt:      pod.CreationTimestamp.Time,
		Conditions:     conditions,
		ResourceUsage:  resourceUsage,
	}
}

// ConvertPods implements PodConverter
func (c *podConverterImpl) ConvertPods(pods []corev1.Pod) []PodInfo {
	podInfos := make([]PodInfo, 0, len(pods))
	for _, pod := range pods {
		// For bulk conversion, we'll use a basic controller detection
		// In practice, you might want to batch the controller detection for performance
		controllerInfo := c.detectBasicControllerInfo(pod)
		podInfo := c.ConvertPod(pod, controllerInfo)
		podInfos = append(podInfos, podInfo)
	}
	return podInfos
}

// ExtractContainerInfo implements PodConverter
func (c *podConverterImpl) ExtractContainerInfo(pod corev1.Pod) []ContainerInfo {
	containers := make([]ContainerInfo, 0, len(pod.Spec.Containers))

	for i, container := range pod.Spec.Containers {
		containerInfo := ContainerInfo{
			Name:  container.Name,
			Image: container.Image,
		}

		// Extract resource information
		if c.enableResourceInfo {
			containerInfo.Resources = c.extractContainerResources(container)
		}

		// Match with status if available
		if i < len(pod.Status.ContainerStatuses) {
			status := pod.Status.ContainerStatuses[i]
			containerInfo.Ready = status.Ready
			containerInfo.RestartCount = status.RestartCount
			containerInfo.State = c.getContainerState(status)
		} else {
			// Fallback for containers without status
			containerInfo.State = "Unknown"
		}

		containers = append(containers, containerInfo)
	}

	return containers
}

// CalculateResourceUsage implements PodConverter
func (c *podConverterImpl) CalculateResourceUsage(pod corev1.Pod) *ResourceUsageInfo {
	// This would integrate with metrics APIs in a real implementation
	// For now, we calculate theoretical usage from requests

	totalCPURequests := resource.Quantity{}
	totalMemoryRequests := resource.Quantity{}

	for _, container := range pod.Spec.Containers {
		if container.Resources.Requests != nil {
			if cpu, exists := container.Resources.Requests[corev1.ResourceCPU]; exists {
				totalCPURequests.Add(cpu)
			}
			if memory, exists := container.Resources.Requests[corev1.ResourceMemory]; exists {
				totalMemoryRequests.Add(memory)
			}
		}
	}

	resourceUsage := &ResourceUsageInfo{}

	if !totalCPURequests.IsZero() {
		resourceUsage.CPU = totalCPURequests.String()
	}

	if !totalMemoryRequests.IsZero() {
		resourceUsage.Memory = totalMemoryRequests.String()
	}

	// Return nil if no resource information
	if resourceUsage.CPU == "" && resourceUsage.Memory == "" {
		return nil
	}

	return resourceUsage
}

// Helper methods

func (c *podConverterImpl) detectBasicControllerInfo(pod corev1.Pod) ControllerInfo {
	// Simple controller detection without API calls for bulk operations
	for _, owner := range pod.OwnerReferences {
		if owner.Controller != nil && *owner.Controller {
			return ControllerInfo{
				Type: ControllerType(owner.Kind),
				Name: owner.Name,
				Kind: owner.Kind,
			}
		}
	}

	// Check for common labels
	if pod.Labels != nil {
		if jobName, ok := pod.Labels["job-name"]; ok {
			return ControllerInfo{
				Type: ControllerTypeJob,
				Name: jobName,
				Kind: "Job",
			}
		}
		if jobName, ok := pod.Labels["batch.kubernetes.io/job-name"]; ok {
			return ControllerInfo{
				Type: ControllerTypeJob,
				Name: jobName,
				Kind: "Job",
			}
		}
	}

	return ControllerInfo{
		Type: ControllerTypeStandalone,
		Name: "",
		Kind: "",
	}
}

func (c *podConverterImpl) getContainerState(status corev1.ContainerStatus) string {
	if status.State.Running != nil {
		return "Running"
	} else if status.State.Waiting != nil {
		reason := status.State.Waiting.Reason
		if reason == "" {
			reason = "Unknown"
		}
		return fmt.Sprintf("Waiting: %s", reason)
	} else if status.State.Terminated != nil {
		reason := status.State.Terminated.Reason
		if reason == "" {
			reason = "Unknown"
		}
		return fmt.Sprintf("Terminated: %s", reason)
	}
	return "Unknown"
}

func (c *podConverterImpl) extractContainerResources(container corev1.Container) *ContainerResources {
	if container.Resources.Requests == nil && container.Resources.Limits == nil {
		return nil
	}

	resources := &ContainerResources{}

	// Extract requests
	if container.Resources.Requests != nil {
		requests := ResourceList{}
		if cpu, exists := container.Resources.Requests[corev1.ResourceCPU]; exists {
			requests.CPU = cpu.String()
		}
		if memory, exists := container.Resources.Requests[corev1.ResourceMemory]; exists {
			requests.Memory = memory.String()
		}
		resources.Requests = requests
	}

	// Extract limits
	if container.Resources.Limits != nil {
		limits := ResourceList{}
		if cpu, exists := container.Resources.Limits[corev1.ResourceCPU]; exists {
			limits.CPU = cpu.String()
		}
		if memory, exists := container.Resources.Limits[corev1.ResourceMemory]; exists {
			limits.Memory = memory.String()
		}
		resources.Limits = limits
	}

	return resources
}

func (c *podConverterImpl) extractPodConditions(pod corev1.Pod) []PodConditionInfo {
	if len(pod.Status.Conditions) == 0 {
		return nil
	}

	conditions := make([]PodConditionInfo, 0, len(pod.Status.Conditions))
	for _, condition := range pod.Status.Conditions {
		conditions = append(conditions, PodConditionInfo{
			Type:               string(condition.Type),
			Status:             string(condition.Status),
			LastTransitionTime: condition.LastTransitionTime.Time,
			Reason:             condition.Reason,
			Message:            condition.Message,
		})
	}

	return conditions
}

func (c *podConverterImpl) formatDuration(d time.Duration) string {
	if d < 0 {
		return "Unknown"
	}

	days := int(d.Hours() / 24)
	hours := int(d.Hours()) % 24
	minutes := int(d.Minutes()) % 60
	seconds := int(d.Seconds()) % 60

	var parts []string

	if days > 0 {
		parts = append(parts, fmt.Sprintf("%dd", days))
	}
	if hours > 0 {
		parts = append(parts, fmt.Sprintf("%dh", hours))
	}
	if minutes > 0 {
		parts = append(parts, fmt.Sprintf("%dm", minutes))
	}
	if len(parts) == 0 || (len(parts) == 1 && days == 0 && hours == 0) {
		parts = append(parts, fmt.Sprintf("%ds", seconds))
	}

	// Limit to first two most significant units
	if len(parts) > 2 {
		parts = parts[:2]
	}

	return strings.Join(parts, "")
}