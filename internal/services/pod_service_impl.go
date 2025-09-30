package services

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"time"

	"github.com/carlosf/k8s-pod-manager/internal/kubernetes"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
)

// podServiceImpl implements PodService with single responsibility for pod operations
type podServiceImpl struct {
	client              kubernetes.ClientInterface
	controllerDetector  ControllerDetector
	restartOrchestrator RestartOrchestrator
	podConverter        PodConverter
	auditService        AuditService
	options             ServiceOptions
}

// NewPodService creates a new pod service with dependency injection
func NewPodService(
	client kubernetes.ClientInterface,
	controllerDetector ControllerDetector,
	restartOrchestrator RestartOrchestrator,
	podConverter PodConverter,
	auditService AuditService,
	options ServiceOptions,
) PodService {
	return &podServiceImpl{
		client:              client,
		controllerDetector:  controllerDetector,
		restartOrchestrator: restartOrchestrator,
		podConverter:        podConverter,
		auditService:        auditService,
		options:             options,
	}
}

// ListPods implements PodService
func (s *podServiceImpl) ListPods(ctx context.Context, namespace string, opts *ListOptions) ([]PodInfo, error) {
	listOpts := metav1.ListOptions{}
	if opts != nil {
		listOpts = s.buildListOptions(opts)
	}

	pods, err := s.client.CoreV1().Pods(namespace).List(ctx, listOpts)
	if err != nil {
		return nil, NewInternalError(
			fmt.Sprintf("failed to list pods in namespace '%s'", namespace),
			err,
		)
	}

	return s.podConverter.ConvertPods(pods.Items), nil
}

// ListAllPods implements PodService
func (s *podServiceImpl) ListAllPods(ctx context.Context, opts *ListOptions) ([]PodInfo, error) {
	namespace := ""
	if opts != nil && opts.Namespace != "" {
		namespace = opts.Namespace
	}

	listOpts := metav1.ListOptions{}
	if opts != nil {
		listOpts = s.buildListOptions(opts)
	}

	pods, err := s.client.CoreV1().Pods(namespace).List(ctx, listOpts)
	if err != nil {
		return nil, NewInternalError("failed to list all pods", err)
	}

	return s.podConverter.ConvertPods(pods.Items), nil
}

// GetPod implements PodService
func (s *podServiceImpl) GetPod(ctx context.Context, namespace, name string) (*PodInfo, error) {
	if namespace == "" || name == "" {
		return nil, NewValidationError("namespace and name are required", nil)
	}

	pod, err := s.client.CoreV1().Pods(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, NewNotFoundError("pod", fmt.Sprintf("%s/%s", namespace, name))
	}

	controllerInfo := s.controllerDetector.DetectController(ctx, *pod)
	podInfo := s.podConverter.ConvertPod(*pod, controllerInfo)

	return &podInfo, nil
}

// DeletePod implements PodService
func (s *podServiceImpl) DeletePod(ctx context.Context, namespace, podName, executedBy string) error {
	if err := s.validatePodParams(namespace, podName, executedBy); err != nil {
		return err
	}

	audit := PodOperationAudit{
		PodName:   podName,
		Namespace: namespace,
		Operation: "delete",
		ExecutedBy: executedBy,
	}

	err := s.client.CoreV1().Pods(namespace).Delete(ctx, podName, metav1.DeleteOptions{})
	if err != nil {
		audit.Status = "failed"
		audit.Message = err.Error()
		if s.options.EnableAudit {
			s.auditService.LogPodOperation(ctx, audit)
		}
		return NewPodOperationError("delete", podName, namespace, "failed to delete pod", err)
	}

	audit.Status = "success"
	audit.Message = "Pod deleted successfully"
	if s.options.EnableAudit {
		s.auditService.LogPodOperation(ctx, audit)
	}

	return nil
}

