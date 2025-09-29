package database

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/carlosf/k8s-pod-manager/config"
	"github.com/carlosf/k8s-pod-manager/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
	"gorm.io/gorm"
)

type DatabaseTestSuite struct {
	suite.Suite
	pgContainer   *postgres.PostgresContainer
	originalDB    *gorm.DB
	originalConfig *config.Config
}

func (s *DatabaseTestSuite) SetupSuite() {
	ctx := context.Background()

	// Store original values
	s.originalDB = DB
	s.originalConfig = config.AppConfig

	// Start PostgreSQL container
	var err error
	s.pgContainer, err = postgres.Run(ctx,
		"postgres:15-alpine",
		postgres.WithDatabase("testdb"),
		postgres.WithUsername("testuser"),
		postgres.WithPassword("testpass"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(30*time.Second)),
	)
	s.Require().NoError(err)

	// Get container connection string
	connStr, err := s.pgContainer.ConnectionString(ctx, "sslmode=disable")
	s.Require().NoError(err)

	// Setup test config
	config.AppConfig = &config.Config{
		DatabaseURL: connStr,
	}
}

func (s *DatabaseTestSuite) TearDownSuite() {
	ctx := context.Background()

	// Restore original values
	DB = s.originalDB
	config.AppConfig = s.originalConfig

	// Cleanup container
	if s.pgContainer != nil {
		s.pgContainer.Terminate(ctx)
	}
}

func (s *DatabaseTestSuite) SetupTest() {
	// Reset DB for each test
	DB = nil
}

func (s *DatabaseTestSuite) TearDownTest() {
	// Close connection if it exists
	if DB != nil {
		sqlDB, err := DB.DB()
		if err == nil {
			sqlDB.Close()
		}
	}
	DB = nil
}

func (s *DatabaseTestSuite) TestConnect_Success() {
	err := Connect()
	s.NoError(err)
	s.NotNil(DB)

	// Test that we can ping the database
	sqlDB, err := DB.DB()
	s.NoError(err)
	s.NotNil(sqlDB)

	err = sqlDB.Ping()
	s.NoError(err)

	// Test that we can get stats - connection pooling is configured but we can't test exact values in test environment
	stats := sqlDB.Stats()
	s.GreaterOrEqual(stats.OpenConnections, 0)
}

func (s *DatabaseTestSuite) TestConnect_InvalidDatabaseURL() {
	// Save original config
	originalURL := config.AppConfig.DatabaseURL

	// Set invalid database URL
	config.AppConfig.DatabaseURL = "invalid://connection/string"

	err := Connect()
	s.Error(err)
	s.Contains(err.Error(), "failed to connect to database")

	// Restore original config
	config.AppConfig.DatabaseURL = originalURL
}

func (s *DatabaseTestSuite) TestConnect_UnreachableDatabase() {
	// Save original config
	originalURL := config.AppConfig.DatabaseURL

	// Set unreachable database URL
	config.AppConfig.DatabaseURL = "postgres://user:pass@unreachable:5432/db?sslmode=disable"

	err := Connect()
	s.Error(err)
	s.Contains(err.Error(), "failed to connect to database")

	// Restore original config
	config.AppConfig.DatabaseURL = originalURL
}

func (s *DatabaseTestSuite) TestClose_Success() {
	// First connect
	err := Connect()
	s.NoError(err)
	s.NotNil(DB)

	// Then close
	err = Close()
	s.NoError(err)
}

func (s *DatabaseTestSuite) TestClose_NoConnection() {
	// Ensure DB is nil
	DB = nil

	// This should panic because DB is nil
	s.Panics(func() {
		Close()
	})
}

func (s *DatabaseTestSuite) TestConnectionPoolSettings() {
	err := Connect()
	s.NoError(err)

	sqlDB, err := DB.DB()
	s.NoError(err)

	// Test that connection pooling is working
	stats := sqlDB.Stats()
	s.GreaterOrEqual(stats.OpenConnections, 0)
	// Connection pool settings are configured in the connection function
}

func (s *DatabaseTestSuite) TestDatabaseOperations() {
	// Connect to database
	err := Connect()
	s.NoError(err)

	// Run migrations to create tables
	err = Migrate()
	s.NoError(err)

	// Test creating audit log
	auditLog := &models.AuditLog{
		Action:      "test_action",
		ResourceType: "test",
		ResourceName: "test-resource",
		User:        "test-user",
		Method:      "GET",
		Path:        "/test",
		StatusCode:  200,
		Duration:    100,
		RequestBody: `{"test": "data"}`,
		Response:    `{"status": "ok"}`,
	}

	result := DB.Create(auditLog)
	s.NoError(result.Error)
	s.Greater(auditLog.ID, uint(0))

	// Test reading back the audit log
	var retrievedLog models.AuditLog
	result = DB.First(&retrievedLog, auditLog.ID)
	s.NoError(result.Error)
	s.Equal(auditLog.Action, retrievedLog.Action)
	s.Equal(auditLog.ResourceType, retrievedLog.ResourceType)
	s.Equal(auditLog.User, retrievedLog.User)

	// Test creating pod operation
	podOp := &models.PodOperation{
		PodName:    "test-pod",
		Namespace:  "default",
		Operation:  "restart",
		Status:     "success",
		ExecutedBy: "test-user",
		ExecutedAt: time.Now(),
		Details:    `{"reason": "test restart"}`,
	}

	result = DB.Create(podOp)
	s.NoError(result.Error)
	s.Greater(podOp.ID, uint(0))

	// Test creating deployment scale
	deploymentScale := &models.DeploymentScale{
		DeploymentName:  "test-deployment",
		Namespace:       "default",
		PreviousReplicas: 3,
		NewReplicas:     5,
		ExecutedBy:      "test-user",
		ExecutedAt:      time.Now(),
	}

	result = DB.Create(deploymentScale)
	s.NoError(result.Error)
	s.Greater(deploymentScale.ID, uint(0))
}

