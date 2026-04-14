# VIGIL Deployment Guide

## Prerequisites

### Hardware Requirements

| Component | CPU | Memory | Storage | Count |
|-----------|-----|--------|---------|-------|
| Application Nodes | 8 cores | 32 GB | 100 GB | 3+ |
| Database Nodes | 4 cores | 16 GB | 500 GB SSD | 3 |
| Kafka Nodes | 4 cores | 16 GB | 500 GB SSD | 3 |

### Software Requirements

| Software | Version | Purpose |
|----------|---------|---------|
| Kubernetes | 1.28+ | Container orchestration |
| Helm | 3.12+ | Package management |
| kubectl | 1.28+ | K8s CLI |
| Docker | 24.0+ | Container runtime |

### Network Requirements

| Port | Protocol | Purpose |
|------|----------|---------|
| 8080 | TCP | HTTP API |
| 8443 | TCP | HTTPS API |
| 9092 | TCP | Kafka |
| 5432 | TCP | PostgreSQL |
| 6379 | TCP | Redis |
| 30720 | UDP | OPIR Data |

---

## Quick Start (Docker Compose)

### 1. Clone Repository

```bash
git clone https://github.com/wezzels/vigil.git
cd vigil
```

### 2. Create Environment File

```bash
cat > .env << EOF
POSTGRES_PASSWORD=your-secure-password
KAFKA_BROKERS=kafka:9092
REDIS_HOST=redis:6379
EOF
```

### 3. Start Services

```bash
docker-compose up -d
```

### 4. Verify

```bash
curl http://localhost:8080/healthz/liveness
```

---

## Kubernetes Deployment

### 1. Create Namespace

```bash
kubectl apply -f k8s/vigil/namespace.yaml
```

### 2. Create Secrets

```bash
kubectl create secret generic vigil-secrets \
  --from-literal=POSTGRES_PASSWORD=your-secure-password \
  --from-literal=KAFKA_SASL_PASSWORD=your-kafka-password \
  -n vigil
```

### 3. Deploy Infrastructure

```bash
# PostgreSQL (Patroni)
kubectl apply -f k8s/vigil/postgres.yaml

# Kafka
kubectl apply -f k8s/vigil/kafka.yaml

# Redis
kubectl apply -f k8s/vigil/redis.yaml
```

### 4. Wait for Infrastructure

```bash
kubectl wait --for=condition=ready pod -l app=postgres -n vigil --timeout=300s
kubectl wait --for=condition=ready pod -l app=kafka -n vigil --timeout=300s
kubectl wait --for=condition=ready pod -l app=redis -n vigil --timeout=300s
```

### 5. Run Database Migrations

```bash
kubectl exec -it postgres-0 -n vigil -- \
  psql -U vigil -f /migrations/001_tracks.up.sql
```

### 6. Deploy Application Services

```bash
kubectl apply -f k8s/vigil/opir-ingest.yaml
kubectl apply -f k8s/vigil/missile-warning.yaml
kubectl apply -f k8s/vigil/sensor-fusion.yaml
kubectl apply -f k8s/vigil/lvc-coordinator.yaml
```

### 7. Configure Autoscaling

```bash
kubectl apply -f k8s/vigil/hpa.yaml
```

### 8. Verify Deployment

```bash
kubectl get pods -n vigil
kubectl get svc -n vigil
kubectl get hpa -n vigil
```

---

## Helm Deployment

### 1. Add Helm Repository

```bash
helm repo add vigil https://charts.vigil.local
helm repo update
```

### 2. Install Chart

```bash
helm install vigil vigil/vigil \
  --namespace vigil \
  --create-namespace \
  --set postgres.password=your-secure-password \
  --set kafka.brokers=3 \
  --set replicas.opirIngest=3 \
  --set replicas.sensorFusion=3 \
  --set replicas.missileWarning=2
```

### 3. Upgrade

```bash
helm upgrade vigil vigil/vigil \
  --namespace vigil \
  --set image.tag=v1.2.0
```

---

## Configuration

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `POSTGRES_HOST` | `postgres` | PostgreSQL host |
| `POSTGRES_PORT` | `5432` | PostgreSQL port |
| `POSTGRES_DB` | `vigil` | Database name |
| `POSTGRES_USER` | `vigil` | Database user |
| `POSTGRES_PASSWORD` | (required) | Database password |
| `KAFKA_BROKERS` | `kafka:9092` | Kafka broker list |
| `KAFKA_TOPIC_PREFIX` | `vigil` | Topic prefix |
| `REDIS_HOST` | `redis` | Redis host |
| `REDIS_PORT` | `6379` | Redis port |
| `LOG_LEVEL` | `info` | Log level |
| `METRICS_PORT` | `9090` | Prometheus metrics port |

### ConfigMap

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: vigil-config
  namespace: vigil
data:
  POSTGRES_HOST: "postgres"
  POSTGRES_PORT: "5432"
  POSTGRES_DB: "vigil"
  KAFKA_BROKERS: "kafka-0.kafka:9092,kafka-1.kafka:9092,kafka-2.kafka:9092"
  REDIS_HOST: "redis"
  LOG_LEVEL: "info"
```

---

## Monitoring

### Prometheus

```bash
kubectl port-forward svc/prometheus 9090:9090 -n monitoring
```

Access: http://localhost:9090

### Grafana

```bash
kubectl port-forward svc/grafana 3000:80 -n monitoring
```

Access: http://localhost:3000 (admin/admin)

### Key Metrics

| Metric | Description | Alert |
|--------|-------------|-------|
| `vigil_tracks_active` | Active tracks | > 10000 |
| `vigil_alerts_pending` | Pending alerts | > 100 |
| `vigil_latency_p99` | P99 latency | > 100ms |
| `vigil_errors_total` | Error count | > 10/min |

---

## Backup & Recovery

### Velero Backup

```bash
# Install Velero
velero install --provider aws --bucket vigil-backups

# Create backup
velero backup create vigil-backup-$(date +%Y%m%d) \
  --include-namespaces vigil

# Restore
velero restore create --from-backup vigil-backup-20260414
```

### Database Backup

```bash
# Manual backup
kubectl exec -it postgres-0 -n vigil -- \
  pg_dump -U vigil vigil > backup.sql

# Restore
kubectl exec -i postgres-0 -n vigil -- \
  psql -U vigil vigil < backup.sql
```

---

## Troubleshooting

### Pods Not Starting

```bash
kubectl describe pod <pod-name> -n vigil
kubectl logs <pod-name> -n vigil --previous
```

### Service Unavailable

```bash
kubectl get endpoints -n vigil
kubectl exec -it <pod> -n vigil -- curl http://<service>:8080/healthz
```

### Database Issues

```bash
kubectl exec -it postgres-0 -n vigil -- \
  psql -U vigil -c "SELECT 1;"
```

---

**Last Updated:** 2026-04-14
**Version:** 1.0