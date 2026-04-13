# VIGIL Installation Guide

## System Requirements

### Hardware Requirements

| Component | Minimum | Recommended |
|-----------|---------|-------------|
| CPU | 4 cores (x86_64) | 8+ cores |
| Memory | 8 GB RAM | 16+ GB RAM |
| Storage | 50 GB SSD | 100+ GB NVMe SSD |
| Network | 1 Gbps | 10+ Gbps |

### Operating Systems

- **Ubuntu**: 22.04 LTS (recommended)
- **RHEL**: 9.x
- **CentOS**: Stream 9
- **Debian**: 12 (bookworm)

### Software Dependencies

| Software | Version | Purpose |
|----------|---------|---------|
| Docker | 24.0+ | Container runtime |
| Docker Compose | 2.20+ | Multi-container orchestration |
| Kubernetes | 1.28+ | Production orchestration |
| Helm | 3.12+ | Kubernetes package manager |
| Go | 1.21+ | Build from source |
| Make | 4.3+ | Build automation |

## Quick Installation

### Option 1: Docker Compose (Development)

```bash
# Install Docker
curl -fsSL https://get.docker.com | sh
sudo usermod -aG docker $USER

# Install Docker Compose
sudo curl -L "https://github.com/docker/compose/releases/download/v2.20.0/docker-compose-$(uname -s)-$(uname -m)" -o /usr/local/bin/docker-compose
sudo chmod +x /usr/local/bin/docker-compose

# Clone repository
git clone https://github.com/wezzels/vigil.git
cd vigil

# Start services
docker compose -f docker-compose.local.yaml up -d

# Verify installation
curl http://localhost:8080/health
```

### Option 2: Kubernetes (Production)

```bash
# Install kubectl
curl -LO "https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/linux/amd64/kubectl"
sudo install -o root -g root -m 0755 kubectl /usr/local/bin/kubectl

# Install Helm
curl https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-3 | bash

# Add VIGIL Helm repository
helm repo add vigil https://charts.wezzel.com/vigil
helm repo update

# Create namespace
kubectl create namespace vigil

# Install VIGIL
helm install vigil vigil/vigil \
  --namespace vigil \
  --set kafka.brokers=kafka:9092 \
  --set redis.url=redis://redis:6379
```

### Option 3: Build from Source

```bash
# Install Go
wget https://go.dev/dl/go1.21.0.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf go1.21.0.linux-amd64.tar.gz
export PATH=$PATH:/usr/local/go/bin

# Clone repository
git clone https://github.com/wezzels/vigil.git
cd vigil

# Download dependencies
go mod download

# Build
make build

# Run tests
make test

# Install binaries
sudo make install
```

## Configuration

### Environment Variables

Create a `.env` file in the project root:

```bash
# .env

# Kafka Configuration
KAFKA_BROKERS=kafka:9092
KAFKA_GROUP_ID=vigil
KAFKA_TOPIC_OPIR=opir-detections
KAFKA_TOPIC_RADAR=radar-tracks
KAFKA_TOPIC_TRACKS=track-updates
KAFKA_TOPIC_CORRELATED=correlated-tracks
KAFKA_TOPIC_ALERTS=alerts

# Redis Configuration
REDIS_URL=redis://redis:6379
REDIS_PASSWORD=

# PostgreSQL Configuration
DATABASE_URL=postgresql://vigil:vigil@postgres:5432/vigil
DATABASE_POOL_SIZE=10

# Logging
LOG_LEVEL=info
LOG_FORMAT=json

# Service Ports
OPIR_PORT=8080
FUSION_PORT=8081
WARNING_PORT=8082
ALERT_PORT=8083
LVC_PORT=8084
REPLAY_PORT=8085
METRICS_PORT=9090

# Performance
MAX_TRACKS=10000
MAX_GOROUTINES=1000
PROCESSING_TIMEOUT=100ms
```

### Configuration Files

#### Kafka Configuration (`config/kafka.yaml`)

```yaml
brokers:
  - kafka-0:9092
  - kafka-1:9092
  - kafka-2:9092

consumer:
  group_id: vigil
  auto_offset_reset: earliest
  enable_auto_commit: true

producer:
  acks: all
  retries: 3
  batch_size: 16384
  linger_ms: 5

topics:
  opir:
    name: opir-detections
    partitions: 12
    replication: 3
  radar:
    name: radar-tracks
    partitions: 12
    replication: 3
  correlated:
    name: correlated-tracks
    partitions: 6
    replication: 3
```

#### Redis Configuration (`config/redis.yaml`)

```yaml
url: redis://redis:6379
pool_size: 100
min_idle_conns: 10
max_retries: 3
dial_timeout: 5s
read_timeout: 3s
write_timeout: 3s
```