// RestartPod implements PodService
func (s *podServiceImpl) RestartPod(ctx context.Context, namespace, podName, executedBy string) (*RestartResult, error) {
	if err := s.validatePodParams(namespace, podName, executedBy); err != nil {
		return nil, err
	}

	// Delegate to restart orchestrator which handles the complexity
	result, err := s.restartOrchestrator.RestartPod(ctx, namespace, podName, executedBy)
	if err != nil {
		return nil, err
	}

	// Audit the operation
	if s.options.EnableAudit {
		audit := PodOperationAudit{
			PodName:        podName,
			Namespace:      namespace,
			Operation:      "restart",
			Controller:     string(result.ControllerType),
			ControllerName: result.ControllerName,
			ExecutedBy:     executedBy,
			Status:         "success",
			Message:        result.Message,
			Details:        result.Details,
		}
		s.auditService.LogPodOperation(ctx, audit)
	}

	return result, nil
}

// WatchPods implements PodService
func (s *podServiceImpl) WatchPods(ctx context.Context, namespace string, opts *WatchOptions, eventsChan chan<- PodWatchEvent) error {
	if eventsChan == nil {
		return NewValidationError("eventsChan is required", nil)
	}

	listOpts := metav1.ListOptions{}
	if opts != nil {
		listOpts = s.buildWatchOptions(opts)
	}

	watcher, err := s.client.CoreV1().Pods(namespace).Watch(ctx, listOpts)
	if err != nil {
		return NewInternalError(
			fmt.Sprintf("failed to create pod watcher for namespace '%s'", namespace),
			err,
		)
	}
	defer watcher.Stop()

	return s.processWatchEvents(ctx, watcher, eventsChan)
}

// WatchAllPods implements PodService
func (s *podServiceImpl) WatchAllPods(ctx context.Context, opts *WatchOptions, eventsChan chan<- PodWatchEvent) error {
	if eventsChan == nil {
		return NewValidationError("eventsChan is required", nil)
	}

	listOpts := metav1.ListOptions{}
	if opts != nil {
		listOpts = s.buildWatchOptions(opts)
	}

	watcher, err := s.client.CoreV1().Pods("").Watch(ctx, listOpts)
	if err != nil {
		return NewInternalError("failed to create all pods watcher", err)
	}
	defer watcher.Stop()

	return s.processWatchEvents(ctx, watcher, eventsChan)
}

// Helper methods

func (s *podServiceImpl) validatePodParams(namespace, podName, executedBy string) error {
	if namespace == "" {
		return NewValidationError("namespace is required", nil)
	}
	if podName == "" {
		return NewValidationError("pod name is required", nil)
	}
	if executedBy == "" {
		return NewValidationError("executedBy is required for audit trail", nil)
	}
	return nil
}