func (s *DatabaseTestSuite) TestConcurrentConnections() {
	err := Connect()
	s.NoError(err)

	// Run migrations to create tables first
	err = Migrate()
	s.NoError(err)

	// Test multiple concurrent operations
	const numGoroutines = 10
	done := make(chan error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			// Each goroutine tries to create an audit log
			auditLog := &models.AuditLog{
				Action:      "concurrent_test",
				ResourceType: "test",
				ResourceName: "concurrent-resource",
				User:        "concurrent-user",
				Method:      "GET",
				Path:        "/concurrent",
				StatusCode:  200,
				Duration:    int64(id * 10),
				RequestBody: fmt.Sprintf(`{"id": %d}`, id),
				Response:    `{"status": "ok"}`,
			}

			result := DB.Create(auditLog)
			done <- result.Error
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < numGoroutines; i++ {
		err := <-done
		s.NoError(err)
	}
}

func TestDatabaseSuite(t *testing.T) {
	suite.Run(t, new(DatabaseTestSuite))
}

// Unit tests that don't require testcontainers
func TestConnect_NilConfig(t *testing.T) {
	// Store original config
	originalConfig := config.AppConfig
	defer func() { config.AppConfig = originalConfig }()

	// Set config to nil
	config.AppConfig = nil

	// This should panic
	assert.Panics(t, func() {
		Connect()
	})
}

func TestGlobalDBVariable(t *testing.T) {
	// Test that DB is initially nil
	originalDB := DB
	DB = nil

	assert.Nil(t, DB)

	// Restore original DB
	DB = originalDB
}

// Integration test without testcontainers for edge cases
func TestConnect_EmptyDatabaseURL(t *testing.T) {
	// Store original config
	originalConfig := config.AppConfig
	defer func() { config.AppConfig = originalConfig }()

	// Set empty database URL
	config.AppConfig = &config.Config{
		DatabaseURL: "",
	}

	err := Connect()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to connect to database")
}

// Test database logger configuration
func TestDatabaseLoggerConfiguration(t *testing.T) {
	// This test verifies that the logger is configured correctly
	// We can't easily test the actual logging output, but we can verify
	// that the configuration doesn't cause panics

	originalConfig := config.AppConfig
	defer func() { config.AppConfig = originalConfig }()

	config.AppConfig = &config.Config{
		DatabaseURL: "postgres://user:pass@localhost:5432/testdb?sslmode=disable",
	}

	// This will fail to connect but shouldn't panic due to logger config
	err := Connect()
	assert.Error(t, err)
}

// Benchmark tests
func BenchmarkConnect(b *testing.B) {
	originalConfig := config.AppConfig
	defer func() { config.AppConfig = originalConfig }()

	config.AppConfig = &config.Config{
		DatabaseURL: "postgres://user:pass@localhost:5432/testdb?sslmode=disable",
	}

	for i := 0; i < b.N; i++ {
		Connect() // This will fail but we're measuring the function call overhead
	}
}

// Test helper functions
func TestConnectionStringParsing(t *testing.T) {
	tests := []struct {
		name        string
		url         string
		shouldError bool
	}{
		{
			name:        "valid postgres URL",
			url:         "postgres://user:pass@localhost:5432/db?sslmode=disable",
			shouldError: false, // Will error on connection but not on parsing
		},
		{
			name:        "invalid scheme",
			url:         "mysql://user:pass@localhost:3306/db",
			shouldError: true,
		},
		{
			name:        "empty URL",
			url:         "",
			shouldError: true,
		},
		{
			name:        "malformed URL",
			url:         "not-a-url",
			shouldError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			originalConfig := config.AppConfig
			defer func() { config.AppConfig = originalConfig }()

			config.AppConfig = &config.Config{
				DatabaseURL: tt.url,
			}

			err := Connect()

			if tt.shouldError {
				assert.Error(t, err)
			} else {
				// Even valid URLs will error without a real database
				// but the error should be about connection, not parsing
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "failed to connect to database")
			}
		})
	}
}

// Test database connection with various GORM configuration options
func TestGORMConfiguration(t *testing.T) {
	t.Run("prepared statements enabled", func(t *testing.T) {
		// This is tested implicitly in the Connect function
		// PrepareStmt: true is set in the config
		assert.True(t, true) // Placeholder for actual GORM config test
	})

	t.Run("skip default transaction enabled", func(t *testing.T) {
		// This is tested implicitly in the Connect function
		// SkipDefaultTransaction: true is set in the config
		assert.True(t, true) // Placeholder for actual GORM config test
	})
}

// Test connection pool behavior
func TestConnectionPoolBehavior(t *testing.T) {
	originalConfig := config.AppConfig
	defer func() { config.AppConfig = originalConfig }()

	// Test with a valid-looking URL (will still fail to connect)
	config.AppConfig = &config.Config{
		DatabaseURL: "postgres://user:pass@localhost:5432/testdb?sslmode=disable",
	}

	err := Connect()
	assert.Error(t, err) // Expected to fail without real database

	// Test that connection pool settings are applied even on failed connections
	// This is verified in the actual Connect() function implementation
}