#### PostgreSQL Configuration (`config/postgres.yaml`)

```yaml
url: postgresql://vigil:vigil@postgres:5432/vigil
max_open_conns: 100
max_idle_conns: 10
conn_max_lifetime: 1h
conn_max_idle_time: 10m
```

## Service Configuration

### OPIR Ingest

```yaml
# apps/opir-ingest/config.yaml
server:
  port: 8080
  read_timeout: 30s
  write_timeout: 30s

kafka:
  topic: opir-detections
  batch_size: 100
  flush_interval: 100ms

processing:
  max_detections_per_batch: 1000
  confidence_threshold: 0.5
  duplicate_window: 60s
```

### Sensor Fusion

```yaml
# apps/sensor-fusion/config.yaml
server:
  port: 8081

kafka:
  input_topics:
    - opir-detections
    - radar-tracks
  output_topic: correlated-tracks

fusion:
  association_gate: 3.0  # Mahalanobis distance threshold
  max_tracks: 10000
  track_lifetime: 60s
  kalman:
    process_noise: 1.0
    measurement_noise: 0.1
```

### Missile Warning Engine

```yaml
# apps/missile-warning-engine/config.yaml
server:
  port: 8082

doctrine:
  alert_rules:
    - name: CONOPREP
      confidence: 0.5
      time_to_impact: 300
    - name: IMMINENT
      confidence: 0.7
      time_to_impact: 120
    - name: INCOMING
      confidence: 0.85
      time_to_impact: 30
    - name: HOSTILE
      confidence: 0.95
      time_to_impact: 10

threat_types:
  - BALLISTIC
  - CRUISE
  - AIRCRAFT
  - UAV
  - ARTILLERY
```

### LVC Coordinator

```yaml
# apps/lvc-coordinator/config.yaml
server:
  port: 8084

dis:
  multicast_group: 224.0.0.1
  port: 3000
  exercise_id: 1
  site_id: 1
  application_id: 1

dead_reckoning:
  default_model: DRM_RPW
  update_interval: 100ms
  position_threshold: 1.0
```

## Infrastructure Setup

### Kafka Cluster

```bash
# Start Kafka with Docker Compose
docker compose -f docker-compose.local.yaml up -d kafka

# Create topics
kafka-topics.sh --create --topic opir-detections \
  --bootstrap-server localhost:9092 \
  --partitions 12 --replication-factor 3

kafka-topics.sh --create --topic radar-tracks \
  --bootstrap-server localhost:9092 \
  --partitions 12 --replication-factor 3

kafka-topics.sh --create --topic correlated-tracks \
  --bootstrap-server localhost:9092 \
  --partitions 6 --replication-factor 3
```

### Redis

```bash
# Start Redis
docker compose -f docker-compose.local.yaml up -d redis

# Test connection
redis-cli -h localhost ping
```

### PostgreSQL

```bash
# Start PostgreSQL
docker compose -f docker-compose.local.yaml up -d postgres

# Initialize database
psql -h localhost -U vigil -d vigil -f scripts/init.sql
```

## Verification

### Health Checks

```bash
# Check all services
make health-check

# Individual service checks
curl http://localhost:8080/health  # OPIR Ingest
curl http://localhost:8081/health  # Sensor Fusion
curl http://localhost:8082/health  # Missile Warning
curl http://localhost:8083/health  # Alert Dissemination
curl http://localhost:8084/health  # LVC Coordinator
curl http://localhost:8085/health  # Replay Engine
```

### Metrics

```bash
# Get Prometheus metrics
curl http://localhost:8080/metrics | grep vigil

# Expected output:
# vigil_tracks_processed_total{service="sensor-fusion"} 12345
# vigil_processing_latency_ms{service="sensor-fusion"} 15.5
# vigil_alerts_generated_total{service="missile-warning"} 42
```

### Integration Tests

```bash
# Run integration tests
make test-integration

# Run all tests
make test-all
```

## Upgrading

### Docker Compose

```bash
# Pull latest images
docker compose pull

# Restart services
docker compose up -d
```

### Kubernetes

```bash
# Update Helm repository
helm repo update

# Upgrade release
helm upgrade vigil vigil/vigil --namespace vigil
```

## Uninstalling

### Docker Compose

```bash
# Stop and remove containers
docker compose down

# Remove volumes
docker compose down -v

# Remove images
docker rmi $(docker images 'vigil*' -q)
```

### Kubernetes

```bash
# Uninstall Helm release
helm uninstall vigil --namespace vigil

# Delete namespace
kubectl delete namespace vigil
```

### Source Installation

```bash
# Uninstall binaries
sudo rm /usr/local/bin/vigil

# Remove data directories
rm -rf ~/.vigil
rm -rf /var/lib/vigil
```