package handlers

import (
	"context"
	"net/http"

	"github.com/carlosf/k8s-pod-manager/internal/services"
	"github.com/gin-gonic/gin"
)

// PodService interface for dependency injection
type PodService interface {
	ListPods(ctx context.Context, namespace string) ([]services.PodInfo, error)
	GetPod(ctx context.Context, namespace, podName string) (*services.PodInfo, error)
	RestartPod(ctx context.Context, namespace, podName, executedBy string) (interface{}, error)
	DeletePod(ctx context.Context, namespace, podName, executedBy string) error
	ScaleDeployment(ctx context.Context, namespace, deploymentName string, replicas int32, executedBy string) (interface{}, error)
	ListDeployments(ctx context.Context, namespace string) (interface{}, error)
	ScaleStatefulSet(ctx context.Context, namespace, statefulSetName string, replicas int32, executedBy string) (interface{}, error)
	ListStatefulSets(ctx context.Context, namespace string) (interface{}, error)
	ListNamespaces(ctx context.Context) (interface{}, error)
}

// RefactoredPodHandler provides Kubernetes pod management endpoints with improved architecture
type RefactoredPodHandler struct {
	service   PodService
	validator *Validator
}

// NewRefactoredPodHandler creates a new refactored pod handler with dependency injection
func NewRefactoredPodHandler(service PodService) *RefactoredPodHandler {
	return &RefactoredPodHandler{
		service:   service,
		validator: NewValidator(),
	}
}

// ListPods handles listing pods in a namespace
func (h *RefactoredPodHandler) ListPods(c *gin.Context) {
	ctx := NewGinContextAdapter(c)
	h.handleListPods(ctx)
}

func (h *RefactoredPodHandler) handleListPods(ctx Context) {
	requestID := ctx.Get("X-Request-ID")
	builder := NewResponseBuilder(requestID)

	namespace := ctx.Param("namespace")
	if namespace == "" {
		namespace = "default"
	}

	// Validate namespace
	if errors := h.validator.ValidateNamespace(namespace); len(errors) > 0 {
		response := builder.ValidationError("Invalid namespace", errors)
		ctx.JSON(HTTPStatusFromErrorCode(ErrCodeValidation), response)
		return
	}

	pods, err := h.service.ListPods(ctx.Context(), namespace)
	if err != nil {
		response := builder.InternalError("Failed to list pods: " + err.Error())
		ctx.JSON(http.StatusInternalServerError, response)
		return
	}

	data := map[string]interface{}{
		"namespace": namespace,
		"count":     len(pods),
		"pods":      pods,
	}

	response := builder.Success(data)
	ctx.JSON(http.StatusOK, response)
}

// GetPod handles getting a specific pod
func (h *RefactoredPodHandler) GetPod(c *gin.Context) {
	ctx := NewGinContextAdapter(c)
	h.handleGetPod(ctx)
}

func (h *RefactoredPodHandler) handleGetPod(ctx Context) {
	requestID := ctx.Get("X-Request-ID")
	builder := NewResponseBuilder(requestID)

	namespace := ctx.Param("namespace")
	podName := ctx.Param("pod")

	if namespace == "" {
		namespace = "default"
	}

	// Validate inputs
	var allErrors []ValidationError
	allErrors = append(allErrors, h.validator.ValidateNamespace(namespace)...)
	allErrors = append(allErrors, h.validator.ValidatePodName(podName)...)

	if len(allErrors) > 0 {
		response := builder.ValidationError("Invalid input parameters", allErrors)
		ctx.JSON(HTTPStatusFromErrorCode(ErrCodeValidation), response)
		return
	}

	pod, err := h.service.GetPod(ctx.Context(), namespace, podName)
	if err != nil {
		response := builder.NotFoundError("Pod")
		ctx.JSON(http.StatusNotFound, response)
		return
	}

	response := builder.Success(pod)
	ctx.JSON(http.StatusOK, response)
}

// RestartPod handles restarting a pod
func (h *RefactoredPodHandler) RestartPod(c *gin.Context) {
	ctx := NewGinContextAdapter(c)
	h.handleRestartPod(ctx)
}

