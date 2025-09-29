//go:build ignore
// +build ignore

package handlers

import (
	"time"

	"github.com/carlosf/k8s-pod-manager/internal/services"
	"github.com/gofiber/fiber/v2"
)

type PodHandler struct {
	service *services.PodService
}

func NewPodHandler() *PodHandler {
	return &PodHandler{
		service: services.NewPodService(),
	}
}

func (h *PodHandler) ListPods(c *fiber.Ctx) error {
	namespace := c.Params("namespace")
	if namespace == "" {
		namespace = "default"
	}

	ctx := c.Context()

	pods, err := h.service.ListPods(ctx, namespace)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"namespace": namespace,
		"count":     len(pods),
		"pods":      pods,
	})
}

func (h *PodHandler) GetPod(c *fiber.Ctx) error {
	namespace := c.Params("namespace")
	podName := c.Params("pod")

	if namespace == "" {
		namespace = "default"
	}

	if podName == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "pod name is required",
		})
	}

	ctx := c.Context()

	pod, err := h.service.GetPod(ctx, namespace, podName)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(pod)
}

func (h *PodHandler) RestartPod(c *fiber.Ctx) error {
	namespace := c.Params("namespace")
	podName := c.Params("pod")

	if namespace == "" {
		namespace = "default"
	}

	if podName == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "pod name is required",
		})
	}

	executedBy := c.Get("X-User", "system")

	ctx := c.Context()

	result, err := h.service.RestartPod(ctx, namespace, podName, executedBy)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(fiber.StatusOK).JSON(result)
}

func (h *PodHandler) DeletePod(c *fiber.Ctx) error {
	namespace := c.Params("namespace")
	podName := c.Params("pod")

	if namespace == "" {
		namespace = "default"
	}

	if podName == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "pod name is required",
		})
	}

	executedBy := c.Get("X-User", "system")

	ctx := c.Context()

	err := h.service.DeletePod(ctx, namespace, podName, executedBy)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message":   "Pod deleted successfully",
		"pod":       podName,
		"namespace": namespace,
	})
}

func (h *PodHandler) ScaleDeployment(c *fiber.Ctx) error {
	namespace := c.Params("namespace")
	deploymentName := c.Params("deployment")

	if namespace == "" {
		namespace = "default"
	}

	if deploymentName == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "deployment name is required",
		})
	}

	var body struct {
		Replicas *int32 `json:"replicas"`
	}

	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	if body.Replicas == nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "replicas field is required",
		})
	}

	if *body.Replicas < 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "replicas must be non-negative",
		})
	}

	executedBy := c.Get("X-User", "system")

	ctx := c.Context()

	result, err := h.service.ScaleDeployment(ctx, namespace, deploymentName, *body.Replicas, executedBy)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(fiber.StatusOK).JSON(result)
}

func (h *PodHandler) ListDeployments(c *fiber.Ctx) error {
	namespace := c.Params("namespace")
	if namespace == "" {
		namespace = "default"
	}

	ctx := c.Context()

	deployments, err := h.service.ListDeployments(ctx, namespace)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	type DeploymentInfo struct {
		Name              string            `json:"name"`
		Namespace         string            `json:"namespace"`
		Replicas          int32             `json:"replicas"`
		UpdatedReplicas   int32             `json:"updated_replicas"`
		ReadyReplicas     int32             `json:"ready_replicas"`
		AvailableReplicas int32             `json:"available_replicas"`
		Labels            map[string]string `json:"labels,omitempty"`
		CreatedAt         time.Time         `json:"created_at"`
	}

	deploymentInfos := make([]DeploymentInfo, 0, len(deployments.Items))
	for _, dep := range deployments.Items {
		info := DeploymentInfo{
			Name:              dep.Name,
			Namespace:         dep.Namespace,
			Replicas:          *dep.Spec.Replicas,
			UpdatedReplicas:   dep.Status.UpdatedReplicas,
			ReadyReplicas:     dep.Status.ReadyReplicas,
			AvailableReplicas: dep.Status.AvailableReplicas,
			Labels:            dep.Labels,
			CreatedAt:         dep.CreationTimestamp.Time,
		}
		deploymentInfos = append(deploymentInfos, info)
	}

	return c.JSON(fiber.Map{
		"namespace":   namespace,
		"count":       len(deploymentInfos),
		"deployments": deploymentInfos,
	})
}

func (h *PodHandler) ListStatefulSets(c *fiber.Ctx) error {
	namespace := c.Params("namespace")
	if namespace == "" {
		namespace = "default"
	}

	ctx := c.Context()

	statefulSets, err := h.service.ListStatefulSets(ctx, namespace)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	type StatefulSetInfo struct {
		Name         string            `json:"name"`
		Namespace    string            `json:"namespace"`
		Replicas     int32             `json:"replicas"`
		ReadyReplicas int32            `json:"ready_replicas"`
		CurrentReplicas int32          `json:"current_replicas"`
		UpdatedReplicas int32          `json:"updated_replicas"`
		Labels       map[string]string `json:"labels,omitempty"`
		CreatedAt    time.Time         `json:"created_at"`
	}

	statefulSetInfos := make([]StatefulSetInfo, 0, len(statefulSets.Items))
	for _, sts := range statefulSets.Items {
		info := StatefulSetInfo{
			Name:            sts.Name,
			Namespace:       sts.Namespace,
			Replicas:        *sts.Spec.Replicas,
			ReadyReplicas:   sts.Status.ReadyReplicas,
			CurrentReplicas: sts.Status.CurrentReplicas,
			UpdatedReplicas: sts.Status.UpdatedReplicas,
			Labels:          sts.Labels,
			CreatedAt:       sts.CreationTimestamp.Time,
		}
		statefulSetInfos = append(statefulSetInfos, info)
	}

	return c.JSON(fiber.Map{
		"namespace":     namespace,
		"count":         len(statefulSetInfos),
		"statefulsets":  statefulSetInfos,
	})
}

func (h *PodHandler) ScaleStatefulSet(c *fiber.Ctx) error {
	namespace := c.Params("namespace")
	statefulSetName := c.Params("statefulset")

	if namespace == "" {
		namespace = "default"
	}

	if statefulSetName == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "statefulset name is required",
		})
	}

	var body struct {
		Replicas *int32 `json:"replicas"`
	}

	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if body.Replicas == nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "replicas field is required",
		})
	}

	if *body.Replicas < 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "replicas must be non-negative",
		})
	}

	executedBy := c.Get("X-User", "system")

	ctx := c.Context()

	result, err := h.service.ScaleStatefulSet(ctx, namespace, statefulSetName, *body.Replicas, executedBy)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(fiber.StatusOK).JSON(result)
}

func (h *PodHandler) ListNamespaces(c *fiber.Ctx) error {
	ctx := c.Context()

	namespaces, err := h.service.ListNamespaces(ctx)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"count":      len(namespaces),
		"namespaces": namespaces,
	})
}