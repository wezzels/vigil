# VIGIL Deployment Guide

## Prerequisites

### System Requirements

| Requirement | Minimum | Recommended |
|-------------|---------|-------------|
| CPU | 4 cores | 8+ cores |
| Memory | 8 GB | 16+ GB |
| Storage | 50 GB SSD | 100+ GB SSD |
| Network | 1 Gbps | 10+ Gbps |

### Software Requirements

- **Operating System**: Ubuntu 22.04 LTS or RHEL 9
- **Docker**: 24.0+
- **Docker Compose**: 2.20+
- **Kubernetes**: 1.28+ (for production)
- **Helm**: 3.12+ (for Kubernetes deployment)

## Quick Start

### Docker Compose (Development)

```bash
# Clone repository
git clone https://github.com/wezzels/vigil.git
cd vigil

# Start infrastructure
docker compose -f docker-compose.local.yaml up -d

# Build services
make build

# Run services
docker compose up -d
```

### Verify Deployment

```bash
# Check service health
curl http://localhost:8080/health
curl http://localhost:8081/health
curl http://localhost:8082/health

# Check metrics
curl http://localhost:8080/metrics
```

## Docker Compose Deployment

### Configuration

Create `.env` file:

```env
# Kafka
KAFKA_BROKERS=kafka:9092

# Redis
REDIS_URL=redis://redis:6379

# PostgreSQL
DATABASE_URL=postgresql://vigil:vigil@postgres:5432/vigil

# Logging
LOG_LEVEL=info

# Service Ports
OPIR_PORT=8080
FUSION_PORT=8081
WARNING_PORT=8082
ALERT_PORT=8083
LVC_PORT=8084
REPLAY_PORT=8085
```

### Start Services

```bash
# Start all services
docker compose up -d

# Start specific service
docker compose up -d opir-ingest

# View logs
docker compose logs -f opir-ingest

# Scale service
docker compose up -d --scale sensor-fusion=3
```

### Stop Services

```bash
# Stop all services
docker compose down

# Stop and remove volumes
docker compose down -v
```

## Kubernetes Deployment

### Namespace

```bash
# Create namespace
kubectl create namespace vigil
```

### Secrets

```bash
# Create secrets
kubectl create secret generic vigil-secrets \
  --from-literal=kafka-password=your-password \
  --from-literal=redis-password=your-password \
  --from-literal=postgres-password=your-password \
  -n vigil
```

### Helm Chart

```bash
# Add Helm repo
helm repo add vigil https://charts.wezzel.com/vigil
helm repo update

# Install
helm install vigil vigil/vigil \
  --namespace vigil \
  --set kafka.brokers=kafka:9092 \
  --set redis.url=redis://redis:6379 \
  --set postgres.url=postgresql://postgres:5432/vigil
```

### Custom Values

Create `values.yaml`:

```yaml
# values.yaml
replicaCount:
  opir-ingest: 2
  sensor-fusion: 3
  missile-warning: 2
  alert-dissem: 2
  lvc-coordinator: 1

image:
  repository: ghcr.io/wezzels/vigil
  tag: latest
  pullPolicy: IfNotPresent

kafka:
  brokers: "kafka-0.kafka:9092,kafka-1.kafka:9092,kafka-2.kafka:9092"
  topics:
    opir: opir-detections
    radar: radar-tracks
    tracks: track-updates
    correlated: correlated-tracks
    alerts: alerts

redis:
  url: "redis://redis:6379"
  password: ""

postgres:
  url: "postgresql://postgres:5432/vigil"
  user: vigil
  password: ""

resources:
  limits:
    cpu: "2"
    memory: "4Gi"
  requests:
    cpu: "500m"
    memory: "1Gi"

ingress:
  enabled: true
  className: nginx
  hosts:
    - host: vigil.example.com
      paths:
        - path: /
          pathType: Prefix
```

### Deploy with Custom Values

```bash
helm install vigil vigil/vigil \
  --namespace vigil \
  -f values.yaml
```

