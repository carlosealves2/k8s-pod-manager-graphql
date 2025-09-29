// +build deprecated_fiber

// Package middleware contains legacy Fiber-based middleware implementations.
//
// DEPRECATED: This file contains Fiber-specific middleware that is incompatible
// with the current Gin-based application. Use the new framework-agnostic middleware
// implementations instead:
//
//   - For error handling: Use DefaultErrorHandler with ToGinHandler()
//   - For rate limiting: Use InMemoryRateLimiter with RateLimitMiddleware
//   - For recovery: Use RecoveryMiddleware
//   - For request IDs: Use RequestIDMiddleware
//   - For logging: Use LoggingMiddleware
//
// See integration_example.go for migration examples.
package middleware

import (
	"crypto/rand"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
)

func ErrorHandler() fiber.ErrorHandler {
	return func(c *fiber.Ctx, err error) error {
		code := fiber.StatusInternalServerError
		message := "Internal Server Error"

		if e, ok := err.(*fiber.Error); ok {
			code = e.Code
			message = e.Message
		} else {
			message = err.Error()
		}

		log.Printf("Error: %v | Path: %s | Method: %s | IP: %s", err, c.Path(), c.Method(), c.IP())

		return c.Status(code).JSON(fiber.Map{
			"error":   message,
			"code":    code,
			"path":    c.Path(),
			"method":  c.Method(),
			"request_id": c.Get("X-Request-ID", ""),
		})
	}
}

func RateLimiter(limit int) fiber.Handler {
	requestCounts := make(map[string]int)
	resetTime := make(map[string]time.Time)
	mu := &sync.RWMutex{}

	return func(c *fiber.Ctx) error {
		ip := c.IP()
		now := time.Now()

		mu.Lock()
		if reset, exists := resetTime[ip]; !exists || now.After(reset) {
			requestCounts[ip] = 0
			resetTime[ip] = now.Add(time.Minute)
		}

		requestCounts[ip]++
		currentCount := requestCounts[ip]
		resetTimeForIP := resetTime[ip]
		mu.Unlock()

		if currentCount > limit {
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"error": "Rate limit exceeded",
				"retry_after": resetTimeForIP.Unix(),
			})
		}

		c.Set("X-RateLimit-Limit", fmt.Sprintf("%d", limit))
		c.Set("X-RateLimit-Remaining", fmt.Sprintf("%d", limit-currentCount))
		c.Set("X-RateLimit-Reset", fmt.Sprintf("%d", resetTimeForIP.Unix()))

		return c.Next()
	}
}

func Recovery() fiber.Handler {
	return func(c *fiber.Ctx) error {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("Panic recovered: %v | Path: %s | Method: %s | IP: %s", r, c.Path(), c.Method(), c.IP())
				_ = c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"error": "Internal server error",
					"message": "An unexpected error occurred",
				})
			}
		}()

		return c.Next()
	}
}

func RequestID() fiber.Handler {
	return func(c *fiber.Ctx) error {
		requestID := c.Get("X-Request-ID")
		if requestID == "" {
			requestID = generateRequestID()
			c.Set("X-Request-ID", requestID)
		}

		c.Locals("requestID", requestID)
		return c.Next()
	}
}

func generateRequestID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return fmt.Sprintf("%x-%x-%x-%x-%x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}