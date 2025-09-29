# 🚀 Exemplos de Uso - K8s Pod Manager API

Este documento contém exemplos práticos de como usar a API K8s Pod Manager.

## 📋 Índice

- [Configuração inicial](#configuração-inicial)
- [Health Checks](#health-checks)
- [Operações com Pods](#operações-com-pods)
- [Operações com Deployments](#operações-com-deployments)
- [Cenários avançados](#cenários-avançados)
- [Scripts úteis](#scripts-úteis)

## ⚙️ Configuração inicial

### Variáveis de ambiente
```bash
# URL base da API
export API_BASE="http://localhost:8080/api/v1"

# Usuário para auditoria
export API_USER="admin@company.com"

# Namespace padrão
export NAMESPACE="default"
```

### Função helper para curl
```bash
# Adicione no seu ~/.bashrc ou ~/.zshrc
api_call() {
    local method=${1:-GET}
    local endpoint=$2
    local data=$3

    curl -s \
        -X "$method" \
        -H "Content-Type: application/json" \
        -H "X-User: ${API_USER:-system}" \
        ${data:+-d "$data"} \
        "${API_BASE}${endpoint}"
}
```

## 🏥 Health Checks

### Verificar se a API está rodando
```bash
curl http://localhost:8080/api/v1/health
```

**Resposta esperada:**
```json
{
  "status": "healthy",
  "timestamp": "2025-09-26T20:51:39.711349-03:00",
  "uptime": "1h23m45s",
  "version": "1.0.0"
}
```

### Verificar dependências (DB + Kubernetes)
```bash
curl http://localhost:8080/api/v1/ready
```

**Resposta esperada:**
```json
{
  "status": "ready",
  "checks": {
    "database": "healthy",
    "kubernetes": "healthy"
  },
  "timestamp": "2025-09-26T20:51:49.288825-03:00"
}
```

### Obter métricas Prometheus
```bash
curl http://localhost:8080/api/v1/metrics
```

## 🎯 Operações com Pods

### 1. Listar todos os pods de um namespace
```bash
curl "${API_BASE}/namespaces/default/pods"
```

**Com filtros via jq:**
```bash
# Apenas pods em execução
curl -s "${API_BASE}/namespaces/default/pods" | \
  jq '.pods[] | select(.phase == "Running") | {name, ready, restarts}'

# Pods com problemas
curl -s "${API_BASE}/namespaces/default/pods" | \
  jq '.pods[] | select(.phase != "Running" or .restarts > 0)'
```

### 2. Obter detalhes de um pod específico
```bash
POD_NAME="my-app-7d4b8c9f5-xyz12"
curl "${API_BASE}/namespaces/default/pods/${POD_NAME}"
```

### 3. Reiniciar um pod

#### Pod gerenciado por Deployment
```bash
POD_NAME="my-app-7d4b8c9f5-xyz12"
curl -X POST \
  -H "X-User: ${API_USER}" \
  "${API_BASE}/namespaces/default/pods/${POD_NAME}/restart"
```

**Resposta esperada:**
```json
{
  "message": "Rollout restart initiated",
  "type": "deployment",
  "deployment": "my-app",
  "pod": "my-app-7d4b8c9f5-xyz12",
  "namespace": "default"
}
```

#### Pod gerenciado por StatefulSet
```bash
POD_NAME="postgres-0"
curl -X POST \
  -H "X-User: ${API_USER}" \
  "${API_BASE}/namespaces/default/pods/${POD_NAME}/restart"
```

**Resposta esperada:**
```json
{
  "message": "Rollout restart initiated",
  "type": "statefulset",
  "statefulset": "postgres",
  "pod": "postgres-0",
  "namespace": "default"
}
```

### 4. Deletar um pod
```bash
POD_NAME="my-app-7d4b8c9f5-xyz12"
curl -X DELETE \
  -H "X-User: ${API_USER}" \
  "${API_BASE}/namespaces/default/pods/${POD_NAME}"
```

**Resposta esperada:**
```json
{
  "message": "Pod deleted successfully",
  "pod": "my-app-7d4b8c9f5-xyz12",
  "namespace": "default"
}
```

## 🚀 Operações com Deployments

### 1. Listar deployments
```bash
curl "${API_BASE}/namespaces/default/deployments"
```

**Resposta esperada:**
```json
{
  "namespace": "default",
  "count": 2,
  "deployments": [
    {
      "name": "my-app",
      "namespace": "default",
      "replicas": 3,
      "updated_replicas": 3,
      "ready_replicas": 3,
      "available_replicas": 3,
      "labels": {
        "app": "my-app"
      },
      "created_at": "2025-09-26T20:24:10-03:00"
    }
  ]
}
```

### 2. Escalar deployment

#### Parar aplicação (0 réplicas)
```bash
DEPLOYMENT_NAME="my-app"
curl -X POST \
  -H "Content-Type: application/json" \
  -H "X-User: ${API_USER}" \
  -d '{"replicas": 0}' \
  "${API_BASE}/namespaces/default/deployments/${DEPLOYMENT_NAME}/scale"
```

#### Iniciar aplicação (3 réplicas)
```bash
DEPLOYMENT_NAME="my-app"
curl -X POST \
  -H "Content-Type: application/json" \
  -H "X-User: ${API_USER}" \
  -d '{"replicas": 3}' \
  "${API_BASE}/namespaces/default/deployments/${DEPLOYMENT_NAME}/scale"
```

#### Escalar para alta demanda (10 réplicas)
```bash
DEPLOYMENT_NAME="my-app"
curl -X POST \
  -H "Content-Type: application/json" \
  -H "X-User: ${API_USER}" \
  -d '{"replicas": 10}' \
  "${API_BASE}/namespaces/default/deployments/${DEPLOYMENT_NAME}/scale"
```

**Resposta esperada:**
```json
{
  "message": "Deployment scaled successfully",
  "deployment": "my-app",
  "namespace": "default",
  "previous_replicas": 3,
  "new_replicas": 10,
  "action": "Scaling from 3 to 10 replicas"
}
```

## 🎭 Cenários avançados

### 1. Rolling restart de todos os pods de um deployment
```bash
#!/bin/bash
NAMESPACE="default"
DEPLOYMENT="my-app"

# Obter lista de pods do deployment
PODS=$(curl -s "${API_BASE}/namespaces/${NAMESPACE}/pods" | \
  jq -r ".pods[] | select(.owners[] | contains(\"$DEPLOYMENT\")) | .name")

echo "Rolling restart dos pods do deployment ${DEPLOYMENT}:"
for POD in $PODS; do
  echo "Reiniciando pod: $POD"
  curl -X POST \
    -H "X-User: ${API_USER}" \
    "${API_BASE}/namespaces/${NAMESPACE}/pods/${POD}/restart"
  echo
  sleep 5  # Aguardar entre restarts
done
```

### 2. Monitorar status de pods após restart
```bash
#!/bin/bash
NAMESPACE="default"
DEPLOYMENT="my-app"

echo "Monitorando pods do deployment ${DEPLOYMENT}..."
while true; do
  clear
  echo "=== Status dos Pods - $(date) ==="
  curl -s "${API_BASE}/namespaces/${NAMESPACE}/pods" | \
    jq -r ".pods[] | select(.owners[] | contains(\"$DEPLOYMENT\")) |
           \"\(.name) | \(.phase) | \(.ready) | \(.age)\""
  sleep 5
done
```

### 3. Auto-scaling baseado em métricas
```bash
#!/bin/bash
NAMESPACE="default"
DEPLOYMENT="my-app"
MAX_REPLICAS=10
MIN_REPLICAS=1

while true; do
  # Obter métricas do sistema (exemplo simplificado)
  LOAD=$(uptime | awk '{print $10}' | sed 's/,//')

  # Decidir escala baseado na carga
  if (( $(echo "$LOAD > 2.0" | bc -l) )); then
    REPLICAS=$MAX_REPLICAS
  elif (( $(echo "$LOAD < 0.5" | bc -l) )); then
    REPLICAS=$MIN_REPLICAS
  else
    REPLICAS=3
  fi

  echo "Carga atual: $LOAD - Escalando para $REPLICAS réplicas"
  curl -X POST \
    -H "Content-Type: application/json" \
    -H "X-User: auto-scaler" \
    -d "{\"replicas\": $REPLICAS}" \
    "${API_BASE}/namespaces/${NAMESPACE}/deployments/${DEPLOYMENT}/scale"

  sleep 60  # Verificar a cada minuto
done
```

### 4. Backup de configurações antes de mudanças
```bash
#!/bin/bash
NAMESPACE="default"
DEPLOYMENT="my-app"
BACKUP_DIR="./backups/$(date +%Y%m%d_%H%M%S)"

mkdir -p "$BACKUP_DIR"

echo "Fazendo backup das configurações..."

# Backup do deployment atual
kubectl get deployment "$DEPLOYMENT" -n "$NAMESPACE" -o yaml > \
  "$BACKUP_DIR/deployment_${DEPLOYMENT}.yaml"

# Backup da lista atual de pods
curl -s "${API_BASE}/namespaces/${NAMESPACE}/pods" | \
  jq ".pods[] | select(.owners[] | contains(\"$DEPLOYMENT\"))" > \
  "$BACKUP_DIR/pods_${DEPLOYMENT}.json"

echo "Backup salvo em: $BACKUP_DIR"
```

## 🛠️ Scripts úteis

### 1. Script para restart massivo
```bash
#!/bin/bash
# restart-all.sh - Reinicia todos os pods de vários deployments

NAMESPACE=${1:-default}
DEPLOYMENTS=("app1" "app2" "app3")

for DEPLOYMENT in "${DEPLOYMENTS[@]}"; do
  echo "=== Reiniciando deployment: $DEPLOYMENT ==="

  # Fazer rollout restart via API
  PODS=$(curl -s "${API_BASE}/namespaces/${NAMESPACE}/pods" | \
    jq -r ".pods[] | select(.owners[] | contains(\"$DEPLOYMENT\")) | .name")

  for POD in $PODS; do
    echo "Reiniciando pod: $POD"
    curl -X POST \
      -H "X-User: ${API_USER}" \
      "${API_BASE}/namespaces/${NAMESPACE}/pods/${POD}/restart"
  done

  echo "Aguardando estabilização..."
  sleep 30
done
```

### 2. Script de manutenção programada
```bash
#!/bin/bash
# maintenance.sh - Script para janela de manutenção

NAMESPACE="production"
DEPLOYMENTS=("frontend" "backend" "worker")

echo "=== INICIANDO MANUTENÇÃO ==="

# Escalar tudo para 0
for DEPLOYMENT in "${DEPLOYMENTS[@]}"; do
  echo "Parando $DEPLOYMENT..."
  curl -X POST \
    -H "Content-Type: application/json" \
    -H "X-User: maintenance-script" \
    -d '{"replicas": 0}' \
    "${API_BASE}/namespaces/${NAMESPACE}/deployments/${DEPLOYMENT}/scale"
done

echo "Todos os serviços parados. Execute sua manutenção."
read -p "Pressione ENTER quando terminar a manutenção..."

# Reativar serviços
for DEPLOYMENT in "${DEPLOYMENTS[@]}"; do
  echo "Reativando $DEPLOYMENT..."
  curl -X POST \
    -H "Content-Type: application/json" \
    -H "X-User: maintenance-script" \
    -d '{"replicas": 3}' \
    "${API_BASE}/namespaces/${NAMESPACE}/deployments/${DEPLOYMENT}/scale"
done

echo "=== MANUTENÇÃO CONCLUÍDA ==="
```

### 3. Monitor de saúde
```bash
#!/bin/bash
# health-monitor.sh - Monitor contínuo de saúde

while true; do
  clear
  echo "=== K8s Pod Manager Health Monitor ==="
  echo "Timestamp: $(date)"
  echo

  # Health da API
  HEALTH=$(curl -s "${API_BASE}/health" | jq -r '.status')
  echo "API Status: $HEALTH"

  # Readiness
  READY=$(curl -s "${API_BASE}/ready" | jq -r '.status')
  echo "Readiness: $READY"

  # Métricas básicas
  echo
  echo "=== Métricas ==="
  curl -s "${API_BASE}/metrics" | grep -E "(uptime|connections)"

  echo
  echo "Pressione Ctrl+C para sair..."
  sleep 10
done
```

## 🚨 Tratamento de erros

### Verificar se pod existe antes de operar
```bash
#!/bin/bash
POD_NAME="my-app-123"
NAMESPACE="default"

# Verificar se pod existe
RESPONSE=$(curl -s -w "%{http_code}" \
  "${API_BASE}/namespaces/${NAMESPACE}/pods/${POD_NAME}")

HTTP_CODE=$(echo "$RESPONSE" | tail -n1)
BODY=$(echo "$RESPONSE" | head -n -1)

if [ "$HTTP_CODE" = "200" ]; then
  echo "Pod encontrado. Reiniciando..."
  curl -X POST \
    -H "X-User: ${API_USER}" \
    "${API_BASE}/namespaces/${NAMESPACE}/pods/${POD_NAME}/restart"
else
  echo "Erro: Pod não encontrado (HTTP $HTTP_CODE)"
  echo "$BODY" | jq '.error'
fi
```

### Retry com backoff exponencial
```bash
#!/bin/bash
retry_with_backoff() {
  local max_attempts=5
  local timeout=1
  local attempt=1
  local exitCode=0

  while [[ $attempt -le $max_attempts ]]; do
    "$@"
    exitCode=$?

    if [[ $exitCode == 0 ]]; then
      break
    fi

    echo "Tentativa $attempt falhou. Tentando novamente em $timeout segundos..."
    sleep $timeout
    attempt=$((attempt + 1))
    timeout=$((timeout * 2))
  done

  return $exitCode
}

# Uso
retry_with_backoff curl -X POST \
  -H "X-User: ${API_USER}" \
  "${API_BASE}/namespaces/default/pods/my-app-123/restart"
```

## 📊 Integração com ferramentas de monitoramento

### Prometheus Alert Manager
```yaml
# alerts.yml
groups:
- name: k8s-pod-manager
  rules:
  - alert: PodManagerAPIDown
    expr: up{job="k8s-pod-manager"} == 0
    for: 1m
    labels:
      severity: critical
    annotations:
      summary: "K8s Pod Manager API is down"

  - alert: PodManagerHighLatency
    expr: histogram_quantile(0.95, rate(http_request_duration_seconds_bucket[5m])) > 0.5
    for: 2m
    labels:
      severity: warning
    annotations:
      summary: "K8s Pod Manager API high latency"
```

### Grafana Dashboard Query
```promql
# Uptime da API
up{job="k8s-pod-manager"}

# Conexões de banco
database_connections_open{job="k8s-pod-manager"}

# Rate de requisições
rate(http_requests_total{job="k8s-pod-manager"}[5m])
```

---

💡 **Dica**: Para mais exemplos e casos de uso específicos, consulte a [documentação completa](./README.md) ou abra uma [issue no GitHub](https://github.com/carlosf/k8s-pod-manager/issues).