package models

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"gorm.io/gorm"
)

type AuditTestSuite struct {
	suite.Suite
}

func (s *AuditTestSuite) TestAuditLogStruct() {
	now := time.Now()

	auditLog := AuditLog{
		ID:          1,
		Action:      "create_pod",
		ResourceType: "pod",
		ResourceName: "test-pod",
		Namespace:   "default",
		User:        "admin",
		UserAgent:   "kubectl/v1.25.0",
		IP:          "192.168.1.100",
		Method:      "POST",
		Path:        "/api/v1/namespaces/default/pods",
		StatusCode:  201,
		RequestBody: `{"apiVersion": "v1", "kind": "Pod"}`,
		Response:    `{"status": "success"}`,
		Error:       "",
		Duration:    150,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	// Test all fields are properly set
	s.Equal(uint(1), auditLog.ID)
	s.Equal("create_pod", auditLog.Action)
	s.Equal("pod", auditLog.ResourceType)
	s.Equal("test-pod", auditLog.ResourceName)
	s.Equal("default", auditLog.Namespace)
	s.Equal("admin", auditLog.User)
	s.Equal("kubectl/v1.25.0", auditLog.UserAgent)
	s.Equal("192.168.1.100", auditLog.IP)
	s.Equal("POST", auditLog.Method)
	s.Equal("/api/v1/namespaces/default/pods", auditLog.Path)
	s.Equal(201, auditLog.StatusCode)
	s.Equal(`{"apiVersion": "v1", "kind": "Pod"}`, auditLog.RequestBody)
	s.Equal(`{"status": "success"}`, auditLog.Response)
	s.Equal("", auditLog.Error)
	s.Equal(int64(150), auditLog.Duration)
	s.Equal(now, auditLog.CreatedAt)
	s.Equal(now, auditLog.UpdatedAt)
}

func (s *AuditTestSuite) TestAuditLogTableName() {
	auditLog := AuditLog{}
	s.Equal("audit_logs", auditLog.TableName())
}

func (s *AuditTestSuite) TestAuditLogWithError() {
	auditLog := AuditLog{
		Action:      "delete_pod",
		ResourceType: "pod",
		ResourceName: "failed-pod",
		Namespace:   "kube-system",
		User:        "user",
		Method:      "DELETE",
		Path:        "/api/v1/namespaces/kube-system/pods/failed-pod",
		StatusCode:  500,
		Error:       "Pod not found",
		Duration:    50,
	}

	s.Equal("delete_pod", auditLog.Action)
	s.Equal("Pod not found", auditLog.Error)
	s.Equal(500, auditLog.StatusCode)
}

func (s *AuditTestSuite) TestPodOperationStruct() {
	now := time.Now()

	podOp := PodOperation{
		ID:           1,
		PodName:      "test-pod",
		Namespace:    "default",
		Operation:    "restart",
		Status:       "success",
		Controller:   "Deployment",
		ControllerName: "test-deployment",
		Message:      "Pod restarted successfully",
		Details:      `{"controller_type": "Deployment", "replicas": 3}`,
		ExecutedBy:   "admin",
		ExecutedAt:   now,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	// Test all fields are properly set
	s.Equal(uint(1), podOp.ID)
	s.Equal("test-pod", podOp.PodName)
	s.Equal("default", podOp.Namespace)
	s.Equal("restart", podOp.Operation)
	s.Equal("success", podOp.Status)
	s.Equal("Deployment", podOp.Controller)
	s.Equal("test-deployment", podOp.ControllerName)
	s.Equal("Pod restarted successfully", podOp.Message)
	s.Equal(`{"controller_type": "Deployment", "replicas": 3}`, podOp.Details)
	s.Equal("admin", podOp.ExecutedBy)
	s.Equal(now, podOp.ExecutedAt)
	s.Equal(now, podOp.CreatedAt)
	s.Equal(now, podOp.UpdatedAt)
}

func (s *AuditTestSuite) TestPodOperationTableName() {
	podOp := PodOperation{}
	s.Equal("pod_operations", podOp.TableName())
}

func (s *AuditTestSuite) TestPodOperationFailure() {
	podOp := PodOperation{
		PodName:      "failed-pod",
		Namespace:    "test",
		Operation:    "delete",
		Status:       "failed",
		Message:      "Pod deletion failed: permission denied",
		ExecutedBy:   "user",
		ExecutedAt:   time.Now(),
	}

	s.Equal("failed", podOp.Status)
	s.Equal("Pod deletion failed: permission denied", podOp.Message)
	s.Equal("delete", podOp.Operation)
}

func (s *AuditTestSuite) TestDeploymentScaleStruct() {
	now := time.Now()

	deploymentScale := DeploymentScale{
		ID:              1,
		DeploymentName:  "nginx-deployment",
		Namespace:       "production",
		PreviousReplicas: 3,
		NewReplicas:     5,
		Reason:          "Scaling up due to increased load",
		ExecutedBy:      "admin",
		ExecutedAt:      now,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	// Test all fields are properly set
	s.Equal(uint(1), deploymentScale.ID)
	s.Equal("nginx-deployment", deploymentScale.DeploymentName)
	s.Equal("production", deploymentScale.Namespace)
	s.Equal(int32(3), deploymentScale.PreviousReplicas)
	s.Equal(int32(5), deploymentScale.NewReplicas)
	s.Equal("Scaling up due to increased load", deploymentScale.Reason)
	s.Equal("admin", deploymentScale.ExecutedBy)
	s.Equal(now, deploymentScale.ExecutedAt)
	s.Equal(now, deploymentScale.CreatedAt)
	s.Equal(now, deploymentScale.UpdatedAt)
}

func (s *AuditTestSuite) TestDeploymentScaleTableName() {
	deploymentScale := DeploymentScale{}
	s.Equal("deployment_scales", deploymentScale.TableName())
}

func (s *AuditTestSuite) TestDeploymentScaleDown() {
	deploymentScale := DeploymentScale{
		DeploymentName:  "test-app",
		Namespace:       "staging",
		PreviousReplicas: 10,
		NewReplicas:     2,
		Reason:          "Scaling down for maintenance",
		ExecutedBy:      "devops",
		ExecutedAt:      time.Now(),
	}

	s.Equal(int32(10), deploymentScale.PreviousReplicas)
	s.Equal(int32(2), deploymentScale.NewReplicas)
	s.Equal("Scaling down for maintenance", deploymentScale.Reason)
	s.True(deploymentScale.PreviousReplicas > deploymentScale.NewReplicas)
}

func (s *AuditTestSuite) TestDeploymentScaleToZero() {
	deploymentScale := DeploymentScale{
		DeploymentName:  "temp-job",
		Namespace:       "jobs",
		PreviousReplicas: 1,
		NewReplicas:     0,
		Reason:          "Stopping deployment",
		ExecutedBy:      "system",
		ExecutedAt:      time.Now(),
	}

	s.Equal(int32(1), deploymentScale.PreviousReplicas)
	s.Equal(int32(0), deploymentScale.NewReplicas)
	s.Equal("Stopping deployment", deploymentScale.Reason)
}

func TestAuditSuite(t *testing.T) {
	suite.Run(t, new(AuditTestSuite))
}

// Table-driven tests for various operation types
func TestPodOperationTypes(t *testing.T) {
	tests := []struct {
		name          string
		operation     string
		status        string
		expectedValid bool
	}{
		{
			name:          "valid restart operation",
			operation:     "restart",
			status:        "success",
			expectedValid: true,
		},
		{
			name:          "valid delete operation",
			operation:     "delete",
			status:        "success",
			expectedValid: true,
		},
		{
			name:          "valid scale operation",
			operation:     "scale",
			status:        "success",
			expectedValid: true,
		},
		{
			name:          "failed operation",
			operation:     "restart",
			status:        "failed",
			expectedValid: true,
		},
		{
			name:          "pending operation",
			operation:     "delete",
			status:        "pending",
			expectedValid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			podOp := PodOperation{
				PodName:    "test-pod",
				Namespace:  "default",
				Operation:  tt.operation,
				Status:     tt.status,
				ExecutedBy: "test-user",
				ExecutedAt: time.Now(),
			}

			assert.Equal(t, tt.operation, podOp.Operation)
			assert.Equal(t, tt.status, podOp.Status)
			assert.NotEmpty(t, podOp.PodName)
			assert.NotEmpty(t, podOp.Namespace)
			assert.NotEmpty(t, podOp.ExecutedBy)
			assert.False(t, podOp.ExecutedAt.IsZero())
		})
	}
}

// Test controller types for pod operations
func TestPodOperationControllerTypes(t *testing.T) {
	tests := []struct {
		name           string
		controllerType string
		controllerName string
	}{
		{
			name:           "deployment controller",
			controllerType: "Deployment",
			controllerName: "nginx-deployment",
		},
		{
			name:           "statefulset controller",
			controllerType: "StatefulSet",
			controllerName: "postgres-statefulset",
		},
		{
			name:           "daemonset controller",
			controllerType: "DaemonSet",
			controllerName: "fluentd-daemonset",
		},
		{
			name:           "job controller",
			controllerType: "Job",
			controllerName: "backup-job",
		},
		{
			name:           "cronjob controller",
			controllerType: "CronJob",
			controllerName: "cleanup-cronjob",
		},
		{
			name:           "replicaset controller",
			controllerType: "ReplicaSet",
			controllerName: "legacy-replicaset",
		},
		{
			name:           "standalone pod",
			controllerType: "",
			controllerName: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			podOp := PodOperation{
				PodName:        "test-pod",
				Namespace:      "default",
				Operation:      "restart",
				Status:         "success",
				Controller:     tt.controllerType,
				ControllerName: tt.controllerName,
				ExecutedBy:     "test-user",
				ExecutedAt:     time.Now(),
			}

			assert.Equal(t, tt.controllerType, podOp.Controller)
			assert.Equal(t, tt.controllerName, podOp.ControllerName)
		})
	}
}

// Test audit log action types
func TestAuditLogActions(t *testing.T) {
	tests := []struct {
		name         string
		action       string
		resourceType string
		resourceName string
		method       string
		path         string
		statusCode   int
	}{
		{
			name:         "get pod action",
			action:       "get_pod",
			resourceType: "pod",
			resourceName: "nginx-pod",
			method:       "GET",
			path:         "/api/v1/namespaces/default/pods/nginx-pod",
			statusCode:   200,
		},
		{
			name:         "list pods action",
			action:       "list_pods",
			resourceType: "pod",
			resourceName: "all",
			method:       "GET",
			path:         "/api/v1/namespaces/default/pods",
			statusCode:   200,
		},
		{
			name:         "delete pod action",
			action:       "delete_pod",
			resourceType: "pod",
			resourceName: "old-pod",
			method:       "DELETE",
			path:         "/api/v1/namespaces/default/pods/old-pod",
			statusCode:   204,
		},
		{
			name:         "scale deployment action",
			action:       "scale_deployment",
			resourceType: "deployment",
			resourceName: "web-app",
			method:       "PATCH",
			path:         "/api/v1/namespaces/default/deployments/web-app/scale",
			statusCode:   200,
		},
		{
			name:         "health check action",
			action:       "health_check",
			resourceType: "system",
			resourceName: "health",
			method:       "GET",
			path:         "/api/v1/health",
			statusCode:   200,
		},
		{
			name:         "graphql mutation",
			action:       "graphql_request",
			resourceType: "graphql",
			resourceName: "mutation",
			method:       "POST",
			path:         "/graphql",
			statusCode:   200,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			auditLog := AuditLog{
				Action:      tt.action,
				ResourceType: tt.resourceType,
				ResourceName: tt.resourceName,
				Method:      tt.method,
				Path:        tt.path,
				StatusCode:  tt.statusCode,
				User:        "test-user",
				IP:          "127.0.0.1",
				Duration:    100,
				CreatedAt:   time.Now(),
			}

			assert.Equal(t, tt.action, auditLog.Action)
			assert.Equal(t, tt.resourceType, auditLog.ResourceType)
			assert.Equal(t, tt.resourceName, auditLog.ResourceName)
			assert.Equal(t, tt.method, auditLog.Method)
			assert.Equal(t, tt.path, auditLog.Path)
			assert.Equal(t, tt.statusCode, auditLog.StatusCode)
		})
	}
}

