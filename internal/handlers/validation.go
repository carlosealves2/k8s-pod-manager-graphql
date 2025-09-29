package handlers

import (
	"strings"
)

// ValidationError represents a validation error
type ValidationError struct {
	Field   string      `json:"field"`
	Message string      `json:"message"`
	Value   interface{} `json:"value,omitempty"`
}

// Validator provides input validation functionality
type Validator struct{}

// NewValidator creates a new validator
func NewValidator() *Validator {
	return &Validator{}
}

// ValidateNamespace validates a Kubernetes namespace name
func (v *Validator) ValidateNamespace(namespace string) []ValidationError {
	var errors []ValidationError

	if namespace == "" {
		return errors // Empty namespace defaults to "default"
	}

	// Kubernetes namespace naming rules
	if len(namespace) > 63 {
		errors = append(errors, ValidationError{
			Field:   "namespace",
			Message: "namespace name cannot exceed 63 characters",
			Value:   namespace,
		})
	}

	if !v.isValidDNSName(namespace) {
		errors = append(errors, ValidationError{
			Field:   "namespace",
			Message: "namespace must be a valid DNS-1123 label",
			Value:   namespace,
		})
	}

	return errors
}

// ValidatePodName validates a Kubernetes pod name
func (v *Validator) ValidatePodName(podName string) []ValidationError {
	var errors []ValidationError

	if podName == "" {
		errors = append(errors, ValidationError{
			Field:   "pod_name",
			Message: "pod name is required",
			Value:   podName,
		})
		return errors
	}

	if len(podName) > 253 {
		errors = append(errors, ValidationError{
			Field:   "pod_name",
			Message: "pod name cannot exceed 253 characters",
			Value:   podName,
		})
	}

	if !v.isValidDNSName(podName) {
		errors = append(errors, ValidationError{
			Field:   "pod_name",
			Message: "pod name must be a valid DNS-1123 subdomain",
			Value:   podName,
		})
	}

	return errors
}

// ValidateDeploymentName validates a Kubernetes deployment name
func (v *Validator) ValidateDeploymentName(deploymentName string) []ValidationError {
	var errors []ValidationError

	if deploymentName == "" {
		errors = append(errors, ValidationError{
			Field:   "deployment_name",
			Message: "deployment name is required",
			Value:   deploymentName,
		})
		return errors
	}

	if len(deploymentName) > 253 {
		errors = append(errors, ValidationError{
			Field:   "deployment_name",
			Message: "deployment name cannot exceed 253 characters",
			Value:   deploymentName,
		})
	}

	if !v.isValidDNSName(deploymentName) {
		errors = append(errors, ValidationError{
			Field:   "deployment_name",
			Message: "deployment name must be a valid DNS-1123 subdomain",
			Value:   deploymentName,
		})
	}

	return errors
}

// ValidateStatefulSetName validates a Kubernetes StatefulSet name
func (v *Validator) ValidateStatefulSetName(statefulSetName string) []ValidationError {
	var errors []ValidationError

	if statefulSetName == "" {
		errors = append(errors, ValidationError{
			Field:   "statefulset_name",
			Message: "statefulset name is required",
			Value:   statefulSetName,
		})
		return errors
	}

	if len(statefulSetName) > 253 {
		errors = append(errors, ValidationError{
			Field:   "statefulset_name",
			Message: "statefulset name cannot exceed 253 characters",
			Value:   statefulSetName,
		})
	}

	if !v.isValidDNSName(statefulSetName) {
		errors = append(errors, ValidationError{
			Field:   "statefulset_name",
			Message: "statefulset name must be a valid DNS-1123 subdomain",
			Value:   statefulSetName,
		})
	}

	return errors
}

// ValidateReplicas validates replica count
func (v *Validator) ValidateReplicas(replicas *int32) []ValidationError {
	var errors []ValidationError

	if replicas == nil {
		errors = append(errors, ValidationError{
			Field:   "replicas",
			Message: "replicas field is required",
			Value:   nil,
		})
		return errors
	}

	if *replicas < 0 {
		errors = append(errors, ValidationError{
			Field:   "replicas",
			Message: "replicas must be non-negative",
			Value:   *replicas,
		})
	}

	return errors
}

// ValidateUserHeader validates the X-User header
func (v *Validator) ValidateUserHeader(user string) string {
	if user == "" {
		return "system"
	}

	// Basic sanitization
	user = strings.TrimSpace(user)
	if len(user) > 100 {
		user = user[:100]
	}

	return user
}

// isValidDNSName checks if a string is a valid DNS name
func (v *Validator) isValidDNSName(name string) bool {
	if name == "" {
		return false
	}

	// Must start and end with alphanumeric
	if !v.isAlphaNumeric(name[0]) || !v.isAlphaNumeric(name[len(name)-1]) {
		return false
	}

	// Can contain hyphens in the middle
	for i, char := range name {
		if i == 0 || i == len(name)-1 {
			continue
		}
		if !v.isAlphaNumeric(byte(char)) && char != '-' {
			return false
		}
	}

	return true
}

// isAlphaNumeric checks if a byte is alphanumeric (lowercase only for DNS-1123)
func (v *Validator) isAlphaNumeric(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= '0' && b <= '9')
}

// ScaleRequest represents a scaling request
type ScaleRequest struct {
	Replicas *int32 `json:"replicas"`
}

// ValidateScaleRequest validates a scale request
func (v *Validator) ValidateScaleRequest(req *ScaleRequest) []ValidationError {
	if req == nil {
		return []ValidationError{{
			Field:   "request",
			Message: "request body is required",
		}}
	}

	return v.ValidateReplicas(req.Replicas)
}