.PHONY: help build run test clean docker-build docker-run docker-stop k8s-deploy k8s-delete generate

# Variables
APP_NAME=k8s-pod-manager
DOCKER_IMAGE=$(APP_NAME):latest
GO_VERSION=1.24.7

help: ## Show this help message
	@echo "Usage: make [target]"
	@echo ""
	@echo "Available targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

build: ## Build the application
	go build -ldflags="-w -s" -o $(APP_NAME) .

run: ## Run the application locally
	go run main.go

test: ## Run tests
	go test -v ./...

clean: ## Clean build artifacts
	rm -f $(APP_NAME)
	go clean -cache

mod-download: ## Download Go modules
	go mod download

mod-tidy: ## Tidy Go modules
	go mod tidy

docker-build: ## Build Docker image
	docker build -t $(DOCKER_IMAGE) .

docker-run: ## Run with docker-compose
	docker-compose up -d

docker-stop: ## Stop docker-compose services
	docker-compose down

docker-logs: ## Show docker-compose logs
	docker-compose logs -f

docker-clean: ## Clean docker resources
	docker-compose down -v
	docker rmi $(DOCKER_IMAGE) || true

k8s-deploy: ## Deploy to Kubernetes
	kubectl apply -f k8s/postgres.yaml
	kubectl apply -f k8s/rbac.yaml
	kubectl apply -f k8s/service.yaml
	kubectl apply -f k8s/deployment.yaml

k8s-delete: ## Delete from Kubernetes
	kubectl delete -f k8s/deployment.yaml || true
	kubectl delete -f k8s/service.yaml || true
	kubectl delete -f k8s/rbac.yaml || true
	kubectl delete -f k8s/postgres.yaml || true

k8s-logs: ## Show Kubernetes logs
	kubectl logs -f -l app=$(APP_NAME)

k8s-status: ## Show Kubernetes deployment status
	kubectl get pods,svc,deployment -l app=$(APP_NAME)

port-forward: ## Port forward to Kubernetes service
	kubectl port-forward service/$(APP_NAME) 8080:80

db-migrate: ## Run database migrations
	@echo "Migrations are auto-applied on startup"

lint: ## Run linter
	golangci-lint run ./...

fmt: ## Format code
	go fmt ./...

vet: ## Run go vet
	go vet ./...

generate: ## Generate GraphQL code
	go generate