func (h *RefactoredPodHandler) handleRestartPod(ctx Context) {
	requestID := ctx.Get("X-Request-ID")
	builder := NewResponseBuilder(requestID)

	namespace := ctx.Param("namespace")
	podName := ctx.Param("pod")

	if namespace == "" {
		namespace = "default"
	}

	// Validate inputs
	var allErrors []ValidationError
	allErrors = append(allErrors, h.validator.ValidateNamespace(namespace)...)
	allErrors = append(allErrors, h.validator.ValidatePodName(podName)...)

	if len(allErrors) > 0 {
		response := builder.ValidationError("Invalid input parameters", allErrors)
		ctx.JSON(HTTPStatusFromErrorCode(ErrCodeValidation), response)
		return
	}

	executedBy := h.validator.ValidateUserHeader(ctx.Get("X-User"))

	result, err := h.service.RestartPod(ctx.Context(), namespace, podName, executedBy)
	if err != nil {
		response := builder.InternalError("Failed to restart pod: " + err.Error())
		ctx.JSON(http.StatusInternalServerError, response)
		return
	}

	response := builder.Success(result)
	ctx.JSON(http.StatusOK, response)
}

// DeletePod handles deleting a pod
func (h *RefactoredPodHandler) DeletePod(c *gin.Context) {
	ctx := NewGinContextAdapter(c)
	h.handleDeletePod(ctx)
}

func (h *RefactoredPodHandler) handleDeletePod(ctx Context) {
	requestID := ctx.Get("X-Request-ID")
	builder := NewResponseBuilder(requestID)

	namespace := ctx.Param("namespace")
	podName := ctx.Param("pod")

	if namespace == "" {
		namespace = "default"
	}

	// Validate inputs
	var allErrors []ValidationError
	allErrors = append(allErrors, h.validator.ValidateNamespace(namespace)...)
	allErrors = append(allErrors, h.validator.ValidatePodName(podName)...)

	if len(allErrors) > 0 {
		response := builder.ValidationError("Invalid input parameters", allErrors)
		ctx.JSON(HTTPStatusFromErrorCode(ErrCodeValidation), response)
		return
	}

	executedBy := h.validator.ValidateUserHeader(ctx.Get("X-User"))

	err := h.service.DeletePod(ctx.Context(), namespace, podName, executedBy)
	if err != nil {
		response := builder.InternalError("Failed to delete pod: " + err.Error())
		ctx.JSON(http.StatusInternalServerError, response)
		return
	}

	data := map[string]interface{}{
		"message":   "Pod deleted successfully",
		"pod":       podName,
		"namespace": namespace,
	}

	response := builder.Success(data)
	ctx.JSON(http.StatusOK, response)
}

// ScaleDeployment handles scaling a deployment
func (h *RefactoredPodHandler) ScaleDeployment(c *gin.Context) {
	ctx := NewGinContextAdapter(c)
	h.handleScaleDeployment(ctx)
}

func (h *RefactoredPodHandler) handleScaleDeployment(ctx Context) {
	requestID := ctx.Get("X-Request-ID")
	builder := NewResponseBuilder(requestID)

	namespace := ctx.Param("namespace")
	deploymentName := ctx.Param("deployment")

	if namespace == "" {
		namespace = "default"
	}

	// Parse request body
	var scaleReq ScaleRequest
	if err := ctx.BodyParser(&scaleReq); err != nil {
		response := builder.ValidationError("Invalid request body", err.Error())
		ctx.JSON(http.StatusBadRequest, response)
		return
	}

	// Validate inputs
	var allErrors []ValidationError
	allErrors = append(allErrors, h.validator.ValidateNamespace(namespace)...)
	allErrors = append(allErrors, h.validator.ValidateDeploymentName(deploymentName)...)
	allErrors = append(allErrors, h.validator.ValidateScaleRequest(&scaleReq)...)

	if len(allErrors) > 0 {
		response := builder.ValidationError("Invalid input parameters", allErrors)
		ctx.JSON(HTTPStatusFromErrorCode(ErrCodeValidation), response)
		return
	}

	executedBy := h.validator.ValidateUserHeader(ctx.Get("X-User"))

	result, err := h.service.ScaleDeployment(ctx.Context(), namespace, deploymentName, *scaleReq.Replicas, executedBy)
	if err != nil {
		response := builder.InternalError("Failed to scale deployment: " + err.Error())
		ctx.JSON(http.StatusInternalServerError, response)
		return
	}

	response := builder.Success(result)
	ctx.JSON(http.StatusOK, response)
}

// RegisterRoutes registers all pod management routes
func (h *RefactoredPodHandler) RegisterRoutes(router Router) {
	router.GET("/namespaces/:namespace/pods", h.handleListPods)
	router.GET("/namespaces/:namespace/pods/:pod", h.handleGetPod)
	router.POST("/namespaces/:namespace/pods/:pod/restart", h.handleRestartPod)
	router.DELETE("/namespaces/:namespace/pods/:pod", h.handleDeletePod)
	router.POST("/namespaces/:namespace/deployments/:deployment/scale", h.handleScaleDeployment)
	// Add other routes as needed...
}