func (s *podServiceImpl) buildListOptions(opts *ListOptions) metav1.ListOptions {
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

func (s *podServiceImpl) buildWatchOptions(opts *WatchOptions) metav1.ListOptions {
	listOpts := metav1.ListOptions{}

	if opts.ResourceVersion != "" {
		listOpts.ResourceVersion = opts.ResourceVersion
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

func (s *podServiceImpl) processWatchEvents(ctx context.Context, watcher watch.Interface, eventsChan chan<- PodWatchEvent) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case event, ok := <-watcher.ResultChan():
			if !ok {
				return NewInternalError("watch channel closed unexpectedly", nil)
			}

			pod, ok := event.Object.(*corev1.Pod)
			if !ok {
				continue // Skip non-pod objects
			}

			controllerInfo := s.controllerDetector.DetectController(ctx, *pod)
			podInfo := s.podConverter.ConvertPod(*pod, controllerInfo)

			watchEvent := PodWatchEvent{
				Type:      string(event.Type),
				Pod:       podInfo,
				Timestamp: time.Now(),
			}

			// Extract reason and message from pod status
			watchEvent.Reason, watchEvent.Message = s.extractEventDetails(*pod)

			select {
			case eventsChan <- watchEvent:
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	}
}

func (s *podServiceImpl) extractEventDetails(pod corev1.Pod) (reason, message string) {
	// Extract reason from pod events if available
	if pod.Status.Phase == corev1.PodFailed || pod.Status.Phase == corev1.PodSucceeded {
		if len(pod.Status.ContainerStatuses) > 0 {
			cs := pod.Status.ContainerStatuses[0]
			if cs.State.Terminated != nil {
				return cs.State.Terminated.Reason, cs.State.Terminated.Message
			} else if cs.State.Waiting != nil {
				return cs.State.Waiting.Reason, cs.State.Waiting.Message
			}
		}
	}

	// Check pod conditions for additional information
	for _, condition := range pod.Status.Conditions {
		if condition.Status == corev1.ConditionFalse && condition.Reason != "" {
			return condition.Reason, condition.Message
		}
	}

	return "", ""
}

// StreamLogs implements PodService for streaming pod logs
func (s *podServiceImpl) StreamLogs(ctx context.Context, namespace, podName string, opts *LogStreamOptions, logsChan chan<- PodLogLine) error {
	// Validate input parameters
	if namespace == "" {
		return NewValidationError("namespace is required", nil)
	}
	if podName == "" {
		return NewValidationError("pod name is required", nil)
	}

	// Verify pod exists and get container info
	pod, err := s.client.CoreV1().Pods(namespace).Get(ctx, podName, metav1.GetOptions{})
	if err != nil {
		return NewNotFoundError("pod", fmt.Sprintf("%s/%s", namespace, podName))
	}

	// Determine container to stream logs from
	containerName := opts.Container
	if containerName == "" {
		// If no container specified and pod has multiple containers, use first one
		if len(pod.Spec.Containers) == 0 {
			return NewValidationError("pod has no containers", nil)
		}
		containerName = pod.Spec.Containers[0].Name
	}

	// Validate container exists in pod
	containerExists := false
	for _, container := range pod.Spec.Containers {
		if container.Name == containerName {
			containerExists = true
			break
		}
	}
	if !containerExists {
		return NewValidationError(
			fmt.Sprintf("container '%s' not found in pod", containerName),
			nil,
		)
	}

	// Build log options
	logOpts := &corev1.PodLogOptions{
		Container:  containerName,
		Follow:     opts.Follow,
		Timestamps: opts.Timestamps,
	}

	if opts.TailLines != nil {
		logOpts.TailLines = opts.TailLines
	}

	if opts.SinceSeconds != nil {
		logOpts.SinceSeconds = opts.SinceSeconds
	}

	// Get log stream from Kubernetes API
	req := s.client.CoreV1().Pods(namespace).GetLogs(podName, logOpts)
	stream, err := req.Stream(ctx)
	if err != nil {
		return NewInternalError(
			fmt.Sprintf("failed to open log stream for pod %s/%s", namespace, podName),
			err,
		)
	}
	defer stream.Close()

	// Stream logs line by line
	return s.streamLogsFromReader(ctx, stream, containerName, logsChan)
}

// streamLogsFromReader reads logs from io.ReadCloser and sends to channel
func (s *podServiceImpl) streamLogsFromReader(ctx context.Context, reader io.ReadCloser, containerName string, logsChan chan<- PodLogLine) error {
	scanner := bufio.NewScanner(reader)

	// Set a reasonable buffer size for long log lines (1MB)
	const maxScanTokenSize = 1024 * 1024
	buf := make([]byte, maxScanTokenSize)
	scanner.Buffer(buf, maxScanTokenSize)

	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			line := scanner.Text()

			logLine := PodLogLine{
				Timestamp: time.Now(),
				Line:      line,
				Container: containerName,
			}

			select {
			case logsChan <- logLine:
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return NewInternalError("error reading log stream", err)
	}

	return nil
}