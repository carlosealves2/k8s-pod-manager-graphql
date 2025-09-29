package kubernetes

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/carlosf/k8s-pod-manager/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/util/homedir"
)

type KubernetesClientTestSuite struct {
	suite.Suite
	originalClient *kubernetes.Clientset
	originalConfig *config.Config
	tempKubeConfig string
}

func (s *KubernetesClientTestSuite) SetupTest() {
	// Store original values
	s.originalClient = Client
	s.originalConfig = config.AppConfig

	// Reset global client
	Client = nil

	// Create a temporary kubeconfig file for testing
	s.createTempKubeConfig()
}

func (s *KubernetesClientTestSuite) TearDownTest() {
	// Restore original values
	Client = s.originalClient
	config.AppConfig = s.originalConfig

	// Clean up temporary file
	if s.tempKubeConfig != "" {
		os.Remove(s.tempKubeConfig)
	}
}

func (s *KubernetesClientTestSuite) createTempKubeConfig() {
	// Create a minimal valid kubeconfig for testing
	kubeConfigContent := `
apiVersion: v1
kind: Config
clusters:
- cluster:
    server: https://localhost:6443
    insecure-skip-tls-verify: true
  name: test-cluster
contexts:
- context:
    cluster: test-cluster
    user: test-user
  name: test-context
current-context: test-context
users:
- name: test-user
  user:
    token: test-token
`

	tmpDir := os.TempDir()
	tmpFile, err := os.CreateTemp(tmpDir, "kubeconfig-test-*.yaml")
	s.Require().NoError(err)

	_, err = tmpFile.WriteString(kubeConfigContent)
	s.Require().NoError(err)

	tmpFile.Close()
	s.tempKubeConfig = tmpFile.Name()
}

func (s *KubernetesClientTestSuite) TestInitClient_InCluster() {
	config.AppConfig = &config.Config{
		InCluster: true,
	}

	// This will fail in test environment since we're not in a pod
	// but we can test that the function handles the in-cluster config path
	err := InitClient()
	s.Error(err)
	s.Contains(err.Error(), "failed to create in-cluster config")
}

func (s *KubernetesClientTestSuite) TestInitClient_WithKubeConfig() {
	config.AppConfig = &config.Config{
		InCluster:  false,
		KubeConfig: s.tempKubeConfig,
	}

	err := InitClient()
	// This may fail due to invalid server in test kubeconfig, but should not fail on file reading
	if err != nil {
		s.Contains(err.Error(), "failed to create kubernetes client")
	}
}

func (s *KubernetesClientTestSuite) TestInitClient_WithEnvKubeConfig() {
	// Set KUBECONFIG environment variable
	os.Setenv("KUBECONFIG", s.tempKubeConfig)
	defer os.Unsetenv("KUBECONFIG")

	config.AppConfig = &config.Config{
		InCluster:  false,
		KubeConfig: "", // Empty, should use env var
	}

	err := InitClient()
	// This may fail due to invalid server in test kubeconfig
	if err != nil {
		s.Contains(err.Error(), "failed to create kubernetes client")
	}
}

func (s *KubernetesClientTestSuite) TestInitClient_WithHomeKubeConfig() {
	// Clear KUBECONFIG env var
	os.Unsetenv("KUBECONFIG")

	// Create kubeconfig in home directory
	home := homedir.HomeDir()
	s.Require().NotEmpty(home)

	kubeDir := filepath.Join(home, ".kube")
	kubeconfigPath := filepath.Join(kubeDir, "config")

	// Create .kube directory if it doesn't exist
	os.MkdirAll(kubeDir, 0755)

	// Check if original kubeconfig exists and back it up
	originalExists := false
	var originalContent []byte
	if _, err := os.Stat(kubeconfigPath); err == nil {
		originalExists = true
		originalContent, _ = os.ReadFile(kubeconfigPath)
	}

	// Create test kubeconfig
	kubeConfigContent := `
apiVersion: v1
kind: Config
clusters:
- cluster:
    server: https://localhost:6443
    insecure-skip-tls-verify: true
  name: test-cluster
contexts:
- context:
    cluster: test-cluster
    user: test-user
  name: test-context
current-context: test-context
users:
- name: test-user
  user:
    token: test-token
`

	err := os.WriteFile(kubeconfigPath, []byte(kubeConfigContent), 0644)
	s.Require().NoError(err)

	// Clean up after test
	defer func() {
		if originalExists {
			os.WriteFile(kubeconfigPath, originalContent, 0644)
		} else {
			os.Remove(kubeconfigPath)
		}
	}()

	config.AppConfig = &config.Config{
		InCluster:  false,
		KubeConfig: "", // Empty, should use home directory
	}

	err = InitClient()
	// This may fail due to invalid server in test kubeconfig
	if err != nil {
		s.Contains(err.Error(), "failed to create kubernetes client")
	}
}

