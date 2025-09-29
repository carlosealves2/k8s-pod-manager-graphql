package graph

import (
	"github.com/carlosf/k8s-pod-manager/graph/model"
	"github.com/carlosf/k8s-pod-manager/internal/services"
	appsv1 "k8s.io/api/apps/v1"
)

// DefaultTypeConverter implements TypeConverter interface
// Handles all conversions between service layer types and GraphQL types
// Following Single Responsibility Principle - only handles type conversions
type DefaultTypeConverter struct{}

// NewDefaultTypeConverter creates a new instance of DefaultTypeConverter
func NewDefaultTypeConverter() *DefaultTypeConverter {
	return &DefaultTypeConverter{}
}

// ConvertPodInfo converts service layer PodInfo to GraphQL PodInfo
func (c *DefaultTypeConverter) ConvertPodInfo(servicePod services.PodInfo) *model.PodInfo {
	return &model.PodInfo{
		Name:           servicePod.Name,
		Namespace:      servicePod.Namespace,
		Phase:          string(servicePod.Phase),
		PodIP:          c.stringPtr(servicePod.PodIP),
		HostIP:         c.stringPtr(servicePod.HostIP),
		NodeName:       c.stringPtr(servicePod.NodeName),
		Age:            servicePod.Age,
		Ready:          servicePod.Ready,
		Restarts:       int(servicePod.Restarts),
		Owners:         servicePod.Owners,
		ControllerType: servicePod.ControllerType,
		Containers:     c.convertContainerInfoList(servicePod.Containers),
		Labels:         c.MapToKeyValuePairs(servicePod.Labels),
	}
}

// ConvertNamespaceInfo converts service layer NamespaceInfo to GraphQL NamespaceInfo
func (c *DefaultTypeConverter) ConvertNamespaceInfo(serviceNS services.NamespaceInfo) *model.NamespaceInfo {
	return &model.NamespaceInfo{
		Name:        serviceNS.Name,
		Status:      serviceNS.Status,
		Age:         serviceNS.Age,
		Labels:      c.MapToKeyValuePairs(serviceNS.Labels),
		Annotations: c.MapToKeyValuePairs(serviceNS.Annotations),
	}
}

// ConvertDeploymentInfo converts Kubernetes Deployment to GraphQL DeploymentInfo
func (c *DefaultTypeConverter) ConvertDeploymentInfo(deployment appsv1.Deployment) *model.DeploymentInfo {
	replicas := int32(1)
	if deployment.Spec.Replicas != nil {
		replicas = *deployment.Spec.Replicas
	}

	return &model.DeploymentInfo{
		Name:              deployment.Name,
		Namespace:         deployment.Namespace,
		Replicas:          int(replicas),
		UpdatedReplicas:   int(deployment.Status.UpdatedReplicas),
		ReadyReplicas:     int(deployment.Status.ReadyReplicas),
		AvailableReplicas: int(deployment.Status.AvailableReplicas),
		Labels:            c.MapToKeyValuePairs(deployment.Labels),
		CreatedAt:         deployment.CreationTimestamp.Time,
	}
}

// ConvertStatefulSetInfo converts Kubernetes StatefulSet to GraphQL StatefulSetInfo
func (c *DefaultTypeConverter) ConvertStatefulSetInfo(statefulSet appsv1.StatefulSet) *model.StatefulSetInfo {
	replicas := int32(1)
	if statefulSet.Spec.Replicas != nil {
		replicas = *statefulSet.Spec.Replicas
	}

	return &model.StatefulSetInfo{
		Name:            statefulSet.Name,
		Namespace:       statefulSet.Namespace,
		Replicas:        int(replicas),
		ReadyReplicas:   int(statefulSet.Status.ReadyReplicas),
		CurrentReplicas: int(statefulSet.Status.CurrentReplicas),
		UpdatedReplicas: int(statefulSet.Status.UpdatedReplicas),
		Labels:          c.MapToKeyValuePairs(statefulSet.Labels),
		CreatedAt:       statefulSet.CreationTimestamp.Time,
	}
}

