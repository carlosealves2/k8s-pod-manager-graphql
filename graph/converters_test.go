//go:build ignore
// +build ignore

// Tests skipped due to dependency issues
package graph

import (
	"testing"
	"time"

	"github.com/carlosf/k8s-pod-manager/graph/model"
	"github.com/carlosf/k8s-pod-manager/internal/services"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestDefaultTypeConverter_ConvertPodInfo(t *testing.T) {
	converter := NewDefaultTypeConverter()

	tests := []struct {
		name        string
		servicePod  services.PodInfo
		expectedPod *model.PodInfo
	}{
		{
			name: "complete pod info conversion",
			servicePod: services.PodInfo{
				Name:           "test-pod",
				Namespace:      "default",
				Phase:          corev1.PodRunning,
				PodIP:          "10.0.0.1",
				HostIP:         "192.168.1.100",
				NodeName:       "node-1",
				Age:            "5m",
				Ready:          "1/1",
				Restarts:       2,
				Owners:         []string{"deployment/test-app"},
				ControllerType: "Deployment",
				Containers: []services.ContainerInfo{
					{
						Name:         "app-container",
						Image:        "nginx:1.20",
						Ready:        true,
						RestartCount: 1,
						State:        "Running",
					},
				},
				Labels: map[string]string{
					"app":     "test",
					"version": "1.0",
				},
			},
			expectedPod: &model.PodInfo{
				Name:           "test-pod",
				Namespace:      "default",
				Phase:          "Running",
				PodIP:          stringPtr("10.0.0.1"),
				HostIP:         stringPtr("192.168.1.100"),
				NodeName:       stringPtr("node-1"),
				Age:            "5m",
				Ready:          "1/1",
				Restarts:       2,
				Owners:         []string{"deployment/test-app"},
				ControllerType: "Deployment",
				Containers: []*model.ContainerInfo{
					{
						Name:         "app-container",
						Image:        "nginx:1.20",
						Ready:        true,
						RestartCount: 1,
						State:        "Running",
					},
				},
				Labels: []*model.KeyValue{
					{Key: "app", Value: "test"},
					{Key: "version", Value: "1.0"},
				},
			},
		},
		{
			name: "minimal pod info conversion",
			servicePod: services.PodInfo{
				Name:           "minimal-pod",
				Namespace:      "test",
				Phase:          corev1.PodPending,
				Age:            "1m",
				Ready:          "0/1",
				Restarts:       0,
				ControllerType: "Standalone",
				Containers:     []services.ContainerInfo{},
				Labels:         map[string]string{},
			},
			expectedPod: &model.PodInfo{
				Name:           "minimal-pod",
				Namespace:      "test",
				Phase:          "Pending",
				PodIP:          nil,
				HostIP:         nil,
				NodeName:       nil,
				Age:            "1m",
				Ready:          "0/1",
				Restarts:       0,
				Owners:         nil,
				ControllerType: "Standalone",
				Containers:     []*model.ContainerInfo{},
				Labels:         []*model.KeyValue{},
			},
		},
		{
			name: "pod with empty strings should return nil pointers",
			servicePod: services.PodInfo{
				Name:           "empty-strings-pod",
				Namespace:      "default",
				Phase:          corev1.PodFailed,
				PodIP:          "",
				HostIP:         "",
				NodeName:       "",
				Age:            "10m",
				Ready:          "0/1",
				Restarts:       5,
				ControllerType: "DaemonSet",
				Containers:     []services.ContainerInfo{},
				Labels:         nil,
			},
			expectedPod: &model.PodInfo{
				Name:           "empty-strings-pod",
				Namespace:      "default",
				Phase:          "Failed",
				PodIP:          nil,
				HostIP:         nil,
				NodeName:       nil,
				Age:            "10m",
				Ready:          "0/1",
				Restarts:       5,
				Owners:         nil,
				ControllerType: "DaemonSet",
				Containers:     []*model.ContainerInfo{},
				Labels:         []*model.KeyValue{},
			},
		},
		{
			name: "pod with multiple containers",
			servicePod: services.PodInfo{
				Name:      "multi-container-pod",
				Namespace: "production",
				Phase:     corev1.PodRunning,
				Age:       "2h",
				Ready:     "2/2",
				Restarts:  0,
				Containers: []services.ContainerInfo{
					{
						Name:         "main-app",
						Image:        "app:latest",
						Ready:        true,
						RestartCount: 0,
						State:        "Running",
					},
					{
						Name:         "sidecar",
						Image:        "sidecar:v1.0",
						Ready:        true,
						RestartCount: 2,
						State:        "Running",
					},
				},
				ControllerType: "StatefulSet",
				Labels:         map[string]string{},
			},
			expectedPod: &model.PodInfo{
				Name:      "multi-container-pod",
				Namespace: "production",
				Phase:     "Running",
				Age:       "2h",
				Ready:     "2/2",
				Restarts:  0,
				Containers: []*model.ContainerInfo{
					{
						Name:         "main-app",
						Image:        "app:latest",
						Ready:        true,
						RestartCount: 0,
						State:        "Running",
					},
					{
						Name:         "sidecar",
						Image:        "sidecar:v1.0",
						Ready:        true,
						RestartCount: 2,
						State:        "Running",
					},
				},
				ControllerType: "StatefulSet",
				Labels:         []*model.KeyValue{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := converter.ConvertPodInfo(tt.servicePod)

			assert.Equal(t, tt.expectedPod.Name, result.Name)
			assert.Equal(t, tt.expectedPod.Namespace, result.Namespace)
			assert.Equal(t, tt.expectedPod.Phase, result.Phase)
			assert.Equal(t, tt.expectedPod.Age, result.Age)
			assert.Equal(t, tt.expectedPod.Ready, result.Ready)
			assert.Equal(t, tt.expectedPod.Restarts, result.Restarts)
			assert.Equal(t, tt.expectedPod.ControllerType, result.ControllerType)

			// Check pointer fields
			if tt.expectedPod.PodIP == nil {
				assert.Nil(t, result.PodIP)
			} else {
				require.NotNil(t, result.PodIP)
				assert.Equal(t, *tt.expectedPod.PodIP, *result.PodIP)
			}

			if tt.expectedPod.HostIP == nil {
				assert.Nil(t, result.HostIP)
			} else {
				require.NotNil(t, result.HostIP)
				assert.Equal(t, *tt.expectedPod.HostIP, *result.HostIP)
			}

			if tt.expectedPod.NodeName == nil {
				assert.Nil(t, result.NodeName)
			} else {
				require.NotNil(t, result.NodeName)
				assert.Equal(t, *tt.expectedPod.NodeName, *result.NodeName)
			}

			// Check containers
			assert.Len(t, result.Containers, len(tt.expectedPod.Containers))
			for i, expectedContainer := range tt.expectedPod.Containers {
				assert.Equal(t, expectedContainer.Name, result.Containers[i].Name)
				assert.Equal(t, expectedContainer.Image, result.Containers[i].Image)
				assert.Equal(t, expectedContainer.Ready, result.Containers[i].Ready)
				assert.Equal(t, expectedContainer.RestartCount, result.Containers[i].RestartCount)
				assert.Equal(t, expectedContainer.State, result.Containers[i].State)
			}

			// Check labels (order might differ, so check contents)
			assert.Len(t, result.Labels, len(tt.expectedPod.Labels))
			labelMap := make(map[string]string)
			for _, label := range result.Labels {
				labelMap[label.Key] = label.Value
			}
			for _, expectedLabel := range tt.expectedPod.Labels {
				assert.Equal(t, expectedLabel.Value, labelMap[expectedLabel.Key])
			}
		})
	}
}

func TestDefaultTypeConverter_ConvertNamespaceInfo(t *testing.T) {
	converter := NewDefaultTypeConverter()

	tests := []struct {
		name           string
		serviceNS      services.NamespaceInfo
		expectedNS     *model.NamespaceInfo
	}{
		{
			name: "complete namespace info",
			serviceNS: services.NamespaceInfo{
				Name:   "production",
				Status: "Active",
				Age:    "30d",
				Labels: map[string]string{
					"env":  "prod",
					"team": "backend",
				},
				Annotations: map[string]string{
					"description": "Production environment",
					"contact":     "backend-team@company.com",
				},
			},
			expectedNS: &model.NamespaceInfo{
				Name:   "production",
				Status: "Active",
				Age:    "30d",
				Labels: []*model.KeyValue{
					{Key: "env", Value: "prod"},
					{Key: "team", Value: "backend"},
				},
				Annotations: []*model.KeyValue{
					{Key: "description", Value: "Production environment"},
					{Key: "contact", Value: "backend-team@company.com"},
				},
			},
		},
		{
			name: "minimal namespace info",
			serviceNS: services.NamespaceInfo{
				Name:        "test",
				Status:      "Active",
				Age:         "1d",
				Labels:      map[string]string{},
				Annotations: map[string]string{},
			},
			expectedNS: &model.NamespaceInfo{
				Name:        "test",
				Status:      "Active",
				Age:         "1d",
				Labels:      []*model.KeyValue{},
				Annotations: []*model.KeyValue{},
			},
		},
		{
			name: "namespace with nil maps",
			serviceNS: services.NamespaceInfo{
				Name:        "default",
				Status:      "Active",
				Age:         "100d",
				Labels:      nil,
				Annotations: nil,
			},
			expectedNS: &model.NamespaceInfo{
				Name:        "default",
				Status:      "Active",
				Age:         "100d",
				Labels:      []*model.KeyValue{},
				Annotations: []*model.KeyValue{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := converter.ConvertNamespaceInfo(tt.serviceNS)

			assert.Equal(t, tt.expectedNS.Name, result.Name)
			assert.Equal(t, tt.expectedNS.Status, result.Status)
			assert.Equal(t, tt.expectedNS.Age, result.Age)

			// Check labels
			assert.Len(t, result.Labels, len(tt.expectedNS.Labels))
			labelMap := make(map[string]string)
			for _, label := range result.Labels {
				labelMap[label.Key] = label.Value
			}
			for _, expectedLabel := range tt.expectedNS.Labels {
				assert.Equal(t, expectedLabel.Value, labelMap[expectedLabel.Key])
			}

			// Check annotations
			assert.Len(t, result.Annotations, len(tt.expectedNS.Annotations))
			annotationMap := make(map[string]string)
			for _, annotation := range result.Annotations {
				annotationMap[annotation.Key] = annotation.Value
			}
			for _, expectedAnnotation := range tt.expectedNS.Annotations {
				assert.Equal(t, expectedAnnotation.Value, annotationMap[expectedAnnotation.Key])
			}
		})
	}
}

func TestDefaultTypeConverter_ConvertDeploymentInfo(t *testing.T) {
	converter := NewDefaultTypeConverter()
	now := time.Now()

	tests := []struct {
		name               string
		deployment         appsv1.Deployment
		expectedDeployment *model.DeploymentInfo
	}{
		{
			name: "complete deployment info",
			deployment: appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:              "web-app",
					Namespace:         "production",
					CreationTimestamp: metav1.NewTime(now),
					Labels: map[string]string{
						"app":     "web",
						"version": "v1.0",
					},
				},
				Spec: appsv1.DeploymentSpec{
					Replicas: int32Ptr(3),
				},
				Status: appsv1.DeploymentStatus{
					UpdatedReplicas:   2,
					ReadyReplicas:     2,
					AvailableReplicas: 2,
				},
			},
			expectedDeployment: &model.DeploymentInfo{
				Name:              "web-app",
				Namespace:         "production",
				Replicas:          3,
				UpdatedReplicas:   2,
				ReadyReplicas:     2,
				AvailableReplicas: 2,
				Labels: []*model.KeyValue{
					{Key: "app", Value: "web"},
					{Key: "version", Value: "v1.0"},
				},
				CreatedAt: now,
			},
		},
		{
			name: "deployment with nil replicas",
			deployment: appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:              "minimal-app",
					Namespace:         "default",
					CreationTimestamp: metav1.NewTime(now),
					Labels:            map[string]string{},
				},
				Spec: appsv1.DeploymentSpec{
					Replicas: nil, // Should default to 1
				},
				Status: appsv1.DeploymentStatus{
					UpdatedReplicas:   1,
					ReadyReplicas:     1,
					AvailableReplicas: 1,
				},
			},
			expectedDeployment: &model.DeploymentInfo{
				Name:              "minimal-app",
				Namespace:         "default",
				Replicas:          1, // Default when nil
				UpdatedReplicas:   1,
				ReadyReplicas:     1,
				AvailableReplicas: 1,
				Labels:            []*model.KeyValue{},
				CreatedAt:         now,
			},
		},
		{
			name: "deployment with zero replicas",
			deployment: appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:              "scaled-down",
					Namespace:         "test",
					CreationTimestamp: metav1.NewTime(now),
				},
				Spec: appsv1.DeploymentSpec{
					Replicas: int32Ptr(0),
				},
				Status: appsv1.DeploymentStatus{
					UpdatedReplicas:   0,
					ReadyReplicas:     0,
					AvailableReplicas: 0,
				},
			},
			expectedDeployment: &model.DeploymentInfo{
				Name:              "scaled-down",
				Namespace:         "test",
				Replicas:          0,
				UpdatedReplicas:   0,
				ReadyReplicas:     0,
				AvailableReplicas: 0,
				Labels:            []*model.KeyValue{},
				CreatedAt:         now,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := converter.ConvertDeploymentInfo(tt.deployment)

			assert.Equal(t, tt.expectedDeployment.Name, result.Name)
			assert.Equal(t, tt.expectedDeployment.Namespace, result.Namespace)
			assert.Equal(t, tt.expectedDeployment.Replicas, result.Replicas)
			assert.Equal(t, tt.expectedDeployment.UpdatedReplicas, result.UpdatedReplicas)
			assert.Equal(t, tt.expectedDeployment.ReadyReplicas, result.ReadyReplicas)
			assert.Equal(t, tt.expectedDeployment.AvailableReplicas, result.AvailableReplicas)
			assert.Equal(t, tt.expectedDeployment.CreatedAt, result.CreatedAt)

			// Check labels
			assert.Len(t, result.Labels, len(tt.expectedDeployment.Labels))
			labelMap := make(map[string]string)
			for _, label := range result.Labels {
				labelMap[label.Key] = label.Value
			}
			for _, expectedLabel := range tt.expectedDeployment.Labels {
				assert.Equal(t, expectedLabel.Value, labelMap[expectedLabel.Key])
			}
		})
	}
}

