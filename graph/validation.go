package graph

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/carlosf/k8s-pod-manager/graph/model"
)

// DefaultInputValidator implements InputValidator interface
// Provides comprehensive validation for GraphQL inputs following Kubernetes naming conventions
type DefaultInputValidator struct{}

// NewDefaultInputValidator creates a new instance of DefaultInputValidator
func NewDefaultInputValidator() *DefaultInputValidator {
	return &DefaultInputValidator{}
}

// Kubernetes DNS-1123 subdomain name regex pattern
var (
	// Kubernetes resource names must be DNS-1123 compliant
	kubernetesNameRegex = regexp.MustCompile(`^[a-z0-9]([a-z0-9\-]*[a-z0-9])?$`)

	// Maximum length for Kubernetes resource names
	maxKubernetesNameLength = 253

	// Maximum length for short resource names (like pod names)
	maxShortNameLength = 63
)

// ValidateNamespace validates a Kubernetes namespace name
func (v *DefaultInputValidator) ValidateNamespace(namespace string) error {
	if namespace == "" {
		return fmt.Errorf("namespace cannot be empty")
	}

	if len(namespace) > maxShortNameLength {
		return fmt.Errorf("namespace name too long: %d characters (max %d)", len(namespace), maxShortNameLength)
	}

	if !kubernetesNameRegex.MatchString(namespace) {
		return fmt.Errorf("invalid namespace name '%s': must be a valid DNS-1123 subdomain", namespace)
	}

	// Reserved namespace validation
	reservedNamespaces := []string{"kube-system", "kube-public", "kube-node-lease"}
	for _, reserved := range reservedNamespaces {
		if namespace == reserved {
			return fmt.Errorf("cannot perform operations on reserved namespace '%s'", namespace)
		}
	}

	return nil
}

// ValidatePodName validates a Kubernetes pod name
func (v *DefaultInputValidator) ValidatePodName(name string) error {
	if name == "" {
		return fmt.Errorf("pod name cannot be empty")
	}

	if len(name) > maxShortNameLength {
		return fmt.Errorf("pod name too long: %d characters (max %d)", len(name), maxShortNameLength)
	}

	if !kubernetesNameRegex.MatchString(name) {
		return fmt.Errorf("invalid pod name '%s': must be a valid DNS-1123 subdomain", name)
	}

	// Check for potentially dangerous pod names
	dangerousPrefixes := []string{"kube-", "system-"}
	for _, prefix := range dangerousPrefixes {
		if strings.HasPrefix(name, prefix) {
			return fmt.Errorf("pod name '%s' starts with reserved prefix '%s'", name, prefix)
		}
	}

	return nil
}

// ValidateScaleInput validates scaling operation input
func (v *DefaultInputValidator) ValidateScaleInput(input model.ScaleInput) error {
	if input.Replicas < 0 {
		return fmt.Errorf("replica count cannot be negative: %d", input.Replicas)
	}

	if input.Replicas > 1000 {
		return fmt.Errorf("replica count too high: %d (max 1000 for safety)", input.Replicas)
	}

	return nil
}

// ValidateUserInput validates user input for audit logging
func (v *DefaultInputValidator) ValidateUserInput(user *string) error {
	if user == nil {
		// User is optional, system will default to "system"
		return nil
	}

	if *user == "" {
		return fmt.Errorf("user cannot be empty string when provided")
	}

	if len(*user) > 100 {
		return fmt.Errorf("user name too long: %d characters (max 100)", len(*user))
	}

	// Basic validation to prevent injection attacks
	if strings.ContainsAny(*user, "<>\"'&") {
		return fmt.Errorf("user name contains invalid characters")
	}

	return nil
}

// CompositeValidator allows combining multiple validators (Composite Pattern)
type CompositeValidator struct {
	validators []InputValidator
}

// NewCompositeValidator creates a new composite validator
func NewCompositeValidator(validators ...InputValidator) *CompositeValidator {
	return &CompositeValidator{
		validators: validators,
	}
}

// ValidateNamespace runs validation through all composed validators
func (c *CompositeValidator) ValidateNamespace(namespace string) error {
	for _, validator := range c.validators {
		if err := validator.ValidateNamespace(namespace); err != nil {
			return err
		}
	}
	return nil
}

// ValidatePodName runs validation through all composed validators
func (c *CompositeValidator) ValidatePodName(name string) error {
	for _, validator := range c.validators {
		if err := validator.ValidatePodName(name); err != nil {
			return err
		}
	}
	return nil
}

// ValidateScaleInput runs validation through all composed validators
func (c *CompositeValidator) ValidateScaleInput(input model.ScaleInput) error {
	for _, validator := range c.validators {
		if err := validator.ValidateScaleInput(input); err != nil {
			return err
		}
	}
	return nil
}

// ValidateUserInput runs validation through all composed validators
func (c *CompositeValidator) ValidateUserInput(user *string) error {
	for _, validator := range c.validators {
		if err := validator.ValidateUserInput(user); err != nil {
			return err
		}
	}
	return nil
}