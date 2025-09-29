package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/extension"
	"github.com/99designs/gqlgen/graphql/handler/transport"
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/carlosf/k8s-pod-manager/config"
	"github.com/carlosf/k8s-pod-manager/graph"
	"github.com/carlosf/k8s-pod-manager/internal/database"
	"github.com/carlosf/k8s-pod-manager/internal/kubernetes"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

func main() {
	if err := config.Load(); err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	if err := database.Connect(); err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer database.Close()

	if err := database.Migrate(); err != nil {
		log.Fatalf("Failed to run database migrations: %v", err)
	}

	if err := kubernetes.InitClient(); err != nil {
		log.Fatalf("Failed to initialize Kubernetes client: %v", err)
	}

	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.Recovery())

	if config.AppConfig.EnableCORS {
		corsConfig := cors.DefaultConfig()
		corsConfig.AllowMethods = []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}
		corsConfig.AllowHeaders = []string{"Origin", "Content-Type", "Accept", "Authorization", "X-Request-ID", "X-User"}
		corsConfig.ExposeHeaders = []string{"X-Request-ID", "X-RateLimit-Limit", "X-RateLimit-Remaining", "X-RateLimit-Reset"}
		corsConfig.MaxAge = 3600 * time.Second

		if len(config.AppConfig.AllowedOrigins) > 0 && config.AppConfig.AllowedOrigins[0] == "*" {
			corsConfig.AllowOrigins = []string{"http://localhost:5173", "http://localhost:3000", "http://127.0.0.1:5173", "http://127.0.0.1:3000", "http://localhost:8080"}
		} else {
			corsConfig.AllowOrigins = config.AppConfig.AllowedOrigins
		}
		corsConfig.AllowCredentials = true

		router.Use(cors.New(corsConfig))
	}

	setupRoutes(router)

	server := &http.Server{
		Addr:         ":" + config.AppConfig.Port,
		Handler:      router,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	go gracefulShutdown(server)

	log.Printf("Starting server on port %s", config.AppConfig.Port)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Failed to start server: %v", err)
	}
}

func setupRoutes(router *gin.Engine) {
	// Create GraphQL server with WebSocket support for subscriptions
	resolver := graph.NewDefaultResolver()
	srv := handler.New(graph.NewExecutableSchema(graph.Config{Resolvers: resolver}))

	// Configure WebSocket transport for subscriptions
	srv.AddTransport(transport.POST{})
	srv.AddTransport(transport.Websocket{
		Upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
			Subprotocols: []string{"graphql-ws", "graphql-transport-ws"},
		},
	})

	// Enable introspection for GraphQL Playground
	srv.Use(extension.Introspection{})

	// GraphQL Playground (for development)
	router.GET("/", gin.WrapH(playground.Handler("GraphQL Playground", "/graphql")))

	// GraphQL endpoint with WebSocket support
	router.POST("/graphql", gin.WrapH(srv))
	router.GET("/graphql", gin.WrapH(srv))

	// Health check endpoints
	api := router.Group("/api/v1")

	// Simple health endpoints for now - will be properly implemented later
	api.GET("/health", func(c *gin.Context) { c.JSON(200, gin.H{"status": "ok"}) })
	api.GET("/ready", func(c *gin.Context) { c.JSON(200, gin.H{"status": "ready"}) })
	api.GET("/info", func(c *gin.Context) { c.JSON(200, gin.H{"app": "k8s-pod-manager", "version": "1.0.0"}) })
	api.GET("/metrics", func(c *gin.Context) { c.String(200, "# Metrics placeholder") })


	// API info endpoint
	api.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"service":       "K8s Pod Manager GraphQL API",
			"version":       "1.0.0",
			"graphql":       "/graphql",
			"playground":    "/",
			"health":        "/api/v1/health",
			"readiness":     "/api/v1/ready",
			"info":          "/api/v1/info",
			"metrics":       "/api/v1/metrics",
			"subscriptions": "/graphql (WebSocket)",
			"description":   "Use GraphQL endpoint at /graphql for all pod management operations",
		})
	})

	// 404 handler
	router.NoRoute(func(c *gin.Context) {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Resource not found. Use GraphQL endpoint at /graphql",
			"path":  c.Request.URL.Path,
		})
	})
}

func gracefulShutdown(server *http.Server) {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	<-quit
	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Printf("Server shutdown error: %v", err)
	}

	log.Println("Server shutdown complete")
}

func joinStrings(strs []string, sep string) string {
	if len(strs) == 0 {
		return ""
	}
	result := strs[0]
	for i := 1; i < len(strs); i++ {
		result += sep + strs[i]
	}
	return result
}