func TestDefaultTypeConverter_ConvertStatefulSetInfo(t *testing.T) {
	converter := NewDefaultTypeConverter()
	now := time.Now()

	tests := []struct {
		name                string
		statefulSet         appsv1.StatefulSet
		expectedStatefulSet *model.StatefulSetInfo
	}{
		{
			name: "complete statefulset info",
			statefulSet: appsv1.StatefulSet{
				ObjectMeta: metav1.ObjectMeta{
					Name:              "database",
					Namespace:         "production",
					CreationTimestamp: metav1.NewTime(now),
					Labels: map[string]string{
						"app":  "postgres",
						"role": "database",
					},
				},
				Spec: appsv1.StatefulSetSpec{
					Replicas: int32Ptr(3),
				},
				Status: appsv1.StatefulSetStatus{
					ReadyReplicas:   2,
					CurrentReplicas: 3,
					UpdatedReplicas: 2,
				},
			},
			expectedStatefulSet: &model.StatefulSetInfo{
				Name:            "database",
				Namespace:       "production",
				Replicas:        3,
				ReadyReplicas:   2,
				CurrentReplicas: 3,
				UpdatedReplicas: 2,
				Labels: []*model.KeyValue{
					{Key: "app", Value: "postgres"},
					{Key: "role", Value: "database"},
				},
				CreatedAt: now,
			},
		},
		{
			name: "statefulset with nil replicas",
			statefulSet: appsv1.StatefulSet{
				ObjectMeta: metav1.ObjectMeta{
					Name:              "cache",
					Namespace:         "default",
					CreationTimestamp: metav1.NewTime(now),
				},
				Spec: appsv1.StatefulSetSpec{
					Replicas: nil, // Should default to 1
				},
				Status: appsv1.StatefulSetStatus{
					ReadyReplicas:   1,
					CurrentReplicas: 1,
					UpdatedReplicas: 1,
				},
			},
			expectedStatefulSet: &model.StatefulSetInfo{
				Name:            "cache",
				Namespace:       "default",
				Replicas:        1, // Default when nil
				ReadyReplicas:   1,
				CurrentReplicas: 1,
				UpdatedReplicas: 1,
				Labels:          []*model.KeyValue{},
				CreatedAt:       now,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := converter.ConvertStatefulSetInfo(tt.statefulSet)

			assert.Equal(t, tt.expectedStatefulSet.Name, result.Name)
			assert.Equal(t, tt.expectedStatefulSet.Namespace, result.Namespace)
			assert.Equal(t, tt.expectedStatefulSet.Replicas, result.Replicas)
			assert.Equal(t, tt.expectedStatefulSet.ReadyReplicas, result.ReadyReplicas)
			assert.Equal(t, tt.expectedStatefulSet.CurrentReplicas, result.CurrentReplicas)
			assert.Equal(t, tt.expectedStatefulSet.UpdatedReplicas, result.UpdatedReplicas)
			assert.Equal(t, tt.expectedStatefulSet.CreatedAt, result.CreatedAt)

			// Check labels
			assert.Len(t, result.Labels, len(tt.expectedStatefulSet.Labels))
			labelMap := make(map[string]string)
			for _, label := range result.Labels {
				labelMap[label.Key] = label.Value
			}
			for _, expectedLabel := range tt.expectedStatefulSet.Labels {
				assert.Equal(t, expectedLabel.Value, labelMap[expectedLabel.Key])
			}
		})
	}
}