## Configuration Reference

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `KAFKA_BROKERS` | Kafka broker addresses | `kafka:9092` |
| `KAFKA_GROUP_ID` | Consumer group ID | `vigil` |
| `KAFKA_TOPIC_OPIR` | OPIR detections topic | `opir-detections` |
| `REDIS_URL` | Redis connection URL | `redis://redis:6379` |
| `DATABASE_URL` | PostgreSQL connection URL | `postgresql://...` |
| `LOG_LEVEL` | Log level (debug, info, warn, error) | `info` |
| `METRICS_PORT` | Prometheus metrics port | `9090` |
| `HEALTH_PORT` | Health check port | `8080` |

### Kafka Topics

| Topic | Partitions | Replication | Retention |
|-------|------------|-------------|-----------|
| `opir-detections` | 12 | 3 | 7 days |
| `radar-tracks` | 12 | 3 | 7 days |
| `track-updates` | 12 | 3 | 7 days |
| `correlated-tracks` | 6 | 3 | 7 days |
| `alerts` | 6 | 3 | 30 days |
| `c2-messages` | 6 | 3 | 30 days |

### Redis Keys

| Key | Type | TTL | Description |
|-----|------|-----|-------------|
| `track:{id}` | Hash | 1 hour | Track state |
| `alert:{id}` | Hash | 24 hours | Alert data |
| `entity:{id}` | Hash | 5 min | Entity state |

## Monitoring

### Prometheus Configuration

```yaml
# prometheus.yaml
scrape_configs:
  - job_name: 'vigil'
    static_configs:
      - targets:
          - opir-ingest:9090
          - sensor-fusion:9090
          - missile-warning:9090
          - lvc-coordinator:9090
    metrics_path: /metrics
    scrape_interval: 15s
```

### Grafana Dashboard

Import dashboard from `docs/dashboards/vigil-dashboard.json`.

Key metrics to monitor:

| Metric | Description | Alert Threshold |
|--------|-------------|-----------------|
| `vigil_tracks_processed_total` | Total tracks processed | - |
| `vigil_processing_latency_ms` | Processing latency | > 100ms |
| `vigil_alerts_generated_total` | Total alerts generated | - |
| `vigil_kafka_lag` | Kafka consumer lag | > 1000 |
| `vigil_errors_total` | Total errors | > 10/min |

### Alerts

```yaml
# alerting-rules.yaml
groups:
  - name: vigil
    rules:
      - alert: HighProcessingLatency
        expr: vigil_processing_latency_ms > 100
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "High processing latency"

      - alert: HighKafkaLag
        expr: vigil_kafka_lag > 1000
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "High Kafka consumer lag"

      - alert: HighErrorRate
        expr: rate(vigil_errors_total[5m]) > 10
        for: 5m
        labels:
          severity: critical
        annotations:
          summary: "High error rate"
```

## High Availability

### Multi-Node Kafka

```yaml
# kafka-cluster.yaml
apiVersion: kafka.strimzi.io/v1beta2
kind: Kafka
metadata:
  name: vigil-kafka
spec:
  replicas: 3
  listeners:
    - name: plain
      port: 9092
      type: internal
  storage:
    type: jbod
    volumes:
      - id: 0
        type: persistent-claim
        size: 100Gi
        class: fast-ssd
```

### Multi-Node Redis

```yaml
# redis-cluster.yaml
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: redis
spec:
  replicas: 3
  serviceName: redis
  template:
    spec:
      containers:
        - name: redis
          image: redis:7.2-alpine
          command: ["redis-server", "--cluster-enabled", "yes"]
```

### Multi-Node PostgreSQL (Patroni)

```yaml
# postgres-cluster.yaml
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: patroni
spec:
  replicas: 3
  serviceName: patroni
  template:
    spec:
      containers:
        - name: patroni
          image: patroni:latest
          env:
            - name: PATRONI_POSTGRESQL_DATA_DIR
              value: /data/patroni
```

## Troubleshooting

### Common Issues

