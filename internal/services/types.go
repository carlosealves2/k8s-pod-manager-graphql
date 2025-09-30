package services

import (
	"time"

	corev1 "k8s.io/api/core/v1"
)

// PodInfo represents comprehensive pod information
type PodInfo struct {
	Name           string                 `json:"name"`
	Namespace      string                 `json:"namespace"`
	Phase          corev1.PodPhase        `json:"phase"`
	PodIP          string                 `json:"pod_ip,omitempty"`
	HostIP         string                 `json:"host_ip,omitempty"`
	NodeName       string                 `json:"node_name,omitempty"`
	Age            string                 `json:"age"`
	Ready          string                 `json:"ready"`
	Restarts       int32                  `json:"restarts"`
	Owners         []string               `json:"owners,omitempty"`
	ControllerType string                 `json:"controller_type"`
	Containers     []ContainerInfo        `json:"containers"`
	Labels         map[string]string      `json:"labels,omitempty"`
	Annotations    map[string]string      `json:"annotations,omitempty"`
	CreatedAt      time.Time              `json:"created_at"`
	Conditions     []PodConditionInfo     `json:"conditions,omitempty"`
	ResourceUsage  *ResourceUsageInfo     `json:"resource_usage,omitempty"`
}

// ContainerInfo represents container information within a pod
type ContainerInfo struct {
	Name         string              `json:"name"`
	Image        string              `json:"image"`
	Ready        bool                `json:"ready"`
	RestartCount int32               `json:"restart_count"`
	State        string              `json:"state"`
	Resources    *ContainerResources `json:"resources,omitempty"`
}

// ContainerResources represents container resource requests and limits
type ContainerResources struct {
	Requests ResourceList `json:"requests,omitempty"`
	Limits   ResourceList `json:"limits,omitempty"`
}

// ResourceList represents CPU and memory resources
type ResourceList struct {
	CPU    string `json:"cpu,omitempty"`
	Memory string `json:"memory,omitempty"`
}

// PodConditionInfo represents pod condition information
type PodConditionInfo struct {
	Type               string    `json:"type"`
	Status             string    `json:"status"`
	LastTransitionTime time.Time `json:"last_transition_time"`
	Reason             string    `json:"reason,omitempty"`
	Message            string    `json:"message,omitempty"`
}

// ResourceUsageInfo represents current resource usage (if metrics available)
type ResourceUsageInfo struct {
	CPU    string `json:"cpu,omitempty"`
	Memory string `json:"memory,omitempty"`
}

// NamespaceInfo represents namespace information
type NamespaceInfo struct {
	Name        string            `json:"name"`
	Status      string            `json:"status"`
	Age         string            `json:"age"`
	Labels      map[string]string `json:"labels,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
	CreatedAt   time.Time         `json:"created_at"`
}

// PodWatchEvent represents a pod watch event
type PodWatchEvent struct {
	Type      string    `json:"type"`
	Pod       PodInfo   `json:"pod"`
	Reason    string    `json:"reason,omitempty"`
	Message   string    `json:"message,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

// RestartResult represents the result of a pod restart operation
type RestartResult struct {
	Success        bool                   `json:"success"`
	Message        string                 `json:"message"`
	Method         RestartMethod          `json:"method"`
	ControllerType ControllerType         `json:"controller_type"`
	ControllerName string                 `json:"controller_name,omitempty"`
	Details        map[string]interface{} `json:"details,omitempty"`
	Warning        string                 `json:"warning,omitempty"`
}

// ScaleResult represents the result of a scaling operation
type ScaleResult struct {
	Success          bool   `json:"success"`
	Message          string `json:"message"`
	ResourceType     string `json:"resource_type"`
	ResourceName     string `json:"resource_name"`
	Namespace        string `json:"namespace"`
	PreviousReplicas int32  `json:"previous_replicas"`
	NewReplicas      int32  `json:"new_replicas"`
	Action           string `json:"action"`
}

// ListOptions represents options for listing resources
type ListOptions struct {
	Namespace    string            `json:"namespace,omitempty"`
	LabelFilter  map[string]string `json:"label_filter,omitempty"`
	FieldFilter  map[string]string `json:"field_filter,omitempty"`
	Limit        int64             `json:"limit,omitempty"`
	Continue     string            `json:"continue,omitempty"`
}

// WatchOptions represents options for watching resources
type WatchOptions struct {
	Namespace       string            `json:"namespace,omitempty"`
	LabelFilter     map[string]string `json:"label_filter,omitempty"`
	FieldFilter     map[string]string `json:"field_filter,omitempty"`
	ResourceVersion string            `json:"resource_version,omitempty"`
}

// LogStreamOptions represents options for streaming pod logs
type LogStreamOptions struct {
	Container    string `json:"container,omitempty"`     // Container name (required for multi-container pods)
	Follow       bool   `json:"follow"`                  // Follow log stream
	TailLines    *int64 `json:"tail_lines,omitempty"`    // Number of lines to tail
	SinceSeconds *int64 `json:"since_seconds,omitempty"` // Show logs since N seconds ago
	Timestamps   bool   `json:"timestamps"`              // Include timestamps in log lines
}

// PodLogLine represents a single line from pod logs
type PodLogLine struct {
	Timestamp time.Time `json:"timestamp"`
	Line      string    `json:"line"`
	Container string    `json:"container,omitempty"`
}