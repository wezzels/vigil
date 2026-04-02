# VIMI Bare Metal Deployment Guide

**VIMI** can be deployed on bare metal (physical servers or VMs) without Docker or Kubernetes. This guide covers deploying VIMI directly on Linux using systemd services.

---

## Table of Contents

1. [Architecture](#1-architecture)
2. [Prerequisites](#2-prerequisites)
3. [Quick Start](#3-quick-start)
4. [Step-by-Step Deployment](#4-step-by-step-deployment)
5. [Service Configuration](#5-service-configuration)
6. [Kafka Setup](#6-kafka-setup)
7. [Networking](#7-networking)
8. [Deployment Scripts](#8-deployment-scripts)
9. [Monitoring](#9-monitoring)
10. [Troubleshooting](#10-troubleshooting)
11. [Production Checklist](#11-production-checklist)

---

## 1. Architecture

### Components

```
┌─────────────────────────────────────────────────────────┐
│                    Bare Metal Host                       │
│                    (Ubuntu 22.04+)                      │
│                                                          │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐ │
│  │opir-     │  │missile-  │  │sensor-   │  │lvc-      │ │
│  │ingest    │→ │warning   │→ │fusion    │→ │coordinator│ │
│  │  :8080   │  │  :8080   │  │  :8082   │  │  :8083   │ │
│  └──────────┘  └──────────┘  └──────────┘  └──────────┘ │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐ │
│  │alert-    │  │env-      │  │replay-   │  │data-     │ │
│  │dissem.   │← │monitor   │  │engine    │  │catalog   │ │
│  │  :8084   │  │  :8085   │  │  :8086   │  │  :8087   │ │
│  └──────────┘  └──────────┘  └──────────┘  └──────────┘ │
│  ┌──────────┐  ┌──────────┐                              │
│  │dis-hla-  │  │vimi-     │                              │
│  │gateway   │  │plugin    │                              │
│  │  :8090   │  │  :8091   │                              │
│  └──────────┘  └──────────┘                              │
│                                                          │
│  ┌──────────────────────────────────────────────────┐   │
│  │  Kafka (KRaft)  │  etcd  │  Redis  │  PostgreSQL  │   │
│  │    :9092        │  :2379 │  :6379  │   :5432      │   │
│  └──────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────┘
```

### Default Ports

| Service | Port | Description |
|---------|------|-------------|
| `opir-ingest` | 8080 | SBIRS IR satellite data ingestion |
| `missile-warning-engine` | 8080 | Missile track detection |
| `sensor-fusion` | 8082 | Multi-sensor track fusion |
| `lvc-coordinator` | 8083 | LVC entity management |
| `alert-dissemination` | 8084 | Alert classification & C2 |
| `env-monitor` | 8085 | Environmental monitoring |
| `replay-engine` | 8086 | DIS PDU recording & playback |
| `data-catalog` | 8087 | OGC CSW asset catalog |
| `dis-hla-gateway` | 8090 | DIS↔HLA protocol bridge |
| `vimi-plugin` | 8091 | Cicerone web UI plugin |

### Kafka Topics

| Topic | Description |
|-------|-------------|
| `vimi.opir.sensor-data` | Raw IR satellite sightings |
| `vimi.tracks` | Detected missile tracks |
| `vimi.fusion.tracks` | Fused multi-sensor tracks |
| `vimi.alerts` | Alert events |
| `vimi.c2.alerts` | C2 system messages |
| `vimi.alert-log` | Alert history |
| `vimi.dis.entity-state` | DIS entity state PDUs |
| `vimi.dis.entity-state-out` | Outbound DIS PDUs |
| `vimi.hla.object-update` | HLA object updates |
| `vimi.hla.interaction` | HLA interactions |
| `vimi.env.events` | Environmental events |
| `vimi.replay.events` | Replay engine events |

---

## 2. Prerequisites

### Hardware

| Resource | Minimum | Recommended |
|----------|---------|-------------|
| CPU | 8 cores | 16+ cores |
| RAM | 16 GB | 32+ GB |
| Disk | 100 GB SSD | 500 GB+ NVMe |
| Network | 1 Gbps | 10 Gbps |

### Software

| Component | Version |
|-----------|---------|
| Ubuntu | 22.04 LTS or later |
| Go | 1.22+ |
| Kafka | 3.7+ (KRaft mode, no Zookeeper) |
| etcd | 3.5+ |
| Redis | 7.0+ |
| PostgreSQL | 15+ |

### Firewall Ports

```bash
# VIMI Services (internal)
ufw allow 8080/tcp   # opir-ingest, missile-warning-engine, dis-hla-gateway, vimi-plugin
ufw allow 8082/tcp   # sensor-fusion
ufw allow 8083/tcp   # lvc-coordinator
ufw allow 8084/tcp   # alert-dissemination
ufw allow 8085/tcp   # env-monitor
ufw allow 8086/tcp   # replay-engine
ufw allow 8087/tcp   # data-catalog
ufw allow 8090/tcp   # dis-hla-gateway
ufw allow 8091/tcp   # vimi-plugin

# Infrastructure
ufw allow 9092/tcp   # Kafka
ufw allow 2379/tcp   # etcd
ufw allow 6379/tcp   # Redis
ufw allow 5432/tcp   # PostgreSQL
```

---

## 3. Quick Start

### One-Command Deploy

```bash
# On a fresh Ubuntu 22.04 host:
curl -sSL https://raw.githubusercontent.com/vimic/trooper-vimi/main/scripts/bare-metal-deploy.sh | bash
```

This script:
1. Installs Go, Kafka, etcd, Redis, PostgreSQL
2. Builds all 10 VIMI services
3. Creates systemd service files
4. Starts all services
5. Creates Kafka topics
6. Verifies deployment

### Manual Deploy (5 Minutes)

```bash
# 1. Install dependencies
sudo apt update && sudo apt install -y golang-go kafka etcd redis-server postgresql

# 2. Build VIMI
git clone https://github.com/vimic/trooper-vimi.git
cd trooper-vimi
make build-all

# 3. Install services
sudo make install-bare-metal

# 4. Start Kafka and infrastructure
sudo systemctl start kafka
sudo systemctl start etcd
sudo systemctl start redis
sudo systemctl start postgresql

# 5. Create Kafka topics
make create-topics

# 6. Start VIMI services
sudo systemctl start vimi-opir-ingest
sudo systemctl start vimi-missile-warning-engine
sudo systemctl start vimi-sensor-fusion
# ... (or start all at once)
sudo systemctl start vimi-@all

# 7. Verify
curl http://localhost:8080/health
curl http://localhost:8084/health
```

---

## 4. Step-by-Step Deployment

### 4.1 Create Deployment User

```bash
# Create dedicated user
sudo useradd -r -s /bin/false vimi 2>/dev/null || true
sudo mkdir -p /opt/vimi /var/log/vimi /var/data/vimi
sudo chown -R vimi:vimi /opt/vimi /var/log/vimi /var/data/vimi
```

### 4.2 Install Go

```bash
# Download Go 1.22+
wget https://go.dev/dl/go1.22.12.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf go1.22.12.linux-amd64.tar.gz
echo 'export PATH=$PATH:/usr/local/go/bin' | sudo tee /etc/profile.d/go.sh
source /etc/profile.d/go.sh
go version  # Should show go1.22
```

### 4.3 Install Infrastructure

#### Kafka (KRaft mode, no Zookeeper)

```bash
# Download Kafka
wget https://downloads.apache.org/kafka/3.7.0/kafka_2.13-3.7.0.tgz
sudo tar -xzf kafka_2.13-3.7.0.tgz -C /opt/
sudo ln -s /opt/kafka_2.13-3.7.0 /opt/kafka

# Format Kafka storage (one-time)
sudo -u vimi /opt/kafka/bin/kafka-storage.sh format \
  -t $(/opt/kafka/bin/kafka-storage.sh random-uuid) \
  -c /opt/kafka/config/kraft/server.properties

# Create systemd service
sudo tee /etc/systemd/system/kafka.service << 'EOF'
[Unit]
Description=Apache Kafka (KRaft)
After=network.target

[Service]
Type=simple
User=vimi
ExecStart=/opt/kafka/bin/kafka-server-start.sh /opt/kafka/config/kraft/server.properties
Restart=on-failure
RestartSec=10
LimitNOFILE=65536

[Install]
WantedBy=multi-user.target
EOF

sudo systemctl daemon-reload
sudo systemctl enable kafka
sudo systemctl start kafka
```

#### etcd

```bash
# Download etcd
wget https://github.com/etcd-io/etcd/releases/download/v3.5.20/etcd-v3.5.20-linux-amd64.tar.gz
sudo tar -xzf etcd-v3.5.20-linux-amd64.tar.gz -C /opt/
sudo ln -s /opt/etcd-v3.5.20-linux-amd64 /opt/etcd

# Create systemd service
sudo tee /etc/systemd/system/etcd.service << 'EOF'
[Unit]
Description=etcd key-value store
After=network.target

[Service]
Type=simple
User=vimi
ExecStart=/opt/etcd/bin/etcd --data-dir=/var/data/etcd --listen-client-urls=http://0.0.0.0:2379 --advertise-client-urls=http://localhost:2379
Restart=on-failure
RestartSec=10

[Install]
WantedBy=multi-user.target
EOF

sudo systemctl daemon-reload
sudo systemctl enable etcd
sudo systemctl start etcd
```

#### Redis

```bash
sudo apt install -y redis-server

# Configure Redis for VIMI
sudo tee /etc/redis/redis.conf << 'EOF'
bind 0.0.0.0
port 6379
maxmemory 512mb
maxmemory-policy allkeys-lru
save ""
EOF

sudo systemctl restart redis
```

#### PostgreSQL

```bash
sudo apt install -y postgresql

# Create VIMI database
sudo -u postgres psql << 'EOF'
CREATE USER vimi WITH PASSWORD 'vimi_secure_password';
CREATE DATABASE vimi OWNER vimi;
GRANT ALL PRIVILEGES ON DATABASE vimi TO vimi;
EOF
```

### 4.4 Build VIMI Services

```bash
# Clone repository
git clone https://github.com/vimic/trooper-vimi.git /opt/vimi
cd /opt/vimi

# Build all services
export REGISTRY=local
make build-all

# Binaries are built at:
# /opt/vimi/bin/opir-ingest
# /opt/vimi/bin/missile-warning-engine
# /opt/vimi/bin/sensor-fusion
# ... etc
```

### 4.5 Create Systemd Service Files

```bash
# Create base directories
sudo mkdir -p /opt/vimi/services/{opir-ingest,missile-warning-engine,sensor-fusion,lvc-coordinator,alert-dissemination,env-monitor,replay-engine,data-catalog,dis-hla-gateway,vimi-plugin}
sudo chown -R vimi:vimi /opt/vimi/services

# Generate service files from template
for service in opir-ingest missile-warning-engine sensor-fusion lvc-coordinator alert-dissemination env-monitor replay-engine data-catalog dis-hla-gateway vimi-plugin; do
  port=$(grep "^${service}:" VIMI-SERVICES.md | cut -d: -f2) 2>/dev/null || {
    case $service in
      opir-ingest) port=8080 ;;
      missile-warning-engine) port=8080 ;;
      sensor-fusion) port=8082 ;;
      lvc-coordinator) port=8083 ;;
      alert-dissemination) port=8084 ;;
      env-monitor) port=8085 ;;
      replay-engine) port=8086 ;;
      data-catalog) port=8087 ;;
      dis-hla-gateway) port=8090 ;;
      vimi-plugin) port=8091 ;;
    esac
  }
  
  cat << EOF | sudo tee /etc/systemd/system/vimi-${service}.service
[Unit]
Description=VIMI ${service}
After=network.target kafka.service
Wants=kafka.service

[Service]
Type=simple
User=vimi
WorkingDirectory=/opt/vimi/services/${service}
Environment="PORT=${port}"
Environment="KAFKA_BROKERS=localhost:9092"
Environment="ETCD_ENDPOINTS=localhost:2379"
Environment="REDIS_ADDR=localhost:6379"
Environment="POSTGRES_URL=postgres://vimi:vimi_secure_password@localhost:5432/vimi"
ExecStart=/opt/vimi/bin/${service}
Restart=on-failure
RestartSec=5
StandardOutput=journal
StandardError=journal
SyslogIdentifier=vimi-${service}

[Install]
WantedBy=multi-user.target
EOF
done

sudo systemctl daemon-reload
```

### 4.6 Create Kafka Topics

```bash
# Create all VIMI Kafka topics
TOPICS=(
  "vimi.opir.sensor-data:3:1"
  "vimi.tracks:3:1"
  "vimi.fusion.tracks:3:1"
  "vimi.alerts:3:1"
  "vimi.c2.alerts:3:1"
  "vimi.alert-log:3:1"
  "vimi.dis.entity-state:3:1"
  "vimi.dis.entity-state-out:3:1"
  "vimi.hla.object-update:3:1"
  "vimi.hla.interaction:3:1"
  "vimi.env.events:3:1"
  "vimi.replay.events:3:1"
)

KAFKA_BIN="/opt/kafka/bin/kafka-topics.sh --bootstrap-server localhost:9092"

for spec in "${TOPICS[@]}"; do
  topic="${spec%%:*}"
  parts="${spec##*:}"
  sudo -u vimi $KAFKA_BIN --create --topic "$topic" \
    --partitions "${spec%%:*}" \
    --replication-factor "${spec##*:}" \
    --if-not-exists 2>/dev/null
  echo "Ensured: $topic"
done
```

### 4.7 Start All Services

```bash
# Start infrastructure first
sudo systemctl start kafka etcd redis postgresql

# Wait for Kafka to be ready
sleep 10

# Start all VIMI services
for service in opir-ingest missile-warning-engine sensor-fusion lvc-coordinator \
               alert-dissemination env-monitor replay-engine data-catalog \
               dis-hla-gateway vimi-plugin; do
  sudo systemctl enable vimi-${service}
  sudo systemctl start vimi-${service}
  echo "Started: vimi-${service}"
done

# Verify all running
sleep 5
systemctl status vimi-opir-ingest --no-pager
```

---

## 5. Service Configuration

### Environment Variables

Each service accepts these environment variables:

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | 8080 | HTTP listen port |
| `KAFKA_BROKERS` | localhost:9092 | Kafka broker addresses |
| `ETCD_ENDPOINTS` | localhost:2379 | etcd endpoints |
| `REDIS_ADDR` | localhost:6379 | Redis address |
| `POSTGRES_URL` | — | PostgreSQL connection URL |
| `DIS_SITE_ID` | 1 | DIS site identifier |
| `DIS_APP_ID` | varies | DIS application identifier |
| `LOG_LEVEL` | info | Logging level (debug/info/warn/error) |

### Service-Specific Variables

#### opir-ingest
```bash
export SATELLITE_SIMULATION=true
export SIGHTING_INTERVAL_MS=2000
```

#### missile-warning-engine
```bash
export DETECTION_THRESHOLD=0.85
export TRACK_TIMEOUT_SECONDS=300
```

#### alert-dissemination
```bash
export NCA_REQUIRED_FOR_HOSTILE=true
export JTIDS_NET_1=239.1.2.3
export JTIDS_NET_2=239.1.2.4
```

### Runtime Overrides

```bash
# Override via systemd drop-in
sudo systemctl edit vimi-opir-ingest

# Add:
# [Service]
# Environment="PORT=8080"
# Environment="KAFKA_BROKERS=kafka1:9092,kafka2:9092"
```

---

## 6. Kafka Setup

### KRaft Mode (Recommended)

Kafka 3.5+ supports KRaft mode — no Zookeeper needed:

```bash
# Generate cluster ID
CLUSTER_ID=$(/opt/kafka/bin/kafka-storage.sh random-uuid)

# Format storage
/opt/kafka/bin/kafka-storage.sh format \
  -t $CLUSTER_ID \
  -c /opt/kafka/config/kraft/server.properties

# Start Kafka
/opt/kafka/bin/kafka-server-start.sh \
  /opt/kafka/config/kraft/server.properties
```

### Kafka Configuration

```properties
# /opt/kafka/config/kraft/server.properties
process.roles=controller,broker
node.id=1
controller.quorum.voters=1@localhost:9093
inter.broker.listener.name=PLAINTEXT
listeners=PLAINTEXT://:9092,CONTROLLER://:9093
advertised.listeners=PLAINTEXT://localhost:9092
listener.security.protocol.map=CONTROLLER:PLAINTEXT,PLAINTEXT:PLAINTEXT
controller.listener.names=CONTROLLER
log.dirs=/var/data/kafka
num.partitions=3
default.replication.factor=1
min.insync.replicas=1
auto.create.topics.enable=true
```

### Multi-Broker Setup

For HA, deploy multiple Kafka brokers:

```bash
# Broker 1 (node1)
node.id=1
listeners=PLAINTEXT://node1:9092,CONTROLLER://node1:9093
advertised.listeners=PLAINTEXT://node1:9092
controller.quorum.voters=1@node1:9093,2@node2:9093,3@node3:9093

# Broker 2 (node2)
node.id=2
listeners=PLAINTEXT://node2:9092,CONTROLLER://node2:9093
advertised.listeners=PLAINTEXT://node2:9092
controller.quorum.voters=1@node1:9093,2@node2:9093,3@node3:9093

# Broker 3 (node3)
node.id=3
listeners=PLAINTEXT://node3:9092,CONTROLLER://node3:9093
advertised.listeners=PLAINTEXT://node3:9092
controller.quorum.voters=1@node1:9093,2@node2:9093,3@node3:9093
```

---

## 7. Networking

### Interface Binding

By default, VIMI services bind to `0.0.0.0` (all interfaces). To restrict:

```bash
# Bind to specific interface
export LISTEN_ADDR=10.0.0.1  # VIMI internal IP

# Or via command line (if supported)
vimi-opir-ingest --listen 10.0.0.1:8080
```

### Firewall Configuration

```bash
# UFW on Ubuntu
sudo ufw default deny incoming
sudo ufw allow ssh
sudo ufw allow 8080/tcp
sudo ufw allow 8082/tcp
sudo ufw allow 8083/tcp
sudo ufw allow 8084/tcp
sudo ufw allow 8085/tcp
sudo ufw allow 8086/tcp
sudo ufw allow 8087/tcp
sudo ufw allow 8090/tcp
sudo ufw allow 8091/tcp
sudo ufw allow 9092/tcp
sudo ufw allow from 10.0.0.0/24  # VIMI internal network
sudo ufw enable
```

### TLS/HTTPS (Optional)

```bash
# Generate self-signed certificate
sudo openssl req -x509 -nodes -days 365 \
  -newkey rsa:2048 \
  -keyout /etc/ssl/private/vimi.key \
  -out /etc/ssl/certs/vimi.crt

# Configure nginx reverse proxy for HTTPS
sudo apt install -y nginx
sudo tee /etc/nginx/sites-available/vimi << 'EOF'
server {
    listen 443 ssl;
    server_name vimi.local;

    ssl_certificate /etc/ssl/certs/vimi.crt;
    ssl_certificate_key /etc/ssl/private/vimi.key;

    location / {
        proxy_pass http://localhost:8081;
    }
}
EOF
```

---

## 8. Deployment Scripts

### Main Deploy Script

Located at: [`scripts/bare-metal-deploy.sh`](./bare-metal-deploy.sh)

Usage:
```bash
# Full deploy
sudo ./scripts/bare-metal-deploy.sh deploy

# Deploy infrastructure only
sudo ./scripts/bare-metal-deploy.sh infra

# Deploy VIMI only
sudo ./scripts/bare-metal-deploy.sh vimi

# Verify deployment
sudo ./scripts/bare-metal-deploy.sh verify

# Stop all services
sudo ./scripts/bare-metal-deploy.sh stop

# Start all services
sudo ./scripts/bare-metal-deploy.sh start
```

### Individual Service Management

```bash
# Start/stop individual service
sudo systemctl start vimi-opir-ingest
sudo systemctl stop vimi-missile-warning-engine

# View logs
sudo journalctl -u vimi-opir-ingest -f
sudo journalctl -u vimi-missile-warning-engine -f

# Restart with config reload
sudo systemctl restart vimi-opir-ingest
```

### Health Check Script

```bash
#!/bin/bash
# scripts/vimi-health-check.sh

SERVICES=(
  "opir-ingest:8080"
  "missile-warning-engine:8080"
  "sensor-fusion:8082"
  "lvc-coordinator:8083"
  "alert-dissemination:8084"
  "env-monitor:8085"
  "replay-engine:8086"
  "data-catalog:8087"
  "dis-hla-gateway:8090"
  "vimi-plugin:8091"
)

ALL_OK=true
for svc in "${SERVICES[@]}"; do
  name="${svc%%:*}"
  port="${svc##*:}"
  if curl -sf http://localhost:$port/health > /dev/null 2>&1; then
    echo "✓ $name"
  else
    echo "✗ $name FAILED"
    ALL_OK=false
  fi
done

# Check Kafka
if /opt/kafka/bin/kafka-topics.sh --bootstrap-server localhost:9092 --list > /dev/null 2>&1; then
  echo "✓ Kafka"
else
  echo "✗ Kafka FAILED"
  ALL_OK=false
fi

# Check etcd
if etcdctl --endpoints=localhost:2379 endpoint health > /dev/null 2>&1; then
  echo "✓ etcd"
else
  echo "✗ etcd FAILED"
  ALL_OK=false
fi

$ALL_OK && echo "All systems operational" || echo "Some systems failed"
```

---

## 9. Monitoring

### Built-in Health Endpoints

```bash
# Check all services
for port in 8080 8082 8083 8084 8085 8086 8087 8090 8091; do
  curl -s http://localhost:$port/health && echo " :$port OK" || echo " :$port FAIL"
done
```

### Prometheus Metrics

All services expose `/metrics`:

```bash
# View metrics
curl http://localhost:8080/metrics

# Prometheus scrape config
cat << 'EOF'
scrape_configs:
  - job_name: 'vimi'
    static_configs:
      - targets:
        - 'localhost:8080'  # opir-ingest
        - 'localhost:8082'  # sensor-fusion
        - 'localhost:8083'  # lvc-coordinator
        - 'localhost:8084'  # alert-dissemination
        - 'localhost:8085'  # env-monitor
        - 'localhost:8086'  # replay-engine
        - 'localhost:8087'  # data-catalog
        - 'localhost:8090'  # dis-hla-gateway
        - 'localhost:8091'  # vimi-plugin
```

### Log Aggregation

```bash
# View all VIMI logs
sudo journalctl -u vimi-\* -f

# Filter by severity
sudo journalctl -u vimi-opir-ingest -p err -f

# Export logs
sudo journalctl -u vimi-\* --since "1 hour ago" > /var/log/vimi/audit.log
```

---

## 10. Troubleshooting

### Service Won't Start

```bash
# Check status
sudo systemctl status vimi-opir-ingest

# View logs
sudo journalctl -u vimi-opir-ingest -n 50 --no-pager

# Check port conflict
sudo ss -tlnp | grep 8080

# Verify binary
ls -la /opt/vimi/bin/opir-ingest
file /opt/vimi/bin/opir-ingest
```

### Kafka Connection Failed

```bash
# Check Kafka is running
systemctl status kafka

# Test Kafka
/opt/kafka/bin/kafka-topics.sh --bootstrap-server localhost:9092 --list

# Check logs
journalctl -u kafka -n 50
```

### Database Connection Failed

```bash
# Check PostgreSQL
systemctl status postgresql
sudo -u postgres psql -c "SELECT 1"

# Test connection
psql "postgres://vimi:vimi_secure_password@localhost:5432/vimi" -c "SELECT 1"
```

### High Memory Usage

```bash
# Check memory per service
ps aux --sort=-%mem | grep vimi

# Check system memory
free -h

# Adjust service memory limits in systemd service file
MemoryMax=512M
MemoryHigh=384M
```

---

## 11. Production Checklist

### Security

- [ ] Run as non-root user (`vimi`)
- [ ] Firewall configured (only required ports exposed)
- [ ] PostgreSQL password set (not default)
- [ ] Redis password set (if accessible externally)
- [ ] Kafka authentication enabled (SASL/SSL) if exposed
- [ ] etcd authentication enabled if exposed
- [ ] TLS configured for external access
- [ ] Regular security updates enabled

### Reliability

- [ ] All services enabled in systemd (auto-start on boot)
- [ ] Kafka replication factor set appropriately
- [ ] Redis persistence enabled (AOF)
- [ ] PostgreSQL WAL configured
- [ ] Monitoring/alerting set up
- [ ] Log rotation configured
- [ ] Backup strategy in place

### Performance

- [ ] Kafka log retention configured
- [ ] Service memory limits set appropriately
- [ ] Network interface binding verified
- [ ] Kernel parameters tuned (`sysctl`)
- [ ] Disk I/O scheduler appropriate (SSD/NVMe)

### Operational

- [ ] Runbooks written for common failures
- [ ] Deployment documented
- [ ] Rollback procedure tested
- [ ] Capacity planning done
- [ ] Network diagrams updated

---

## Quick Reference

```bash
# Status
systemctl status 'vimi-*'

# Logs
journalctl -u vimi-opir-ingest -f

# Restart all
systemctl restart 'vimi-@all'  # if target installed

# Health
curl http://localhost:8080/health

# Kafka topics
/opt/kafka/bin/kafka-topics.sh --bootstrap-server localhost:9092 --list

# Stop all
systemctl stop 'vimi-*'
```