func (s *KubernetesClientTestSuite) TestInitClient_NoKubeConfig() {
	// Clear environment
	os.Unsetenv("KUBECONFIG")

	config.AppConfig = &config.Config{
		InCluster:  false,
		KubeConfig: "", // Empty
	}

	// Remove home kubeconfig if it exists temporarily
	home := homedir.HomeDir()
	kubeconfigPath := filepath.Join(home, ".kube", "config")

	var originalContent []byte
	originalExists := false
	if _, err := os.Stat(kubeconfigPath); err == nil {
		originalExists = true
		originalContent, _ = os.ReadFile(kubeconfigPath)
		os.Remove(kubeconfigPath)
	}

	defer func() {
		if originalExists {
			os.WriteFile(kubeconfigPath, originalContent, 0644)
		}
	}()

	err := InitClient()
	s.Error(err)
	// Could be either "kubeconfig not found" or "kubeconfig file does not exist"
	s.True(Contains(err.Error(), "kubeconfig not found") || Contains(err.Error(), "kubeconfig file does not exist"))
}

func (s *KubernetesClientTestSuite) TestInitClient_InvalidKubeConfigFile() {
	config.AppConfig = &config.Config{
		InCluster:  false,
		KubeConfig: "/nonexistent/path/config",
	}

	err := InitClient()
	s.Error(err)
	s.Contains(err.Error(), "kubeconfig file does not exist")
}

func (s *KubernetesClientTestSuite) TestInitClient_MalformedKubeConfig() {
	// Create malformed kubeconfig
	tmpFile, err := os.CreateTemp("", "kubeconfig-malformed-*.yaml")
	s.Require().NoError(err)
	defer os.Remove(tmpFile.Name())

	_, err = tmpFile.WriteString("invalid yaml content {[}")
	s.Require().NoError(err)
	tmpFile.Close()

	config.AppConfig = &config.Config{
		InCluster:  false,
		KubeConfig: tmpFile.Name(),
	}

	err = InitClient()
	s.Error(err)
	s.Contains(err.Error(), "failed to build config from kubeconfig")
}

func TestKubernetesClientSuite(t *testing.T) {
	suite.Run(t, new(KubernetesClientTestSuite))
}

// Unit tests
func TestInitClient_NilConfig(t *testing.T) {
	originalConfig := config.AppConfig
	defer func() { config.AppConfig = originalConfig }()

	config.AppConfig = nil

	assert.Panics(t, func() {
		InitClient()
	})
}

func TestGlobalClientVariable(t *testing.T) {
	originalClient := Client

	// Test that Client can be set to nil
	Client = nil
	assert.Nil(t, Client)

	// Restore original client
	Client = originalClient
}

// Test kubeconfig path resolution
func TestKubeConfigPathResolution(t *testing.T) {
	tests := []struct {
		name           string
		inCluster      bool
		kubeConfig     string
		kubeconfigEnv  string
		expectedError  bool
		errorContains  string
	}{
		{
			name:          "in-cluster config",
			inCluster:     true,
			kubeConfig:    "",
			kubeconfigEnv: "",
			expectedError: true,
			errorContains: "failed to create in-cluster config",
		},
		{
			name:          "explicit kubeconfig path - nonexistent",
			inCluster:     false,
			kubeConfig:    "/nonexistent/config",
			kubeconfigEnv: "",
			expectedError: true,
			errorContains: "kubeconfig file does not exist",
		},
		{
			name:          "env kubeconfig - nonexistent",
			inCluster:     false,
			kubeConfig:    "",
			kubeconfigEnv: "/nonexistent/config",
			expectedError: true,
			errorContains: "kubeconfig file does not exist",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			originalConfig := config.AppConfig
			originalClient := Client
			defer func() {
				config.AppConfig = originalConfig
				Client = originalClient
			}()

			// Set environment variable if specified
			if tt.kubeconfigEnv != "" {
				os.Setenv("KUBECONFIG", tt.kubeconfigEnv)
				defer os.Unsetenv("KUBECONFIG")
			} else {
				os.Unsetenv("KUBECONFIG")
			}

			config.AppConfig = &config.Config{
				InCluster:  tt.inCluster,
				KubeConfig: tt.kubeConfig,
			}

			Client = nil

			err := InitClient()

			if tt.expectedError {
				assert.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, Client)
			}
		})
	}
}