// ConvertPodWatchEvent converts service layer PodWatchEvent to GraphQL PodWatchEvent
func (c *DefaultTypeConverter) ConvertPodWatchEvent(serviceEvent services.PodWatchEvent) *model.PodWatchEvent {
	graphqlEvent := &model.PodWatchEvent{
		Type: serviceEvent.Type,
		Pod:  c.ConvertPodInfo(serviceEvent.Pod),
	}

	if serviceEvent.Reason != "" {
		graphqlEvent.Reason = &serviceEvent.Reason
	}

	return graphqlEvent
}

// MapToKeyValuePairs converts a map[string]string to GraphQL KeyValue pairs
func (c *DefaultTypeConverter) MapToKeyValuePairs(m map[string]string) []*model.KeyValue {
	if m == nil {
		return []*model.KeyValue{}
	}

	pairs := make([]*model.KeyValue, 0, len(m))
	for k, v := range m {
		pairs = append(pairs, &model.KeyValue{
			Key:   k,
			Value: v,
		})
	}
	return pairs
}

// convertContainerInfoList converts a slice of service layer ContainerInfo to GraphQL ContainerInfo
func (c *DefaultTypeConverter) convertContainerInfoList(containers []services.ContainerInfo) []*model.ContainerInfo {
	if len(containers) == 0 {
		return []*model.ContainerInfo{}
	}

	graphqlContainers := make([]*model.ContainerInfo, len(containers))
	for i, container := range containers {
		graphqlContainers[i] = &model.ContainerInfo{
			Name:         container.Name,
			Image:        container.Image,
			Ready:        container.Ready,
			RestartCount: int(container.RestartCount),
			State:        container.State,
		}
	}
	return graphqlContainers
}

// stringPtr returns a pointer to string if the string is not empty, otherwise nil
func (c *DefaultTypeConverter) stringPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// CachingTypeConverter wraps a TypeConverter with basic caching (Decorator Pattern)
// Useful for expensive conversions that are frequently repeated
type CachingTypeConverter struct {
	wrapped TypeConverter
	cache   map[string]interface{} // Simple in-memory cache
}

// NewCachingTypeConverter creates a new caching type converter
func NewCachingTypeConverter(wrapped TypeConverter) *CachingTypeConverter {
	return &CachingTypeConverter{
		wrapped: wrapped,
		cache:   make(map[string]interface{}),
	}
}

// ConvertPodInfo with caching (example implementation)
func (c *CachingTypeConverter) ConvertPodInfo(servicePod services.PodInfo) *model.PodInfo {
	// For simplicity, we'll just delegate to wrapped for now
	// In a real implementation, you'd implement caching logic based on pod UID + resourceVersion
	return c.wrapped.ConvertPodInfo(servicePod)
}

// ConvertNamespaceInfo delegates to wrapped converter
func (c *CachingTypeConverter) ConvertNamespaceInfo(serviceNS services.NamespaceInfo) *model.NamespaceInfo {
	return c.wrapped.ConvertNamespaceInfo(serviceNS)
}

// ConvertDeploymentInfo delegates to wrapped converter
func (c *CachingTypeConverter) ConvertDeploymentInfo(deployment appsv1.Deployment) *model.DeploymentInfo {
	return c.wrapped.ConvertDeploymentInfo(deployment)
}

// ConvertStatefulSetInfo delegates to wrapped converter
func (c *CachingTypeConverter) ConvertStatefulSetInfo(statefulSet appsv1.StatefulSet) *model.StatefulSetInfo {
	return c.wrapped.ConvertStatefulSetInfo(statefulSet)
}

// ConvertPodWatchEvent delegates to wrapped converter
func (c *CachingTypeConverter) ConvertPodWatchEvent(serviceEvent services.PodWatchEvent) *model.PodWatchEvent {
	return c.wrapped.ConvertPodWatchEvent(serviceEvent)
}

// MapToKeyValuePairs delegates to wrapped converter
func (c *CachingTypeConverter) MapToKeyValuePairs(m map[string]string) []*model.KeyValue {
	return c.wrapped.MapToKeyValuePairs(m)
}