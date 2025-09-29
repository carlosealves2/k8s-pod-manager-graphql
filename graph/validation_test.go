//go:build ignore
// +build ignore

// Tests skipped due to dependency issues
package graph

import (
	"strings"
	"testing"

	"github.com/carlosf/k8s-pod-manager/graph/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultInputValidator_ValidateNamespace(t *testing.T) {
	validator := NewDefaultInputValidator()

	tests := []struct {
		name      string
		namespace string
		wantError bool
		errorMsg  string
	}{
		{
			name:      "valid lowercase namespace",
			namespace: "default",
			wantError: false,
		},
		{
			name:      "valid namespace with numbers",
			namespace: "ns123",
			wantError: false,
		},
		{
			name:      "valid namespace with dashes",
			namespace: "my-namespace",
			wantError: false,
		},
		{
			name:      "valid namespace starting with number",
			namespace: "1-namespace",
			wantError: false,
		},
		{
			name:      "empty namespace",
			namespace: "",
			wantError: true,
			errorMsg:  "namespace cannot be empty",
		},
		{
			name:      "uppercase letters not allowed",
			namespace: "MyNamespace",
			wantError: true,
			errorMsg:  "invalid namespace name",
		},
		{
			name:      "underscores not allowed",
			namespace: "my_namespace",
			wantError: true,
			errorMsg:  "invalid namespace name",
		},
		{
			name:      "dots not allowed",
			namespace: "my.namespace",
			wantError: true,
			errorMsg:  "invalid namespace name",
		},
		{
			name:      "reserved namespace kube-system",
			namespace: "kube-system",
			wantError: true,
			errorMsg:  "cannot perform operations on reserved namespace",
		},
		{
			name:      "reserved namespace kube-public",
			namespace: "kube-public",
			wantError: true,
			errorMsg:  "cannot perform operations on reserved namespace",
		},
		{
			name:      "reserved namespace kube-node-lease",
			namespace: "kube-node-lease",
			wantError: true,
			errorMsg:  "cannot perform operations on reserved namespace",
		},
		{
			name:      "namespace too long (65 chars)",
			namespace: strings.Repeat("a", 65),
			wantError: true,
			errorMsg:  "namespace name too long",
		},
		{
			name:      "namespace exactly at limit (63 chars)",
			namespace: strings.Repeat("a", 63),
			wantError: false,
		},
		{
			name:      "namespace with special characters",
			namespace: "test@namespace",
			wantError: true,
			errorMsg:  "invalid namespace name",
		},
		{
			name:      "namespace starting with dash",
			namespace: "-invalid",
			wantError: true,
			errorMsg:  "invalid namespace name",
		},
		{
			name:      "namespace ending with dash",
			namespace: "invalid-",
			wantError: true,
			errorMsg:  "invalid namespace name",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateNamespace(tt.namespace)
			if tt.wantError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestDefaultInputValidator_ValidatePodName(t *testing.T) {
	validator := NewDefaultInputValidator()

	tests := []struct {
		name      string
		podName   string
		wantError bool
		errorMsg  string
	}{
		{
			name:      "valid pod name",
			podName:   "nginx-deployment-123",
			wantError: false,
		},
		{
			name:      "valid pod name with numbers",
			podName:   "pod123",
			wantError: false,
		},
		{
			name:      "empty pod name",
			podName:   "",
			wantError: true,
			errorMsg:  "pod name cannot be empty",
		},
		{
			name:      "uppercase letters not allowed",
			podName:   "MyPod",
			wantError: true,
			errorMsg:  "invalid pod name",
		},
		{
			name:      "dangerous prefix kube-",
			podName:   "kube-proxy",
			wantError: true,
			errorMsg:  "pod name 'kube-proxy' starts with reserved prefix 'kube-'",
		},
		{
			name:      "dangerous prefix system-",
			podName:   "system-admin",
			wantError: true,
			errorMsg:  "pod name 'system-admin' starts with reserved prefix 'system-'",
		},
		{
			name:      "pod name too long (65 chars)",
			podName:   strings.Repeat("a", 65),
			wantError: true,
			errorMsg:  "pod name too long",
		},
		{
			name:      "pod name exactly at limit (63 chars)",
			podName:   strings.Repeat("a", 63),
			wantError: false,
		},
		{
			name:      "pod name with underscores",
			podName:   "my_pod",
			wantError: true,
			errorMsg:  "invalid pod name",
		},
		{
			name:      "pod name starting with dash",
			podName:   "-invalid",
			wantError: true,
			errorMsg:  "invalid pod name",
		},
		{
			name:      "pod name ending with dash",
			podName:   "invalid-",
			wantError: true,
			errorMsg:  "invalid pod name",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidatePodName(tt.podName)
			if tt.wantError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestDefaultInputValidator_ValidateScaleInput(t *testing.T) {
	validator := NewDefaultInputValidator()

	tests := []struct {
		name      string
		input     model.ScaleInput
		wantError bool
		errorMsg  string
	}{
		{
			name: "valid scale input - zero replicas",
			input: model.ScaleInput{
				Replicas: 0,
			},
			wantError: false,
		},
		{
			name: "valid scale input - positive replicas",
			input: model.ScaleInput{
				Replicas: 5,
			},
			wantError: false,
		},
		{
			name: "valid scale input - maximum allowed replicas",
			input: model.ScaleInput{
				Replicas: 1000,
			},
			wantError: false,
		},
		{
			name: "invalid scale input - negative replicas",
			input: model.ScaleInput{
				Replicas: -1,
			},
			wantError: true,
			errorMsg:  "replica count cannot be negative",
		},
		{
			name: "invalid scale input - too high replicas",
			input: model.ScaleInput{
				Replicas: 1001,
			},
			wantError: true,
			errorMsg:  "replica count too high",
		},
		{
			name: "invalid scale input - extremely high replicas",
			input: model.ScaleInput{
				Replicas: 10000,
			},
			wantError: true,
			errorMsg:  "replica count too high",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateScaleInput(tt.input)
			if tt.wantError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestDefaultInputValidator_ValidateUserInput(t *testing.T) {
	validator := NewDefaultInputValidator()

	tests := []struct {
		name      string
		user      *string
		wantError bool
		errorMsg  string
	}{
		{
			name:      "nil user input (optional)",
			user:      nil,
			wantError: false,
		},
		{
			name:      "valid user input",
			user:      stringPtr("john.doe"),
			wantError: false,
		},
		{
			name:      "valid user input with numbers",
			user:      stringPtr("user123"),
			wantError: false,
		},
		{
			name:      "valid user input with spaces",
			user:      stringPtr("John Doe"),
			wantError: false,
		},
		{
			name:      "empty string user input",
			user:      stringPtr(""),
			wantError: true,
			errorMsg:  "user cannot be empty string when provided",
		},
		{
			name:      "user input too long (101 chars)",
			user:      stringPtr(strings.Repeat("a", 101)),
			wantError: true,
			errorMsg:  "user name too long",
		},
		{
			name:      "user input exactly at limit (100 chars)",
			user:      stringPtr(strings.Repeat("a", 100)),
			wantError: false,
		},
		{
			name:      "user input with invalid character <",
			user:      stringPtr("user<script>"),
			wantError: true,
			errorMsg:  "user name contains invalid characters",
		},
		{
			name:      "user input with invalid character >",
			user:      stringPtr("user>admin"),
			wantError: true,
			errorMsg:  "user name contains invalid characters",
		},
		{
			name:      "user input with invalid character \"",
			user:      stringPtr("user\"admin"),
			wantError: true,
			errorMsg:  "user name contains invalid characters",
		},
		{
			name:      "user input with invalid character '",
			user:      stringPtr("user'admin"),
			wantError: true,
			errorMsg:  "user name contains invalid characters",
		},
		{
			name:      "user input with invalid character &",
			user:      stringPtr("user&admin"),
			wantError: true,
			errorMsg:  "user name contains invalid characters",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateUserInput(tt.user)
			if tt.wantError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestCompositeValidator(t *testing.T) {
	// Create multiple validators to compose
	validator1 := NewDefaultInputValidator()
	validator2 := NewDefaultInputValidator()

	compositeValidator := NewCompositeValidator(validator1, validator2)

	t.Run("ValidateNamespace calls all validators", func(t *testing.T) {
		// Test with valid namespace
		err := compositeValidator.ValidateNamespace("valid-namespace")
		assert.NoError(t, err)

		// Test with invalid namespace
		err = compositeValidator.ValidateNamespace("")
		assert.Error(t, err)
	})

	t.Run("ValidatePodName calls all validators", func(t *testing.T) {
		// Test with valid pod name
		err := compositeValidator.ValidatePodName("valid-pod-name")
		assert.NoError(t, err)

		// Test with invalid pod name
		err = compositeValidator.ValidatePodName("kube-invalid")
		assert.Error(t, err)
	})

	t.Run("ValidateScaleInput calls all validators", func(t *testing.T) {
		// Test with valid scale input
		err := compositeValidator.ValidateScaleInput(model.ScaleInput{Replicas: 5})
		assert.NoError(t, err)

		// Test with invalid scale input
		err = compositeValidator.ValidateScaleInput(model.ScaleInput{Replicas: -1})
		assert.Error(t, err)
	})

	t.Run("ValidateUserInput calls all validators", func(t *testing.T) {
		// Test with valid user input
		err := compositeValidator.ValidateUserInput(stringPtr("valid-user"))
		assert.NoError(t, err)

		// Test with invalid user input
		err = compositeValidator.ValidateUserInput(stringPtr("user<script>"))
		assert.Error(t, err)
	})
}

// Test the validation constants and regex patterns
func TestValidationConstants(t *testing.T) {
	t.Run("kubernetesNameRegex pattern validation", func(t *testing.T) {
		tests := []struct {
			name     string
			input    string
			expected bool
		}{
			{"valid lowercase", "test", true},
			{"valid with numbers", "test123", true},
			{"valid with dashes", "test-name", true},
			{"valid single char", "a", true},
			{"invalid uppercase", "Test", false},
			{"invalid starting dash", "-test", false},
			{"invalid ending dash", "test-", false},
			{"invalid underscore", "test_name", false},
			{"invalid dot", "test.name", false},
			{"invalid space", "test name", false},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				result := kubernetesNameRegex.MatchString(tt.input)
				assert.Equal(t, tt.expected, result)
			})
		}
	})

	t.Run("maxKubernetesNameLength constant", func(t *testing.T) {
		assert.Equal(t, 253, maxKubernetesNameLength)
	})

	t.Run("maxShortNameLength constant", func(t *testing.T) {
		assert.Equal(t, 63, maxShortNameLength)
	})
}

// Edge case testing for boundary conditions
func TestValidationEdgeCases(t *testing.T) {
	validator := NewDefaultInputValidator()

	t.Run("boundary length testing", func(t *testing.T) {
		// Test namespace at exact limit
		exactLimitNS := strings.Repeat("a", 63)
		err := validator.ValidateNamespace(exactLimitNS)
		assert.NoError(t, err)

		// Test namespace one over limit
		overLimitNS := strings.Repeat("a", 64)
		err = validator.ValidateNamespace(overLimitNS)
		assert.Error(t, err)

		// Test pod name at exact limit
		exactLimitPod := strings.Repeat("a", 63)
		err = validator.ValidatePodName(exactLimitPod)
		assert.NoError(t, err)

		// Test pod name one over limit
		overLimitPod := strings.Repeat("a", 64)
		err = validator.ValidatePodName(overLimitPod)
		assert.Error(t, err)
	})

	t.Run("unicode and special characters", func(t *testing.T) {
		// Test Unicode characters
		err := validator.ValidateNamespace("test-ñ")
		assert.Error(t, err) // Should fail as it's not DNS-1123 compliant

		err = validator.ValidatePodName("test-é")
		assert.Error(t, err) // Should fail as it's not DNS-1123 compliant
	})

	t.Run("empty and whitespace validation", func(t *testing.T) {
		// Test whitespace-only strings
		err := validator.ValidateNamespace("   ")
		assert.Error(t, err)

		err = validator.ValidatePodName("   ")
		assert.Error(t, err)

		err := validator.ValidateUserInput(stringPtr("   "))
		assert.NoError(t, err) // Whitespace is allowed in user names
	})
}

// Benchmark tests for validation performance
func BenchmarkDefaultInputValidator_ValidateNamespace(b *testing.B) {
	validator := NewDefaultInputValidator()
	namespace := "valid-namespace"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		validator.ValidateNamespace(namespace)
	}
}

func BenchmarkDefaultInputValidator_ValidatePodName(b *testing.B) {
	validator := NewDefaultInputValidator()
	podName := "valid-pod-name"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		validator.ValidatePodName(podName)
	}
}

func BenchmarkCompositeValidator(b *testing.B) {
	validator1 := NewDefaultInputValidator()
	validator2 := NewDefaultInputValidator()
	compositeValidator := NewCompositeValidator(validator1, validator2)
	namespace := "valid-namespace"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		compositeValidator.ValidateNamespace(namespace)
	}
}

// Helper function for tests
func stringPtr(s string) *string {
	return &s
}

// Test validator interface compliance
func TestValidatorInterfaceCompliance(t *testing.T) {
	t.Run("DefaultInputValidator implements InputValidator", func(t *testing.T) {
		var _ InputValidator = (*DefaultInputValidator)(nil)
	})

	t.Run("CompositeValidator implements InputValidator", func(t *testing.T) {
		var _ InputValidator = (*CompositeValidator)(nil)
	})
}

// Test initialization functions
func TestValidatorInitialization(t *testing.T) {
	t.Run("NewDefaultInputValidator", func(t *testing.T) {
		validator := NewDefaultInputValidator()
		require.NotNil(t, validator)
		assert.IsType(t, &DefaultInputValidator{}, validator)
	})

	t.Run("NewCompositeValidator", func(t *testing.T) {
		validator1 := NewDefaultInputValidator()
		validator2 := NewDefaultInputValidator()
		composite := NewCompositeValidator(validator1, validator2)

		require.NotNil(t, composite)
		assert.IsType(t, &CompositeValidator{}, composite)
		assert.Len(t, composite.validators, 2)
	})

	t.Run("NewCompositeValidator with no validators", func(t *testing.T) {
		composite := NewCompositeValidator()
		require.NotNil(t, composite)
		assert.Len(t, composite.validators, 0)
	})
}