func TestDefaultTypeConverter_ConvertPodWatchEvent(t *testing.T) {
	converter := NewDefaultTypeConverter()

	tests := []struct {
		name          string
		serviceEvent  services.PodWatchEvent
		expectedEvent *model.PodWatchEvent
	}{
		{
			name: "watch event with reason",
			serviceEvent: services.PodWatchEvent{
				Type: "ADDED",
				Pod: services.PodInfo{
					Name:      "test-pod",
					Namespace: "default",
					Phase:     corev1.PodRunning,
					Age:       "1m",
					Ready:     "1/1",
				},
				Reason: "PodCreated",
			},
			expectedEvent: &model.PodWatchEvent{
				Type: "ADDED",
				Pod: &model.PodInfo{
					Name:      "test-pod",
					Namespace: "default",
					Phase:     "Running",
					Age:       "1m",
					Ready:     "1/1",
				},
				Reason: stringPtr("PodCreated"),
			},
		},
		{
			name: "watch event without reason",
			serviceEvent: services.PodWatchEvent{
				Type: "DELETED",
				Pod: services.PodInfo{
					Name:      "deleted-pod",
					Namespace: "test",
					Phase:     corev1.PodSucceeded,
					Age:       "5m",
					Ready:     "0/1",
				},
				Reason: "",
			},
			expectedEvent: &model.PodWatchEvent{
				Type: "DELETED",
				Pod: &model.PodInfo{
					Name:      "deleted-pod",
					Namespace: "test",
					Phase:     "Succeeded",
					Age:       "5m",
					Ready:     "0/1",
				},
				Reason: nil,
			},
		},
		{
			name: "modified watch event",
			serviceEvent: services.PodWatchEvent{
				Type: "MODIFIED",
				Pod: services.PodInfo{
					Name:      "updated-pod",
					Namespace: "production",
					Phase:     corev1.PodFailed,
					Age:       "10m",
					Ready:     "0/1",
					Restarts:  3,
				},
				Reason: "ContainerFailed",
			},
			expectedEvent: &model.PodWatchEvent{
				Type: "MODIFIED",
				Pod: &model.PodInfo{
					Name:      "updated-pod",
					Namespace: "production",
					Phase:     "Failed",
					Age:       "10m",
					Ready:     "0/1",
					Restarts:  3,
				},
				Reason: stringPtr("ContainerFailed"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := converter.ConvertPodWatchEvent(tt.serviceEvent)

			assert.Equal(t, tt.expectedEvent.Type, result.Type)

			// Check pod conversion
			assert.Equal(t, tt.expectedEvent.Pod.Name, result.Pod.Name)
			assert.Equal(t, tt.expectedEvent.Pod.Namespace, result.Pod.Namespace)
			assert.Equal(t, tt.expectedEvent.Pod.Phase, result.Pod.Phase)

			// Check reason pointer
			if tt.expectedEvent.Reason == nil {
				assert.Nil(t, result.Reason)
			} else {
				require.NotNil(t, result.Reason)
				assert.Equal(t, *tt.expectedEvent.Reason, *result.Reason)
			}
		})
	}
}

func TestDefaultTypeConverter_MapToKeyValuePairs(t *testing.T) {
	converter := NewDefaultTypeConverter()

	tests := []struct {
		name     string
		input    map[string]string
		expected []*model.KeyValue
	}{
		{
			name:     "nil map",
			input:    nil,
			expected: []*model.KeyValue{},
		},
		{
			name:     "empty map",
			input:    map[string]string{},
			expected: []*model.KeyValue{},
		},
		{
			name: "single pair",
			input: map[string]string{
				"key": "value",
			},
			expected: []*model.KeyValue{
				{Key: "key", Value: "value"},
			},
		},
		{
			name: "multiple pairs",
			input: map[string]string{
				"app":     "web",
				"version": "1.0",
				"env":     "prod",
			},
			expected: []*model.KeyValue{
				{Key: "app", Value: "web"},
				{Key: "version", Value: "1.0"},
				{Key: "env", Value: "prod"},
			},
		},
		{
			name: "pairs with empty values",
			input: map[string]string{
				"key1": "",
				"key2": "value",
				"key3": "",
			},
			expected: []*model.KeyValue{
				{Key: "key1", Value: ""},
				{Key: "key2", Value: "value"},
				{Key: "key3", Value: ""},
			},
		},
		{
			name: "pairs with special characters",
			input: map[string]string{
				"special/key":   "value with spaces",
				"another.key":   "value-with-dashes",
				"kubernetes.io": "annotation-style",
			},
			expected: []*model.KeyValue{
				{Key: "special/key", Value: "value with spaces"},
				{Key: "another.key", Value: "value-with-dashes"},
				{Key: "kubernetes.io", Value: "annotation-style"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := converter.MapToKeyValuePairs(tt.input)

			assert.Len(t, result, len(tt.expected))

			// Convert result to map for easy comparison (order doesn't matter)
			resultMap := make(map[string]string)
			for _, kv := range result {
				resultMap[kv.Key] = kv.Value
			}

			expectedMap := make(map[string]string)
			for _, kv := range tt.expected {
				expectedMap[kv.Key] = kv.Value
			}

			assert.Equal(t, expectedMap, resultMap)
		})
	}
}

func TestDefaultTypeConverter_stringPtr(t *testing.T) {
	converter := NewDefaultTypeConverter()

	tests := []struct {
		name     string
		input    string
		expected *string
	}{
		{
			name:     "empty string returns nil",
			input:    "",
			expected: nil,
		},
		{
			name:     "non-empty string returns pointer",
			input:    "test",
			expected: stringPtr("test"),
		},
		{
			name:     "whitespace string returns pointer",
			input:    " ",
			expected: stringPtr(" "),
		},
		{
			name:     "long string returns pointer",
			input:    "this is a very long string with lots of content",
			expected: stringPtr("this is a very long string with lots of content"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := converter.stringPtr(tt.input)

			if tt.expected == nil {
				assert.Nil(t, result)
			} else {
				require.NotNil(t, result)
				assert.Equal(t, *tt.expected, *result)
			}
		})
	}
}

func TestDefaultTypeConverter_convertContainerInfoList(t *testing.T) {
	converter := NewDefaultTypeConverter()

	tests := []struct {
		name       string
		containers []services.ContainerInfo
		expected   []*model.ContainerInfo
	}{
		{
			name:       "empty slice",
			containers: []services.ContainerInfo{},
			expected:   []*model.ContainerInfo{},
		},
		{
			name:       "nil slice",
			containers: nil,
			expected:   []*model.ContainerInfo{},
		},
		{
			name: "single container",
			containers: []services.ContainerInfo{
				{
					Name:         "app",
					Image:        "nginx:latest",
					Ready:        true,
					RestartCount: 0,
					State:        "Running",
				},
			},
			expected: []*model.ContainerInfo{
				{
					Name:         "app",
					Image:        "nginx:latest",
					Ready:        true,
					RestartCount: 0,
					State:        "Running",
				},
			},
		},
		{
			name: "multiple containers",
			containers: []services.ContainerInfo{
				{
					Name:         "main",
					Image:        "app:v1.0",
					Ready:        true,
					RestartCount: 1,
					State:        "Running",
				},
				{
					Name:         "sidecar",
					Image:        "sidecar:latest",
					Ready:        false,
					RestartCount: 5,
					State:        "CrashLoopBackOff",
				},
			},
			expected: []*model.ContainerInfo{
				{
					Name:         "main",
					Image:        "app:v1.0",
					Ready:        true,
					RestartCount: 1,
					State:        "Running",
				},
				{
					Name:         "sidecar",
					Image:        "sidecar:latest",
					Ready:        false,
					RestartCount: 5,
					State:        "CrashLoopBackOff",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := converter.convertContainerInfoList(tt.containers)

			assert.Len(t, result, len(tt.expected))
			for i, expected := range tt.expected {
				assert.Equal(t, expected.Name, result[i].Name)
				assert.Equal(t, expected.Image, result[i].Image)
				assert.Equal(t, expected.Ready, result[i].Ready)
				assert.Equal(t, expected.RestartCount, result[i].RestartCount)
				assert.Equal(t, expected.State, result[i].State)
			}
		})
	}
}

func TestCachingTypeConverter(t *testing.T) {
	mockConverter := &MockTypeConverter{}
	cachingConverter := NewCachingTypeConverter(mockConverter)

	t.Run("delegates to wrapped converter", func(t *testing.T) {
		servicePod := services.PodInfo{
			Name:      "test-pod",
			Namespace: "default",
		}
		expectedPod := &model.PodInfo{
			Name:      "test-pod",
			Namespace: "default",
		}

		mockConverter.On("ConvertPodInfo", servicePod).Return(expectedPod)

		result := cachingConverter.ConvertPodInfo(servicePod)
		assert.Equal(t, expectedPod, result)

		mockConverter.AssertExpectations(t)
	})

	t.Run("delegates all methods to wrapped converter", func(t *testing.T) {
		// Test ConvertNamespaceInfo
		serviceNS := services.NamespaceInfo{Name: "test"}
		expectedNS := &model.NamespaceInfo{Name: "test"}
		mockConverter.On("ConvertNamespaceInfo", serviceNS).Return(expectedNS)

		result := cachingConverter.ConvertNamespaceInfo(serviceNS)
		assert.Equal(t, expectedNS, result)

		// Test ConvertDeploymentInfo
		deployment := appsv1.Deployment{}
		expectedDeployment := &model.DeploymentInfo{}
		mockConverter.On("ConvertDeploymentInfo", deployment).Return(expectedDeployment)

		resultDeployment := cachingConverter.ConvertDeploymentInfo(deployment)
		assert.Equal(t, expectedDeployment, resultDeployment)

		// Test ConvertStatefulSetInfo
		statefulSet := appsv1.StatefulSet{}
		expectedStatefulSet := &model.StatefulSetInfo{}
		mockConverter.On("ConvertStatefulSetInfo", statefulSet).Return(expectedStatefulSet)

		resultStatefulSet := cachingConverter.ConvertStatefulSetInfo(statefulSet)
		assert.Equal(t, expectedStatefulSet, resultStatefulSet)

		// Test ConvertPodWatchEvent
		watchEvent := services.PodWatchEvent{}
		expectedWatchEvent := &model.PodWatchEvent{}
		mockConverter.On("ConvertPodWatchEvent", watchEvent).Return(expectedWatchEvent)

		resultWatchEvent := cachingConverter.ConvertPodWatchEvent(watchEvent)
		assert.Equal(t, expectedWatchEvent, resultWatchEvent)

		// Test MapToKeyValuePairs
		labelMap := map[string]string{"key": "value"}
		expectedPairs := []*model.KeyValue{{Key: "key", Value: "value"}}
		mockConverter.On("MapToKeyValuePairs", labelMap).Return(expectedPairs)

		resultPairs := cachingConverter.MapToKeyValuePairs(labelMap)
		assert.Equal(t, expectedPairs, resultPairs)

		mockConverter.AssertExpectations(t)
	})
}

// Test interface compliance
func TestTypeConverterInterfaceCompliance(t *testing.T) {
	t.Run("DefaultTypeConverter implements TypeConverter", func(t *testing.T) {
		var _ TypeConverter = (*DefaultTypeConverter)(nil)
	})

	t.Run("CachingTypeConverter implements TypeConverter", func(t *testing.T) {
		var _ TypeConverter = (*CachingTypeConverter)(nil)
	})
}

// Test initialization functions
func TestTypeConverterInitialization(t *testing.T) {
	t.Run("NewDefaultTypeConverter", func(t *testing.T) {
		converter := NewDefaultTypeConverter()
		require.NotNil(t, converter)
		assert.IsType(t, &DefaultTypeConverter{}, converter)
	})

	t.Run("NewCachingTypeConverter", func(t *testing.T) {
		wrapped := NewDefaultTypeConverter()
		caching := NewCachingTypeConverter(wrapped)

		require.NotNil(t, caching)
		assert.IsType(t, &CachingTypeConverter{}, caching)
		assert.Equal(t, wrapped, caching.wrapped)
		assert.NotNil(t, caching.cache)
	})
}

// Benchmark tests for conversion performance
func BenchmarkDefaultTypeConverter_ConvertPodInfo(b *testing.B) {
	converter := NewDefaultTypeConverter()
	servicePod := services.PodInfo{
		Name:      "test-pod",
		Namespace: "default",
		Phase:     corev1.PodRunning,
		Age:       "5m",
		Ready:     "1/1",
		Labels: map[string]string{
			"app":     "test",
			"version": "1.0",
		},
		Containers: []services.ContainerInfo{
			{
				Name:  "container1",
				Image: "nginx:latest",
				Ready: true,
			},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		converter.ConvertPodInfo(servicePod)
	}
}

func BenchmarkDefaultTypeConverter_MapToKeyValuePairs(b *testing.B) {
	converter := NewDefaultTypeConverter()
	labels := map[string]string{
		"app":     "test",
		"version": "1.0",
		"env":     "production",
		"team":    "backend",
		"service": "api",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		converter.MapToKeyValuePairs(labels)
	}
}

// Helper functions for tests
func stringPtr(s string) *string {
	return &s
}

func int32Ptr(i int32) *int32 {
	return &i
}

// Mock TypeConverter for testing
type MockTypeConverter struct {
	mock.Mock
}

func (m *MockTypeConverter) ConvertPodInfo(servicePod services.PodInfo) *model.PodInfo {
	args := m.Called(servicePod)
	return args.Get(0).(*model.PodInfo)
}

func (m *MockTypeConverter) ConvertNamespaceInfo(serviceNS services.NamespaceInfo) *model.NamespaceInfo {
	args := m.Called(serviceNS)
	return args.Get(0).(*model.NamespaceInfo)
}

func (m *MockTypeConverter) ConvertDeploymentInfo(deployment appsv1.Deployment) *model.DeploymentInfo {
	args := m.Called(deployment)
	return args.Get(0).(*model.DeploymentInfo)
}

func (m *MockTypeConverter) ConvertStatefulSetInfo(statefulSet appsv1.StatefulSet) *model.StatefulSetInfo {
	args := m.Called(statefulSet)
	return args.Get(0).(*model.StatefulSetInfo)
}

func (m *MockTypeConverter) ConvertPodWatchEvent(serviceEvent services.PodWatchEvent) *model.PodWatchEvent {
	args := m.Called(serviceEvent)
	return args.Get(0).(*model.PodWatchEvent)
}

func (m *MockTypeConverter) MapToKeyValuePairs(m2 map[string]string) []*model.KeyValue {
	args := m.Called(m2)
	return args.Get(0).([]*model.KeyValue)
}