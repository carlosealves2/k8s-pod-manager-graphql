package services

import (
	"context"
	"fmt"
	"time"

	"github.com/carlosf/k8s-pod-manager/internal/kubernetes"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// namespaceServiceImpl implements NamespaceService interface
type namespaceServiceImpl struct {
	client  kubernetes.ClientInterface
	options ServiceOptions
}

// NewNamespaceService creates a new namespace service
func NewNamespaceService(
	client kubernetes.ClientInterface,
	options ServiceOptions,
) NamespaceService {
	return &namespaceServiceImpl{
		client:  client,
		options: options,
	}
}

// ListNamespaces implements NamespaceService
func (s *namespaceServiceImpl) ListNamespaces(ctx context.Context, opts *ListOptions) ([]NamespaceInfo, error) {
	listOpts := metav1.ListOptions{}
	if opts != nil {
		listOpts = s.buildListOptions(opts)
	}

	namespaces, err := s.client.CoreV1().Namespaces().List(ctx, listOpts)
	if err != nil {
		return nil, NewInternalError("failed to list namespaces", err)
	}

	namespaceInfos := make([]NamespaceInfo, 0, len(namespaces.Items))
	now := time.Now()

	for _, ns := range namespaces.Items {
		namespaceInfo := s.convertNamespaceToInfo(ns, now)
		namespaceInfos = append(namespaceInfos, namespaceInfo)
	}

	return namespaceInfos, nil
}

// GetNamespace implements NamespaceService
func (s *namespaceServiceImpl) GetNamespace(ctx context.Context, name string) (*NamespaceInfo, error) {
	if name == "" {
		return nil, NewValidationError("namespace name is required", nil)
	}

	namespace, err := s.client.CoreV1().Namespaces().Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, NewNotFoundError("namespace", name)
	}

	namespaceInfo := s.convertNamespaceToInfo(*namespace, time.Now())
	return &namespaceInfo, nil
}

// Helper methods

func (s *namespaceServiceImpl) convertNamespaceToInfo(ns corev1.Namespace, now time.Time) NamespaceInfo {
	age := now.Sub(ns.CreationTimestamp.Time)
	ageStr := s.formatDuration(age)

	status := "Active"
	if ns.Status.Phase != corev1.NamespaceActive {
		status = string(ns.Status.Phase)
	}

	return NamespaceInfo{
		Name:        ns.Name,
		Status:      status,
		Age:         ageStr,
		Labels:      ns.Labels,
		Annotations: ns.Annotations,
		CreatedAt:   ns.CreationTimestamp.Time,
	}
}

func (s *namespaceServiceImpl) formatDuration(d time.Duration) string {
	if d < 0 {
		return "Unknown"
	}

	days := int(d.Hours() / 24)
	hours := int(d.Hours()) % 24
	minutes := int(d.Minutes()) % 60

	if days > 0 {
		if hours > 0 {
			return fmt.Sprintf("%dd%dh", days, hours)
		}
		return fmt.Sprintf("%dd", days)
	}
	if hours > 0 {
		if minutes > 0 {
			return fmt.Sprintf("%dh%dm", hours, minutes)
		}
		return fmt.Sprintf("%dh", hours)
	}
	if minutes > 0 {
		return fmt.Sprintf("%dm", minutes)
	}
	return fmt.Sprintf("%ds", int(d.Seconds()))
}

func (s *namespaceServiceImpl) buildListOptions(opts *ListOptions) metav1.ListOptions {
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