# K8s Pod Manager

<div align="center">

**Enterprise-grade GraphQL API for Kubernetes Pod Management**

[![Go Version](https://img.shields.io/badge/Go-1.24.7-00ADD8?style=flat&logo=go)](https://golang.org/)
[![GraphQL](https://img.shields.io/badge/GraphQL-gqlgen-E10098?style=flat&logo=graphql)](https://gqlgen.com/)
[![Kubernetes](https://img.shields.io/badge/Kubernetes-client--go-326CE5?style=flat&logo=kubernetes)](https://kubernetes.io/)
[![PostgreSQL](https://img.shields.io/badge/PostgreSQL-15-336791?style=flat&logo=postgresql)](https://www.postgresql.org/)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)

[Features](#features) • [Quick Start](#quick-start) • [API Documentation](#api-documentation) • [Architecture](#architecture) • [Deployment](#deployment)

</div>

---

## 📑 Table of Contents

- [Overview](#overview)
- [Features](#features)
- [Technology Stack](#technology-stack)
- [Architecture](#architecture)
- [Prerequisites](#prerequisites)
- [Quick Start](#quick-start)
- [API Documentation](#api-documentation)
  - [GraphQL Queries](#graphql-queries)
  - [GraphQL Mutations](#graphql-mutations)
  - [GraphQL Subscriptions](#graphql-subscriptions)
  - [REST Endpoints](#rest-endpoints)
- [Configuration](#configuration)
- [Deployment Options](#deployment-options)
  - [Local Development](#local-development)
  - [Docker](#docker)
  - [Kubernetes](#kubernetes)
- [Development](#development)
- [CI/CD Pipeline](#cicd-pipeline)
- [Security](#security)
- [Contributing](#contributing)
- [License](#license)

---

## Overview

**K8s Pod Manager** is a production-ready GraphQL API for managing Kubernetes resources with real-time monitoring capabilities. Originally built as a REST API, it has been completely redesigned as a **schema-first GraphQL** implementation using gqlgen, providing a modern, type-safe, and efficient API for Kubernetes operations.

### Key Highlights

- 🚀 **GraphQL-First Design** - Schema-first approach with automatic code generation
- 📡 **Real-Time Subscriptions** - WebSocket-based live pod monitoring and log streaming
- 🧠 **Intelligent Operations** - Auto-detects pod controller types for optimal restart strategies
- 📊 **Complete Audit Trail** - PostgreSQL-backed operation logging for compliance
- 🔒 **Production Security** - Security scanning, RBAC, non-root containers
- 🎯 **Zero-Downtime Restarts** - Rollout restart for Deployments and StatefulSets
- 🏗️ **SOLID Architecture** - Clean, testable, and maintainable codebase

---

## Features

### ✅ Pod Management
- List pods across all namespaces or filtered by namespace
- Get detailed pod information including containers, status, and controller type
- **Intelligent pod restart** with automatic controller detection
- Delete pods with audit logging
- **Real-time pod watching** via GraphQL subscriptions
- **Live log streaming** from pod containers

### ✅ Deployment & StatefulSet Management
- List deployments and statefulsets
- Scale replicas up or down
- Automated rollout restart for zero-downtime updates

### ✅ Real-Time Monitoring
- **WebSocket subscriptions** for pod events (ADDED, MODIFIED, DELETED)
- **Continuous log streaming** with automatic reconnection
- Multi-protocol support (`graphql-ws` and `graphql-transport-ws`)

### ✅ Smart Restart Logic
Automatically detects pod controller type and applies the appropriate strategy:
- **Deployments/StatefulSets**: Rollout restart (zero-downtime)
- **DaemonSets/ReplicaSets**: Pod deletion (controller recreates)
- **Standalone pods**: Deletion with warning (won't be recreated)

### ✅ Audit & Compliance
- Complete audit trail in PostgreSQL
- Operation logging with user/executor tracking
- Before/after state tracking for scaling operations
- Timestamp and duration recording

### ✅ Production-Ready
- Health and readiness probes
- Rate limiting and CORS support
- Security scanning (Gosec, Trivy)
- Code quality analysis (SonarQube)
- Comprehensive CI/CD pipeline

---

## Technology Stack

| Component | Technology | Version |
|-----------|-----------|---------|
| **Language** | Go | 1.24.7 |
| **GraphQL** | gqlgen | 0.17.81 |
| **HTTP Framework** | Gin | Latest |
| **Kubernetes Client** | client-go | 0.34.1 |
| **Database** | PostgreSQL | 15+ |
| **ORM** | GORM | Latest |
| **WebSocket** | gorilla/websocket | Latest |
| **Containerization** | Docker | Multi-stage |
| **Orchestration** | Kubernetes | 1.25+ |

---

## Architecture

### Layered Architecture

```
┌─────────────────────────────────────────────────────┐
│              GraphQL Layer (graph/)                 │
│  • Schema definition (schema.graphqls)              │
│  • Resolvers (queries, mutations, subscriptions)    │
│  • Type converters (K8s ↔ GraphQL models)           │
└─────────────────────────────────────────────────────┘
                         ↓
┌─────────────────────────────────────────────────────┐
│          Service Layer (internal/services/)         │
│  • PodService - Pod operations                      │
│  • DeploymentService - Deployment operations        │
│  • StatefulSetService - StatefulSet operations      │
│  • NamespaceService - Namespace operations          │
│  • RestartOrchestrator - Intelligent restart logic  │
│  • ControllerDetector - Pod owner detection         │
└─────────────────────────────────────────────────────┘
                         ↓
┌─────────────────────────────────────────────────────┐
│              Integration Layer                      │
│  • Kubernetes Client (client-go)                    │
│  • PostgreSQL Database (GORM)                       │
└─────────────────────────────────────────────────────┘
```

### Controller Detection System

The system automatically detects pod ownership and controller types:

- **Deployment** (via ReplicaSet ownership chain)
- **StatefulSet** (direct ownership)
- **DaemonSet** (direct ownership)
- **Job/CronJob** (with ownership chain)
- **ReplicaSet** (standalone)
- **Node** (static pods)
- **Standalone** (no controller)

### Project Structure

```
k8s-pod-manager/
├── graph/                      # GraphQL layer
│   ├── schema.graphqls         # GraphQL schema definition
│   ├── schema.resolvers.go     # Resolver implementations
│   ├── generated.go            # Generated GraphQL server code
│   ├── converters.go           # Type converters
│   ├── interfaces.go           # GraphQL interfaces
│   ├── scalar/                 # Custom scalar types
│   └── model/                  # Generated GraphQL models
├── internal/
│   ├── services/               # Business logic
│   │   ├── pod_service_impl.go
│   │   ├── deployment_service_impl.go
│   │   ├── statefulset_service_impl.go
│   │   ├── namespace_service_impl.go
│   │   ├── restart_orchestrator.go
│   │   └── controller_detector.go
│   ├── handlers/               # HTTP handlers (health checks)
│   ├── kubernetes/             # Kubernetes client setup
│   ├── database/               # Database connection and migrations
│   ├── models/                 # GORM models (audit logs)
│   └── middleware/             # HTTP middlewares
├── config/                     # Configuration management
├── k8s/                        # Kubernetes manifests
├── migrations/                 # Database migrations
├── .github/workflows/          # CI/CD pipeline
├── main.go                     # Application entry point
├── gqlgen.yml                  # GraphQL code generation config
└── sonar-project.properties    # SonarQube configuration
```

---

## Prerequisites

- **Go** 1.24.7 or higher
- **Docker** and **Docker Compose**
- **Kubernetes cluster** (minikube, kind, or production cluster)
- **PostgreSQL** 15 or higher
- **kubectl** configured with cluster access

---

## Quick Start

### 1. Clone the Repository

```bash
git clone https://github.com/seu-usuario/k8s-pod-manager.git
cd k8s-pod-manager
```

### 2. Configure Environment

```bash
cp .env.example .env
# Edit .env with your configurations
```

### 3. Start with Docker Compose

```bash
docker-compose up -d
```

### 4. Access GraphQL Playground

Open your browser at **http://localhost:8080/**

### 5. Try Your First Query

```graphql
query {
  namespaces {
    name
    age
  }
}
```

---

## API Documentation

### GraphQL Endpoint

Access the **GraphQL Playground** at `http://localhost:8080/` to explore the complete API schema interactively.

### Available Operations

#### **Queries**
- `health`, `readiness`, `info` - Health checks and system information
- `namespaces` - List all namespaces
- `pods(namespace)`, `allPods`, `pod(namespace, name)` - Pod information
- `deployments(namespace)` - List deployments
- `statefulsets(namespace)` - List statefulsets

#### **Mutations**
- `restartPod(namespace, name, user)` - Intelligent pod restart
- `deletePod(namespace, name, user)` - Delete pod
- `scaleDeployment(namespace, name, input, user)` - Scale deployment
- `scaleStatefulSet(namespace, name, input, user)` - Scale statefulset

#### **Subscriptions**
- `watchPods(namespace)` - Real-time pod events for namespace
- `watchAllPods` - Real-time pod events for all namespaces
- `streamPodLogs(namespace, name, container, user)` - Live log streaming

### Example Usage

```graphql
# List pods
query {
  pods(namespace: "default") {
    name
    phase
    ready
    restarts
  }
}

# Restart pod
mutation {
  restartPod(namespace: "default", name: "my-pod", user: "admin") {
    message
    type
  }
}

# Watch pods in real-time
subscription {
  watchPods(namespace: "default") {
    type
    pod { name phase }
  }
}
```

### REST Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/` | GraphQL Playground UI |
| POST/GET | `/graphql` | GraphQL endpoint |
| WebSocket | `/graphql` | GraphQL subscriptions |
| GET | `/api/v1/health` | Health check |
| GET | `/api/v1/ready` | Readiness probe |
| GET | `/api/v1/info` | System information |

---

## Configuration

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | `8080` | Server port |
| `DATABASE_URL` | `postgres://...` | PostgreSQL connection string |
| `IN_CLUSTER` | `false` | Run in Kubernetes cluster mode |
| `KUBECONFIG` | `~/.kube/config` | Path to kubeconfig file |
| `LOG_LEVEL` | `info` | Logging level (debug, info, warn, error) |
| `ENABLE_CORS` | `true` | Enable CORS support |
| `ALLOWED_ORIGINS` | `*` | Comma-separated allowed origins |
| `RATE_LIMIT` | `100` | Rate limit per minute |

### Example `.env` File

```env
PORT=8080
DATABASE_URL=postgres://user:password@localhost:5432/k8s_pod_manager?sslmode=disable
IN_CLUSTER=false
KUBECONFIG=/home/user/.kube/config
LOG_LEVEL=info
ENABLE_CORS=true
ALLOWED_ORIGINS=*
RATE_LIMIT=100
```

---

## Deployment Options

### Local Development

#### With Docker Compose (Recommended)

```bash
# Start PostgreSQL and API
docker-compose up -d

# View logs
docker-compose logs -f

# Stop services
docker-compose down
```

#### Native Go Execution

```bash
# Install dependencies
go mod download

# Run application
go run main.go

# Or build and run
make build
./k8s-pod-manager
```

### Docker

#### Build Image

```bash
# Using Makefile
make docker-build

# Or directly
docker build -t k8s-pod-manager:latest .
```

#### Run Container

```bash
docker run -d \
  -p 8080:8080 \
  -v ~/.kube/config:/root/.kube/config:ro \
  -e DATABASE_URL="postgres://..." \
  k8s-pod-manager:latest
```

### Kubernetes

#### Prerequisites

Ensure you have:
- Kubernetes cluster (1.25+)
- `kubectl` configured
- Sufficient RBAC permissions

#### Deploy All Resources

```bash
# Deploy using Makefile
make k8s-deploy

# Or manually
kubectl apply -f k8s/postgres.yaml
kubectl apply -f k8s/rbac.yaml
kubectl apply -f k8s/service.yaml
kubectl apply -f k8s/deployment.yaml
```

#### Deployed Resources

- **PostgreSQL StatefulSet** with persistent volume
- **ServiceAccount** with ClusterRole for pod management
- **Deployment** (2 replicas, security context, resource limits)
- **ClusterIP Service** exposing port 80
- **Secret** for database credentials

#### Access the API

```bash
# Port forward
kubectl port-forward service/k8s-pod-manager 8080:80

# Or use Makefile
make port-forward
```

#### Check Status

```bash
# View deployment status
make k8s-status

# View logs
make k8s-logs
```

#### Cleanup

```bash
# Delete all resources
make k8s-delete
```

### RBAC Requirements

The application requires a ServiceAccount with ClusterRole permissions:

**Required Permissions:**
- **Pods**: `get`, `list`, `watch`, `delete`, `get` on `pods/log`
- **Deployments**: `get`, `list`, `watch`, `update`, `patch` (including `/scale`)
- **StatefulSets**: `get`, `list`, `watch`, `update`, `patch`
- **DaemonSets**: `get`, `list`, `watch`
- **ReplicaSets**: `get`, `list`, `watch`
- **Namespaces**: `get`, `list`
- **Events**: `get`, `list`, `watch`

See `k8s/rbac.yaml` for complete RBAC configuration.

---

## Development

### Available Make Targets

```bash
make help              # Show all available commands
make build             # Build application binary
make run               # Run application locally
make test              # Run all tests
make lint              # Run golangci-lint
make fmt               # Format code
make generate          # Generate GraphQL code from schema
make docker-build      # Build Docker image
make docker-run        # Start with docker-compose
make docker-stop       # Stop docker-compose
make k8s-deploy        # Deploy to Kubernetes
make k8s-delete        # Delete from Kubernetes
make k8s-logs          # Show Kubernetes logs
make k8s-status        # Show deployment status
```

### GraphQL Code Generation

After modifying `graph/schema.graphqls`:

```bash
# Generate code
go generate ./...
# or
make generate
```

This regenerates:
- `graph/generated.go` - GraphQL server code
- `graph/model/models_gen.go` - Type models
- `graph/schema.resolvers.go` - Resolver stubs (preserves custom code)

### Running Tests

```bash
# Run all tests
go test -v ./...
# or
make test

# Run with coverage
go test -v -race -coverprofile=coverage.out ./...

# Run specific package
go test -v ./internal/services/...
```

### Code Quality

```bash
# Run linter
golangci-lint run ./...
# or
make lint

# Format code
go fmt ./...
# or
make fmt

# Run go vet
go vet ./...
```

---

## CI/CD Pipeline

The project includes a comprehensive GitHub Actions CI/CD pipeline (`.github/workflows/ci.yml`):

### Pipeline Stages

1. **Lint** - `golangci-lint` with comprehensive checks
2. **Test** - Unit and integration tests with race detector
3. **Security Scan** - `gosec` and `trivy` vulnerability scanning
4. **SonarQube Analysis** - Code quality and coverage reporting
5. **Build** - Binary compilation and verification

### Triggers

- Push to `main`, `develop`, or `feature/*` branches
- Pull requests to `main` or `develop`

### Required Secrets

Configure in GitHub repository settings:
- `SONAR_TOKEN` - SonarQube authentication token
- `SONAR_HOST_URL` - SonarQube server URL

---

## Security

### Container Security

- **Non-root user** (UID 1000)
- **Read-only root filesystem**
- **Dropped all capabilities**
- **Security context** configured
- **Multi-stage Docker build** with minimal attack surface

### Application Security

- **RBAC** with least-privilege principle
- **Rate limiting** to prevent abuse
- **CORS** configuration for web security
- **Audit logging** for compliance
- **Secret management** via Kubernetes Secrets

### Security Scanning

- **Gosec** - Go security scanner
- **Trivy** - Vulnerability scanner for dependencies
- **SonarQube** - Code quality and security analysis

---

## Contributing

We welcome contributions! Please follow these steps:

1. **Fork** the repository
2. **Create** a feature branch (`git checkout -b feature/AmazingFeature`)
3. **Commit** your changes (`git commit -m 'Add AmazingFeature'`)
4. **Push** to the branch (`git push origin feature/AmazingFeature`)
5. **Open** a Pull Request

### Contribution Guidelines

- Follow Go best practices and idiomatic code
- Add tests for new features
- Update documentation as needed
- Ensure CI pipeline passes
- Keep commits atomic and well-described

---

## License

This project is licensed under the **MIT License** - see the [LICENSE](LICENSE) file for details.

---

<div align="center">

**Made with ❤️ using Go and GraphQL**

[Report Bug](https://github.com/seu-usuario/k8s-pod-manager/issues) • [Request Feature](https://github.com/seu-usuario/k8s-pod-manager/issues)

</div>