// Test soft delete functionality
func TestSoftDeleteFields(t *testing.T) {
	t.Run("audit log soft delete", func(t *testing.T) {
		auditLog := AuditLog{
			Action:      "test_action",
			ResourceType: "test",
			ResourceName: "test-resource",
			DeletedAt:   gorm.DeletedAt{},
		}

		assert.False(t, auditLog.DeletedAt.Valid)
		assert.True(t, auditLog.DeletedAt.Time.IsZero())
	})

	t.Run("pod operation soft delete", func(t *testing.T) {
		podOp := PodOperation{
			PodName:   "test-pod",
			Namespace: "default",
			Operation: "restart",
			DeletedAt: gorm.DeletedAt{},
		}

		assert.False(t, podOp.DeletedAt.Valid)
		assert.True(t, podOp.DeletedAt.Time.IsZero())
	})

	t.Run("deployment scale soft delete", func(t *testing.T) {
		deploymentScale := DeploymentScale{
			DeploymentName: "test-deployment",
			Namespace:      "default",
			DeletedAt:      gorm.DeletedAt{},
		}

		assert.False(t, deploymentScale.DeletedAt.Valid)
		assert.True(t, deploymentScale.DeletedAt.Time.IsZero())
	})
}

// Benchmark tests for performance
func BenchmarkAuditLogCreation(b *testing.B) {
	for i := 0; i < b.N; i++ {
		auditLog := AuditLog{
			Action:      "benchmark_action",
			ResourceType: "benchmark",
			ResourceName: "test",
			User:        "bench-user",
			Method:      "GET",
			Path:        "/api/test",
			StatusCode:  200,
			Duration:    100,
			CreatedAt:   time.Now(),
		}
		_ = auditLog.TableName()
	}
}

func BenchmarkPodOperationCreation(b *testing.B) {
	for i := 0; i < b.N; i++ {
		podOp := PodOperation{
			PodName:    "bench-pod",
			Namespace:  "default",
			Operation:  "restart",
			Status:     "success",
			ExecutedBy: "bench-user",
			ExecutedAt: time.Now(),
		}
		_ = podOp.TableName()
	}
}

func BenchmarkDeploymentScaleCreation(b *testing.B) {
	for i := 0; i < b.N; i++ {
		deploymentScale := DeploymentScale{
			DeploymentName:  "bench-deployment",
			Namespace:       "default",
			PreviousReplicas: 3,
			NewReplicas:     5,
			ExecutedBy:      "bench-user",
			ExecutedAt:      time.Now(),
		}
		_ = deploymentScale.TableName()
	}
}