// +build deprecated_fiber

// DEPRECATED: This file contains Fiber-specific middleware that is incompatible
// with the current Gin-based application. Use the new LoggingMiddleware with
// DatabaseAuditLogger instead. See integration_example.go for migration examples.
package middleware

import (
	"encoding/json"
	"time"

	"github.com/carlosf/k8s-pod-manager/internal/database"
	"github.com/carlosf/k8s-pod-manager/internal/models"
	"github.com/gofiber/fiber/v2"
)

func Logger() fiber.Handler {
	return func(c *fiber.Ctx) error {
		start := time.Now()

		auditLog := &models.AuditLog{
			Method:    c.Method(),
			Path:      c.Path(),
			IP:        c.IP(),
			UserAgent: c.Get("User-Agent"),
			User:      c.Get("X-User", "anonymous"),
			CreatedAt: start,
		}

		if c.Method() == "POST" || c.Method() == "PUT" || c.Method() == "PATCH" {
			body := c.Body()
			if len(body) > 0 {
				var jsonBody interface{}
				if err := json.Unmarshal(body, &jsonBody); err == nil {
					if bodyBytes, err := json.Marshal(jsonBody); err == nil {
						auditLog.RequestBody = string(bodyBytes)
					}
				}
			}
		}

		err := c.Next()

		duration := time.Since(start)
		auditLog.Duration = duration.Milliseconds()
		auditLog.StatusCode = c.Response().StatusCode()

		parseAction(c, auditLog)

		if err != nil {
			auditLog.Error = err.Error()
		}

		if database.DB != nil {
			go func() {
				database.DB.Create(auditLog)
			}()
		}

		return err
	}
}

func parseAction(c *fiber.Ctx, log *models.AuditLog) {
	path := c.Path()
	method := c.Method()

	namespace := c.Params("namespace", "")
	pod := c.Params("pod", "")
	deployment := c.Params("deployment", "")

	log.Namespace = namespace

	switch {
	case method == "GET" && pod != "":
		log.Action = "get_pod"
		log.ResourceType = "pod"
		log.ResourceName = pod
	case method == "GET" && path == "/api/v1/namespaces/:namespace/pods":
		log.Action = "list_pods"
		log.ResourceType = "pod"
		log.ResourceName = "all"
	case method == "POST" && path == "/graphql":
		log.Action = "graphql_request"
		log.ResourceType = "graphql"
		log.ResourceName = "mutation"
	case method == "DELETE" && pod != "":
		log.Action = "delete_pod"
		log.ResourceType = "pod"
		log.ResourceName = pod
	case method == "POST" && deployment != "" && path == "/api/v1/namespaces/:namespace/deployments/:deployment/scale":
		log.Action = "scale_deployment"
		log.ResourceType = "deployment"
		log.ResourceName = deployment
	case method == "GET" && path == "/api/v1/namespaces/:namespace/deployments":
		log.Action = "list_deployments"
		log.ResourceType = "deployment"
		log.ResourceName = "all"
	case path == "/api/v1/health":
		log.Action = "health_check"
		log.ResourceType = "system"
		log.ResourceName = "health"
	case path == "/api/v1/ready":
		log.Action = "readiness_check"
		log.ResourceType = "system"
		log.ResourceName = "readiness"
	case path == "/api/v1/metrics":
		log.Action = "metrics"
		log.ResourceType = "system"
		log.ResourceName = "metrics"
	case path == "/api/v1/info":
		log.Action = "info"
		log.ResourceType = "system"
		log.ResourceName = "info"
	default:
		log.Action = "unknown"
		log.ResourceType = "unknown"
		log.ResourceName = path
	}
}