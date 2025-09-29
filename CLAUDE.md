# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Development Commands

### Running the Application
```bash
go run main.go
```
The server runs on port 8080 by default. GraphQL Playground is available at `/` and the GraphQL endpoint at `/graphql`.

### Development Tools
```bash
go test ./...              # Run all tests
golangci-lint run          # Run linter
go build -o k8s-pod-manager .  # Build binary
go mod tidy                # Update dependencies
make build                 # Build using Makefile
make test                  # Run tests via Makefile
make lint                  # Run linter via Makefile
make fmt                   # Format code
```

### GraphQL Code Generation
```bash
go generate ./...          # Generate GraphQL code from schema
```
This uses gqlgen to regenerate resolvers and models from `graph/schema.graphqls`.

### Docker Development
```bash
docker-compose up -d       # Start with PostgreSQL
docker build -t k8s-pod-manager:latest .  # Build Docker image
make docker-build          # Build Docker image via Makefile
make docker-run            # Run with docker-compose
make docker-stop           # Stop docker-compose
make docker-logs           # Show container logs
```

### Kubernetes Deployment
```bash
make k8s-deploy            # Deploy to Kubernetes
make k8s-delete            # Delete from Kubernetes
make k8s-logs              # Show K8s logs
make k8s-status            # Show deployment status
```

## Architecture Overview

This is a **GraphQL API** for managing Kubernetes pods, converted from a REST API. The architecture follows a layered approach:

### Core Architecture Layers
1. **GraphQL Layer** (`graph/`): Schema-first GraphQL API using gqlgen
2. **Service Layer** (`internal/services/`): Business logic for Kubernetes operations
3. **Handler Layer** (`internal/handlers/`): Legacy REST handlers (health checks, WebSocket)
4. **Data Layer** (`internal/database/`, `internal/models/`): PostgreSQL with GORM for audit logging

### Key Architectural Decisions

**GraphQL-First Design**: The API primarily uses GraphQL with some REST endpoints maintained for:
- Health checks (`/api/v1/health`, `/api/v1/ready`, `/api/v1/info`)
- Metrics (`/api/v1/metrics`)

**GraphQL WebSocket Subscriptions**: Real-time pod watching via GraphQL subscriptions using WebSocket transport protocols (`graphql-ws`, `graphql-transport-ws`).

**Service Layer Integration**: GraphQL resolvers (`graph/schema.resolvers.go`) delegate to existing service layer (`internal/services/`) maintaining separation of concerns.

**Type Conversion Architecture**: Helper functions in `graph/helpers.go` convert between:
- Kubernetes API types (from client-go)
- Internal service types
- GraphQL model types (generated in `graph/model/`)

### Critical Integration Points

**Kubernetes Client**: Initialized in `internal/kubernetes/` and used throughout service layer. Supports both in-cluster and external kubeconfig authentication.

**Database Audit Trail**: All operations are logged to PostgreSQL via GORM models in `internal/models/` for compliance and auditing.

**GraphQL Subscriptions**: Real-time pod watching implemented via GraphQL subscriptions that integrate with Kubernetes watch API.

### Configuration and Dependencies

**Environment Variables**: Loaded via `config/` package with support for `.env` files:
- `DATABASE_URL`: PostgreSQL connection
- `IN_CLUSTER`: Kubernetes client mode
- `PORT`: Server port (default 8080)
- Rate limiting, CORS, and other operational settings

**Key Dependencies**:
- `gqlgen`: GraphQL code generation and server
- `gin`: HTTP framework for REST endpoints and GraphQL integration
- `client-go`: Kubernetes API client
- `gorm`: ORM for PostgreSQL

### GraphQL Schema Management

**Schema Definition**: Primary schema in `graph/schema.graphqls` defines all types, queries, mutations, and subscriptions.

**Code Generation**: `gqlgen.yml` configures automatic generation of:
- Resolvers: `graph/schema.resolvers.go`
- Models: `graph/model/models_gen.go`
- Server code: `graph/generated.go`

**Resolver Pattern**: Resolvers inject `*services.PodService` and `*handlers.HealthHandler` for business logic delegation.

## Kubernetes Operations

This application requires Kubernetes cluster access with RBAC permissions for:
- Pods (list, get, delete, watch)
- Deployments (list, get, update for scaling)
- StatefulSets (list, get, update for scaling)
- DaemonSets (list, get)
- ReplicaSets (list, get)
- Namespaces (list, get)

### Pod Controller Type Detection

The service layer automatically detects pod controller types and includes this information in all pod queries:
- **Deployment** (via ReplicaSet ownership)
- **StatefulSet** (direct ownership)
- **DaemonSet** (direct ownership)
- **Job** (direct ownership)
- **CronJob** (via Job ownership)
- **ReplicaSet** (direct ownership)
- **Node** (for system pods)
- **Standalone** (pods without controllers)

### Smart Pod Restart Logic

The service layer handles pod restart operations intelligently:
- **Deployments/StatefulSets**: Rollout restart via annotation updates
- **DaemonSets/ReplicaSets**: Direct pod deletion (controller recreates)
- **Standalone pods**: Direct deletion with warning (not recreated)