// Test home directory resolution
func TestHomeDirectoryResolution(t *testing.T) {
	originalConfig := config.AppConfig
	originalClient := Client
	defer func() {
		config.AppConfig = originalConfig
		Client = originalClient
	}()

	// Clear environment
	os.Unsetenv("KUBECONFIG")

	config.AppConfig = &config.Config{
		InCluster:  false,
		KubeConfig: "",
	}

	home := homedir.HomeDir()
	if home == "" {
		t.Skip("Cannot determine home directory")
	}

	// The function should attempt to use ~/.kube/config
	// Even if it fails to connect, it should not fail on path resolution
	err := InitClient()
	if err != nil {
		// The error should be about file not existing or connection, not path resolution
		assert.True(t,
			err.Error() == "kubeconfig not found" ||
			Contains(err.Error(), "kubeconfig file does not exist") ||
			Contains(err.Error(), "failed to build config from kubeconfig") ||
			Contains(err.Error(), "failed to create kubernetes client"),
			"Error should be about file access or connection, got: %s", err.Error())
	}
}

// Helper function since strings.Contains is not available in test context
func Contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > len(substr) && hasSubstring(s, substr)))
}

func hasSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// Benchmark tests
func BenchmarkInitClient(b *testing.B) {
	originalConfig := config.AppConfig
	originalClient := Client
	defer func() {
		config.AppConfig = originalConfig
		Client = originalClient
	}()

	config.AppConfig = &config.Config{
		InCluster:  false,
		KubeConfig: "/nonexistent/config", // Will fail fast
	}

	for i := 0; i < b.N; i++ {
		Client = nil
		InitClient() // Will fail but we're measuring overhead
	}
}

// Test error handling edge cases
func TestInitClient_EdgeCases(t *testing.T) {
	t.Run("empty home directory", func(t *testing.T) {
		originalConfig := config.AppConfig
		originalClient := Client
		defer func() {
			config.AppConfig = originalConfig
			Client = originalClient
		}()

		// This test is difficult to create reliably since homedir.HomeDir()
		// uses environment variables and OS-specific logic
		// For now, we just ensure the function doesn't panic
		config.AppConfig = &config.Config{
			InCluster:  false,
			KubeConfig: "",
		}

		os.Unsetenv("KUBECONFIG")

		err := InitClient()
		// May or may not error depending on whether ~/.kube/config exists
		if err != nil {
			assert.Contains(t, err.Error(), "kubeconfig")
		}
	})

	t.Run("client assignment", func(t *testing.T) {
		originalClient := Client

		// Test that Client can be set to nil and restored
		Client = nil
		assert.Nil(t, Client)

		// Restore
		Client = originalClient
	})
}

// Test config validation
func TestConfigValidation(t *testing.T) {
	tests := []struct {
		name        string
		config      *config.Config
		shouldPanic bool
	}{
		{
			name: "valid config - in cluster",
			config: &config.Config{
				InCluster: true,
			},
			shouldPanic: false,
		},
		{
			name: "valid config - with kubeconfig",
			config: &config.Config{
				InCluster:  false,
				KubeConfig: "/some/path",
			},
			shouldPanic: false,
		},
		{
			name:        "nil config",
			config:      nil,
			shouldPanic: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			originalConfig := config.AppConfig
			originalClient := Client
			defer func() {
				config.AppConfig = originalConfig
				Client = originalClient
			}()

			config.AppConfig = tt.config
			Client = nil

			if tt.shouldPanic {
				assert.Panics(t, func() {
					InitClient()
				})
			} else {
				// Should not panic, but may error
				InitClient()
				// We don't check for error here since we're testing panic behavior
			}
		})
	}
}