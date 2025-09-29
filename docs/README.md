# 📚 Documentação da API K8s Pod Manager

Esta pasta contém a documentação completa da API K8s Pod Manager.

## 📋 Índice

- [Swagger/OpenAPI Specification](#swagger-openapi-specification)
- [Como usar a documentação](#como-usar-a-documentação)
- [Endpoints principais](#endpoints-principais)
- [Autenticação](#autenticação)
- [Exemplos de uso](#exemplos-de-uso)

## 📝 Swagger/OpenAPI Specification

A especificação completa da API está disponível em:
- **Arquivo**: [`swagger.yaml`](./swagger.yaml)
- **Formato**: OpenAPI 3.0.3
- **URL local**: http://localhost:8080/api/v1/ (quando a API estiver rodando)

## 🔧 Como usar a documentação

### 1. Swagger UI Online

Acesse o [Swagger Editor](https://editor.swagger.io/) e cole o conteúdo do arquivo `swagger.yaml` para visualizar a documentação interativa.

### 2. Swagger UI Local

Você pode rodar o Swagger UI localmente usando Docker:

```bash
# Na raiz do projeto
docker run -p 8081:8080 -e SWAGGER_JSON=/docs/swagger.yaml -v $(pwd)/docs:/docs swaggerapi/swagger-ui
```

Acesse: http://localhost:8081

### 3. VS Code

Se você usa VS Code, instale a extensão "Swagger Viewer" para visualizar o arquivo YAML diretamente no editor.

## 🚀 Endpoints principais

### Health & System
- `GET /health` - Health check
- `GET /ready` - Readiness check
- `GET /metrics` - Métricas Prometheus
- `GET /info` - Informações do sistema

### Pods
- `GET /namespaces/{namespace}/pods` - Listar pods
- `GET /namespaces/{namespace}/pods/{pod}` - Detalhes do pod
- `POST /namespaces/{namespace}/pods/{pod}/restart` - Reiniciar pod
- `DELETE /namespaces/{namespace}/pods/{pod}` - Deletar pod

### Deployments
- `GET /namespaces/{namespace}/deployments` - Listar deployments
- `POST /namespaces/{namespace}/deployments/{deployment}/scale` - Escalar deployment

## 🔐 Autenticação

A API suporta duas formas de autenticação:

### 1. API Key (Header)
```bash
curl -H "X-API-Key: your-api-key" http://localhost:8080/api/v1/namespaces/default/pods
```

### 2. Bearer Token (JWT)
```bash
curl -H "Authorization: Bearer your-jwt-token" http://localhost:8080/api/v1/namespaces/default/pods
```

### 3. User Identification (Auditoria)
Para auditoria, inclua o header `X-User`:
```bash
curl -H "X-User: admin@company.com" http://localhost:8080/api/v1/namespaces/default/pods
```

## 💡 Exemplos de uso

### Listar pods
```bash
curl http://localhost:8080/api/v1/namespaces/kube-system/pods
```

### Obter detalhes de um pod
```bash
curl http://localhost:8080/api/v1/namespaces/default/pods/my-app-123
```

### Reiniciar um pod
```bash
curl -X POST \
  -H "X-User: admin@company.com" \
  http://localhost:8080/api/v1/namespaces/default/pods/my-app-123/restart
```

### Escalar deployment (parar - 0 réplicas)
```bash
curl -X POST \
  -H "Content-Type: application/json" \
  -H "X-User: admin@company.com" \
  -d '{"replicas": 0}' \
  http://localhost:8080/api/v1/namespaces/default/deployments/my-app/scale
```

### Escalar deployment (iniciar - 3 réplicas)
```bash
curl -X POST \
  -H "Content-Type: application/json" \
  -H "X-User: admin@company.com" \
  -d '{"replicas": 3}' \
  http://localhost:8080/api/v1/namespaces/default/deployments/my-app/scale
```

### Deletar um pod
```bash
curl -X DELETE \
  -H "X-User: admin@company.com" \
  http://localhost:8080/api/v1/namespaces/default/pods/my-app-123
```

## 📊 Códigos de resposta

- **200** - Sucesso
- **400** - Requisição inválida (dados malformados)
- **404** - Recurso não encontrado
- **500** - Erro interno do servidor
- **503** - Serviço indisponível (readiness check)

## 🏷️ Rate Limiting

A API possui rate limiting configurável:
- **Padrão**: 100 requisições por minuto por IP
- **Headers de resposta**:
  - `X-RateLimit-Limit`: Limite configurado
  - `X-RateLimit-Remaining`: Requisições restantes
  - `X-RateLimit-Reset`: Timestamp do reset

## 🔍 CORS

CORS está habilitado por padrão:
- **Origens permitidas**: Configurável via `ALLOWED_ORIGINS` (padrão: `*`)
- **Métodos**: GET, POST, PUT, DELETE, OPTIONS
- **Headers**: Origin, Content-Type, Accept, Authorization, X-Request-ID, X-User

## 📈 Monitoramento

### Métricas Prometheus

A API expõe métricas no endpoint `/metrics`:

```
# HELP database_connections_open Number of open connections
# TYPE database_connections_open gauge
database_connections_open 1

# HELP app_uptime_seconds Application uptime in seconds
# TYPE app_uptime_seconds gauge
app_uptime_seconds 3600.0

# HELP app_info Application info
# TYPE app_info gauge
app_info{version="1.0.0"} 1
```

### Health Checks

- **Liveness**: `/health` - Verifica se a app está rodando
- **Readiness**: `/ready` - Verifica dependências (DB + K8s)

## 🗂️ Estrutura de resposta

### Sucesso
```json
{
  "message": "Operation completed successfully",
  "data": { ... },
  "timestamp": "2025-09-26T20:51:39.711349-03:00"
}
```

### Erro
```json
{
  "error": "Resource not found",
  "code": 404,
  "path": "/api/v1/namespaces/default/pods/non-existent",
  "method": "GET",
  "request_id": "req-123-abc-456"
}
```

## 📝 Auditoria

Todas as operações são auditadas no PostgreSQL:
- **Tabela**: `audit_logs`
- **Dados**: Usuário, IP, timestamp, operação, resultado
- **Retenção**: Configurável

## 🔧 Desenvolvimento

Para contribuir com a documentação:

1. Edite o arquivo `swagger.yaml`
2. Valide a sintaxe no [Swagger Editor](https://editor.swagger.io/)
3. Teste os endpoints documentados
4. Atualize os exemplos se necessário

## 📞 Suporte

- **Issues**: [GitHub Issues](https://github.com/carlosf/k8s-pod-manager/issues)
- **Email**: suporte@company.com
- **Slack**: #k8s-pod-manager