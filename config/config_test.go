package config

import (
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type ConfigTestSuite struct {
	suite.Suite
	originalEnv map[string]string
}

func (s *ConfigTestSuite) SetupTest() {
	// Store original environment variables
	s.originalEnv = make(map[string]string)
	envVars := []string{
		"PORT", "DATABASE_URL", "KUBECONFIG", "IN_CLUSTER",
		"LOG_LEVEL", "ENABLE_CORS", "ALLOWED_ORIGINS", "RATE_LIMIT",
	}

	for _, envVar := range envVars {
		if val, exists := os.LookupEnv(envVar); exists {
			s.originalEnv[envVar] = val
		}
		os.Unsetenv(envVar)
	}

	// Reset global config
	AppConfig = nil
}

func (s *ConfigTestSuite) TearDownTest() {
	// Restore original environment variables
	for key, value := range s.originalEnv {
		os.Setenv(key, value)
	}

	// Clear any variables that weren't originally set
	envVars := []string{
		"PORT", "DATABASE_URL", "KUBECONFIG", "IN_CLUSTER",
		"LOG_LEVEL", "ENABLE_CORS", "ALLOWED_ORIGINS", "RATE_LIMIT",
	}

	for _, envVar := range envVars {
		if _, exists := s.originalEnv[envVar]; !exists {
			os.Unsetenv(envVar)
		}
	}
}

func (s *ConfigTestSuite) TestLoad_DefaultValues() {
	err := Load()
	s.NoError(err)
	s.NotNil(AppConfig)

	s.Equal("8080", AppConfig.Port)
	s.Equal("postgres://postgres:postgres@localhost:5432/k8s_pod_manager?sslmode=disable", AppConfig.DatabaseURL)
	// KubeConfig will be set to default home path when not in-cluster and not specified
	home, _ := os.UserHomeDir()
	expectedKubeConfig := fmt.Sprintf("%s/.kube/config", home)
	s.Equal(expectedKubeConfig, AppConfig.KubeConfig)
	s.False(AppConfig.InCluster)
	s.Equal("info", AppConfig.LogLevel)
	s.True(AppConfig.EnableCORS)
	s.Equal([]string{"*"}, AppConfig.AllowedOrigins)
	s.Equal(100, AppConfig.RateLimit)
}

func (s *ConfigTestSuite) TestLoad_CustomEnvironmentVariables() {
	// Set custom environment variables
	os.Setenv("PORT", "9090")
	os.Setenv("DATABASE_URL", "postgres://custom:password@localhost:5432/custom_db")
	os.Setenv("KUBECONFIG", "/custom/kube/config")
	os.Setenv("IN_CLUSTER", "true")
	os.Setenv("LOG_LEVEL", "debug")
	os.Setenv("ENABLE_CORS", "true")
	os.Setenv("ALLOWED_ORIGINS", "https://example.com")
	os.Setenv("RATE_LIMIT", "50")

	err := Load()
	s.NoError(err)
	s.NotNil(AppConfig)

	s.Equal("9090", AppConfig.Port)
	s.Equal("postgres://custom:password@localhost:5432/custom_db", AppConfig.DatabaseURL)
	s.Equal("/custom/kube/config", AppConfig.KubeConfig)
	s.True(AppConfig.InCluster)
	s.Equal("debug", AppConfig.LogLevel)
	s.True(AppConfig.EnableCORS)
	s.Equal([]string{"https://example.com"}, AppConfig.AllowedOrigins)
	s.Equal(50, AppConfig.RateLimit)
}

func (s *ConfigTestSuite) TestLoad_KubeConfigDefaultPath() {
	home, _ := os.UserHomeDir()
	expectedPath := home + "/.kube/config"

	err := Load()
	s.NoError(err)
	s.Equal(expectedPath, AppConfig.KubeConfig)
}

func (s *ConfigTestSuite) TestLoad_InClusterKubeConfig() {
	os.Setenv("IN_CLUSTER", "true")

	err := Load()
	s.NoError(err)
	s.True(AppConfig.InCluster)
	s.Equal("", AppConfig.KubeConfig) // Should remain empty when in-cluster
}

func (s *ConfigTestSuite) TestLoad_CORSWildcardOrigins() {
	os.Setenv("ENABLE_CORS", "true")
	os.Setenv("ALLOWED_ORIGINS", "*")

	err := Load()
	s.NoError(err)
	s.True(AppConfig.EnableCORS)
	s.Equal([]string{"*"}, AppConfig.AllowedOrigins)
}

func (s *ConfigTestSuite) TestLoad_CORSSpecificOrigins() {
	os.Setenv("ENABLE_CORS", "true")
	os.Setenv("ALLOWED_ORIGINS", "https://example.com")

	err := Load()
	s.NoError(err)
	s.True(AppConfig.EnableCORS)
	s.Equal([]string{"https://example.com"}, AppConfig.AllowedOrigins)
}

func (s *ConfigTestSuite) TestLoad_CORSDisabled() {
	os.Setenv("ENABLE_CORS", "false")

	err := Load()
	s.NoError(err)
	s.False(AppConfig.EnableCORS)
	s.Nil(AppConfig.AllowedOrigins)
}

func TestConfigSuite(t *testing.T) {
	suite.Run(t, new(ConfigTestSuite))
}

// Table-driven tests for helper functions
func TestGetEnv(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		defaultValue string
		envValue     string
		setEnv       bool
		expected     string
	}{
		{
			name:         "environment variable exists",
			key:          "TEST_VAR",
			defaultValue: "default",
			envValue:     "custom",
			setEnv:       true,
			expected:     "custom",
		},
		{
			name:         "environment variable does not exist",
			key:          "TEST_VAR",
			defaultValue: "default",
			envValue:     "",
			setEnv:       false,
			expected:     "default",
		},
		{
			name:         "environment variable is empty string",
			key:          "TEST_VAR",
			defaultValue: "default",
			envValue:     "",
			setEnv:       true,
			expected:     "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clean up environment
			os.Unsetenv(tt.key)

			if tt.setEnv {
				os.Setenv(tt.key, tt.envValue)
				defer os.Unsetenv(tt.key)
			}

			result := getEnv(tt.key, tt.defaultValue)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetEnvBool(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		defaultValue bool
		envValue     string
		setEnv       bool
		expected     bool
	}{
		{
			name:         "valid true value",
			key:          "TEST_BOOL",
			defaultValue: false,
			envValue:     "true",
			setEnv:       true,
			expected:     true,
		},
		{
			name:         "valid false value",
			key:          "TEST_BOOL",
			defaultValue: true,
			envValue:     "false",
			setEnv:       true,
			expected:     false,
		},
		{
			name:         "valid 1 value",
			key:          "TEST_BOOL",
			defaultValue: false,
			envValue:     "1",
			setEnv:       true,
			expected:     true,
		},
		{
			name:         "valid 0 value",
			key:          "TEST_BOOL",
			defaultValue: true,
			envValue:     "0",
			setEnv:       true,
			expected:     false,
		},
		{
			name:         "invalid value returns default",
			key:          "TEST_BOOL",
			defaultValue: true,
			envValue:     "invalid",
			setEnv:       true,
			expected:     true,
		},
		{
			name:         "empty value returns default",
			key:          "TEST_BOOL",
			defaultValue: true,
			envValue:     "",
			setEnv:       true,
			expected:     true,
		},
		{
			name:         "variable not set returns default",
			key:          "TEST_BOOL",
			defaultValue: false,
			envValue:     "",
			setEnv:       false,
			expected:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clean up environment
			os.Unsetenv(tt.key)

			if tt.setEnv {
				os.Setenv(tt.key, tt.envValue)
				defer os.Unsetenv(tt.key)
			}

			result := getEnvBool(tt.key, tt.defaultValue)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetEnvInt(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		defaultValue int
		envValue     string
		setEnv       bool
		expected     int
	}{
		{
			name:         "valid positive integer",
			key:          "TEST_INT",
			defaultValue: 100,
			envValue:     "200",
			setEnv:       true,
			expected:     200,
		},
		{
			name:         "valid zero value",
			key:          "TEST_INT",
			defaultValue: 100,
			envValue:     "0",
			setEnv:       true,
			expected:     0,
		},
		{
			name:         "valid negative integer",
			key:          "TEST_INT",
			defaultValue: 100,
			envValue:     "-50",
			setEnv:       true,
			expected:     -50,
		},
		{
			name:         "invalid value returns default",
			key:          "TEST_INT",
			defaultValue: 100,
			envValue:     "invalid",
			setEnv:       true,
			expected:     100,
		},
		{
			name:         "empty value returns default",
			key:          "TEST_INT",
			defaultValue: 100,
			envValue:     "",
			setEnv:       true,
			expected:     100,
		},
		{
			name:         "variable not set returns default",
			key:          "TEST_INT",
			defaultValue: 50,
			envValue:     "",
			setEnv:       false,
			expected:     50,
		},
		{
			name:         "decimal value returns default",
			key:          "TEST_INT",
			defaultValue: 100,
			envValue:     "50.5",
			setEnv:       true,
			expected:     100,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clean up environment
			os.Unsetenv(tt.key)

			if tt.setEnv {
				os.Setenv(tt.key, tt.envValue)
				defer os.Unsetenv(tt.key)
			}

			result := getEnvInt(tt.key, tt.defaultValue)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// Test edge cases and boundary conditions
func TestConfigEdgeCases(t *testing.T) {
	t.Run("multiple calls to Load preserve config", func(t *testing.T) {
		os.Setenv("PORT", "8888")
		defer os.Unsetenv("PORT")

		err1 := Load()
		assert.NoError(t, err1)
		firstConfig := AppConfig

		err2 := Load()
		assert.NoError(t, err2)

		// Should be a new config object but with same values
		assert.NotSame(t, firstConfig, AppConfig)
		assert.Equal(t, "8888", AppConfig.Port)
	})

	t.Run("global AppConfig is accessible", func(t *testing.T) {
		err := Load()
		assert.NoError(t, err)
		assert.NotNil(t, AppConfig)
		assert.IsType(t, &Config{}, AppConfig)
	})
}

// Benchmarks for performance testing
func BenchmarkLoad(b *testing.B) {
	for i := 0; i < b.N; i++ {
		Load()
	}
}

func BenchmarkGetEnv(b *testing.B) {
	os.Setenv("BENCH_VAR", "value")
	defer os.Unsetenv("BENCH_VAR")

	for i := 0; i < b.N; i++ {
		getEnv("BENCH_VAR", "default")
	}
}

func BenchmarkGetEnvBool(b *testing.B) {
	os.Setenv("BENCH_BOOL", "true")
	defer os.Unsetenv("BENCH_BOOL")

	for i := 0; i < b.N; i++ {
		getEnvBool("BENCH_BOOL", false)
	}
}

func BenchmarkGetEnvInt(b *testing.B) {
	os.Setenv("BENCH_INT", "100")
	defer os.Unsetenv("BENCH_INT")

	for i := 0; i < b.N; i++ {
		getEnvInt("BENCH_INT", 50)
	}
}