#### Kafka Connection Failed

```bash
# Check Kafka is running
kubectl get pods -n kafka

# Check Kafka logs
kubectl logs -n kafka kafka-0

# Verify Kafka topics
kubectl exec -n kafka kafka-0 -- kafka-topics.sh --list --bootstrap-server localhost:9092
```

#### Redis Connection Failed

```bash
# Check Redis
redis-cli -h redis ping

# Check Redis logs
kubectl logs -n vigil redis-0

# Verify Redis keys
redis-cli -h redis keys "*"
```

#### High Latency

```bash
# Check processing metrics
curl http://localhost:8081/metrics | grep latency

# Check Kafka lag
kafka-consumer-groups --describe --group vigil --bootstrap-server kafka:9092

# Check resource usage
kubectl top pods -n vigil
```

#### Memory Issues

```bash
# Check memory usage
kubectl top pods -n vigil

# Increase memory limits
helm upgrade vigil vigil/vigil --set resources.limits.memory=8Gi

# Check for memory leaks
curl http://localhost:8081/debug/pprof/heap
```

### Health Check Script

```bash
#!/bin/bash
# health-check.sh

services=("opir-ingest" "sensor-fusion" "missile-warning" "lvc-coordinator" "replay")

for service in "${services[@]}"; do
  response=$(curl -s -o /dev/null -w "%{http_code}" "http://${service}:8080/health")
  if [ "$response" == "200" ]; then
    echo "✓ ${service} healthy"
  else
    echo "✗ ${service} unhealthy (${response})"
  fi
done
```

### Log Aggregation

```bash
# Elasticsearch/Kibana setup
kubectl apply -f https://raw.githubusercontent.com/elastic/k8s-operator/master/config/samples/kb.yaml

# Fluentd daemon set
kubectl apply -f https://raw.githubusercontent.com/fluent/fluentd-kubernetes-daemonset/master/fluentd-daemonset-elasticsearch.yaml
```

## Backup and Recovery

### Backup Script

```bash
#!/bin/bash
# backup.sh

# Backup PostgreSQL
kubectl exec -n vigil postgres-0 -- pg_dump -U vigil vigil > vigil-backup-$(date +%Y%m%d).sql

# Backup Redis
kubectl exec -n vigil redis-0 -- redis-cli BGSAVE
kubectl cp vigil/redis-0:/data/dump.rdb redis-backup-$(date +%Y%m%d).rdb

# Backup Kafka topics
kafka-topics.sh --bootstrap-server kafka:9092 --list > kafka-topics-$(date +%Y%m%d).txt
```

### Recovery

```bash
# Restore PostgreSQL
kubectl exec -i -n vigil postgres-0 -- psql -U vigil vigil < vigil-backup-20240101.sql

# Restore Redis
kubectl cp redis-backup-20240101.rdb vigil/redis-0:/data/dump.rdb
kubectl exec -n vigil redis-0 -- redis-cli SHUTDOWN NOSAVE

# Restore Kafka topics
for topic in $(cat kafka-topics-20240101.txt); do
  kafka-topics.sh --bootstrap-server kafka:9092 --create --topic ${topic} --partitions 12 --replication-factor 3
done
```

## Security

### TLS Configuration

```yaml
# tls.yaml
apiVersion: cert-manager.io/v1
kind: Certificate
metadata:
  name: vigil-tls
spec:
  secretName: vigil-tls
  issuerRef:
    name: letsencrypt
    kind: ClusterIssuer
  dnsNames:
    - vigil.example.com
```

### Network Policies

```yaml
# network-policy.yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: vigil-network
spec:
  podSelector:
    matchLabels:
      app: vigil
  policyTypes:
    - Ingress
    - Egress
  ingress:
    - from:
        - podSelector:
            matchLabels:
              app: vigil
  egress:
    - to:
        - podSelector:
            matchLabels:
              app: kafka
      ports:
        - port: 9092
    - to:
        - podSelector:
            matchLabels:
              app: redis
      ports:
        - port: 6379
```