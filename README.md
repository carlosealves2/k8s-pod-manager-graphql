# K8s Pod Manager API

API REST para gerenciamento de pods Kubernetes usando Go 1.24.7, Fiber v2 e PostgreSQL.

## Funcionalidades

- ✅ Listar pods de um namespace
- ✅ Obter detalhes de um pod específico
- ✅ Reiniciar pods (rollout restart para Deployments/StatefulSets)
- ✅ Deletar pods
- ✅ Escalar Deployments (parar/iniciar)
- ✅ Auditoria completa em PostgreSQL
- ✅ Rate limiting
- ✅ CORS configurável
- ✅ Health checks
- ✅ Métricas Prometheus

## Arquitetura

```
k8s-pod-manager/
├── config/           # Configurações
├── internal/
│   ├── database/     # Conexão e migrations
│   ├── models/       # Modelos GORM
│   ├── kubernetes/   # Cliente K8s
│   ├── services/     # Lógica de negócio
│   ├── handlers/     # Handlers HTTP
│   └── middleware/   # Middlewares
├── migrations/       # SQL migrations
├── k8s/              # Manifests Kubernetes
└── main.go           # Entry point
```

## Pré-requisitos

- Go 1.24.7+
- Docker e Docker Compose
- Kubernetes cluster (ou minikube/kind)
- PostgreSQL 15+

## Instalação

### 1. Clonar o repositório

```bash
git clone https://github.com/seu-usuario/k8s-pod-manager.git
cd k8s-pod-manager
```

### 2. Configurar variáveis de ambiente

```bash
cp .env.example .env
# Editar .env com suas configurações
```

### 3. Desenvolvimento Local

```bash
# Iniciar com docker-compose
docker-compose up -d

# Ou executar localmente
go mod download
go run main.go
```

## Deploy no Kubernetes

### 1. Build da imagem Docker

```bash
docker build -t k8s-pod-manager:latest .
```

### 2. Aplicar manifests

```bash
# Deploy do PostgreSQL
kubectl apply -f k8s/postgres.yaml

# Aplicar RBAC
kubectl apply -f k8s/rbac.yaml

# Deploy da aplicação
kubectl apply -f k8s/service.yaml
kubectl apply -f k8s/deployment.yaml
```

## API Endpoints

### Health Checks

- `GET /api/v1/health` - Health check
- `GET /api/v1/ready` - Readiness check
- `GET /api/v1/info` - Informações do sistema
- `GET /api/v1/metrics` - Métricas Prometheus

### Pods

- `GET /api/v1/namespaces/:namespace/pods` - Listar pods
- `GET /api/v1/namespaces/:namespace/pods/:pod` - Detalhes do pod
- `POST /api/v1/namespaces/:namespace/pods/:pod/restart` - Reiniciar pod
- `DELETE /api/v1/namespaces/:namespace/pods/:pod` - Deletar pod

### Deployments

- `GET /api/v1/namespaces/:namespace/deployments` - Listar deployments
- `POST /api/v1/namespaces/:namespace/deployments/:deployment/scale` - Escalar deployment

## Exemplos de Uso

### Listar pods

```bash
curl http://localhost:8080/api/v1/namespaces/default/pods
```

### Reiniciar pod

```bash
curl -X POST http://localhost:8080/api/v1/namespaces/default/pods/meu-pod-123/restart
```

### Escalar deployment

```bash
# Parar (0 réplicas)
curl -X POST http://localhost:8080/api/v1/namespaces/default/deployments/minha-app/scale \
  -H 'Content-Type: application/json' \
  -d '{"replicas": 0}'

# Iniciar (3 réplicas)
curl -X POST http://localhost:8080/api/v1/namespaces/default/deployments/minha-app/scale \
  -H 'Content-Type: application/json' \
  -d '{"replicas": 3}'
```

## Variáveis de Ambiente

| Variável | Descrição | Default |
|----------|-----------|---------|
| `PORT` | Porta da API | 8080 |
| `DATABASE_URL` | URL de conexão PostgreSQL | - |
| `IN_CLUSTER` | Executando dentro do cluster | false |
| `KUBECONFIG` | Path do kubeconfig | ~/.kube/config |
| `LOG_LEVEL` | Nível de log (debug/info/warn/error) | info |
| `ENABLE_CORS` | Habilitar CORS | true |
| `ALLOWED_ORIGINS` | Origins permitidas | * |
| `RATE_LIMIT` | Limite de requisições por minuto | 100 |

## Segurança

- RBAC configurado com permissões mínimas
- Execução com usuário non-root (UID 1000)
- Rate limiting configurável
- Auditoria completa de operações
- Secrets do Kubernetes para credenciais

## Monitoramento

A API expõe métricas no formato Prometheus em `/api/v1/metrics`:

- Conexões de database
- Uptime da aplicação
- Informações de versão

## Desenvolvimento

### Executar testes

```bash
go test ./...
```

### Verificar lint

```bash
golangci-lint run
```

### Build local

```bash
go build -o k8s-pod-manager .
```

## Licença

MIT

## Contribuindo

1. Fork o projeto
2. Crie sua feature branch (`git checkout -b feature/AmazingFeature`)
3. Commit suas mudanças (`git commit -m 'Add some AmazingFeature'`)
4. Push para a branch (`git push origin feature/AmazingFeature`)
5. Abra um Pull Request