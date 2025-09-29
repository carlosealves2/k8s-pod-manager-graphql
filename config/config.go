package config

import (
	"fmt"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	Port           string
	DatabaseURL    string
	KubeConfig     string
	InCluster      bool
	LogLevel       string
	EnableCORS     bool
	AllowedOrigins []string
	RateLimit      int
}

var AppConfig *Config

func Load() error {
	_ = godotenv.Load()

	AppConfig = &Config{
		Port:        getEnv("PORT", "8080"),
		DatabaseURL: getEnv("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/k8s_pod_manager?sslmode=disable"),
		KubeConfig:  getEnv("KUBECONFIG", ""),
		InCluster:   getEnvBool("IN_CLUSTER", false),
		LogLevel:    getEnv("LOG_LEVEL", "info"),
		EnableCORS:  getEnvBool("ENABLE_CORS", true),
		RateLimit:   getEnvInt("RATE_LIMIT", 100),
	}

	if AppConfig.EnableCORS {
		origins := getEnv("ALLOWED_ORIGINS", "*")
		if origins == "*" {
			AppConfig.AllowedOrigins = []string{"*"}
		} else {
			AppConfig.AllowedOrigins = []string{origins}
		}
	}

	if AppConfig.KubeConfig == "" && !AppConfig.InCluster {
		home, _ := os.UserHomeDir()
		AppConfig.KubeConfig = fmt.Sprintf("%s/.kube/config", home)
	}

	return nil
}

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
	strValue := getEnv(key, "")
	if strValue == "" {
		return defaultValue
	}
	value, err := strconv.ParseBool(strValue)
	if err != nil {
		return defaultValue
	}
	return value
}

func getEnvInt(key string, defaultValue int) int {
	strValue := getEnv(key, "")
	if strValue == "" {
		return defaultValue
	}
	value, err := strconv.Atoi(strValue)
	if err != nil {
		return defaultValue
	}
	return value
}