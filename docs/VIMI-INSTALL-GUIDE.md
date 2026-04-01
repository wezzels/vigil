# VIMI —Installation & Implementation Guide

**VIMI** (VIMI Integrated Mission Infrastructure) is a DoD-grade LVC (Live/Virtual/Constructive) mission processing system built on the STS Gym infrastructure. It processes OPIR satellite data through missile warning, sensor fusion, alert dissemination, and HLA/DIS federation — in real time.

---

## Table of Contents

1. [Overview](#1-overview)
2. [Architecture](#2-architecture)
3. [Prerequisites](#3-prerequisites)
4. [Repository Structure](#4-repository-structure)
5. [Step 1 — Base Infrastructure](#step-1--base-infrastructure)
6. [Step 2 — Build & Load Container Images](#step-2--build--load-container-images)
7. [Step 3 — Deploy to Kubernetes](#step-3--deploy-to-kubernetes)
8. [Step 4 — Verify the Pipeline](#step-4--verify-the-pipeline)
9. [Step 5 — CICERONE CLI Integration](#step-5--ciccerone-cli-integration)
10. [Step 6 — Prometheus Monitoring](#step-6--prometheus-monitoring)
11. [Kafka Topics Reference](#kafka-topics-reference)
12. [Service API Reference](#service-api-reference)
13. [Troubleshooting](#troubleshooting)
14. [Development](#development)

---

## 1. Overview

VIMI processes OPIR infrared satellite data through a streaming pipeline:

```
SBIRS/AWACS Sensors
       ↓
  opir-ingest          ← Ingests IR sighting data, publishes to Kafka
       ↓
missile-warning-engine ← Detects missile tracks, classifies type/range
       ↓
  sensor-fusion         ← Correlates tracks across multiple sensors
       ↓
 alert-dissemination    ← Issues CONOPREP→IMMINENT→INCOMING→HOSTILE alerts
       ↓
   lvc-coordinator       ← Manages live/virtual/constructive entity state
       ↓
   env-monitor           ← Tracks environmental conditions (weather, EM)
       ↓
   replay-engine         ← Records DIS PDU events to .dispcap files
       ↓
   data-catalog          ← OGC CSW catalog for asset discovery
       ↓
   dis-hla-gateway       ← Bridges DIS ↔ HLA federation (IEEE 1278.1 ↔ IEEE 1516)
```

**10 services** run in the `vimi` Kubernetes namespace, consuming shared infrastructure (Kafka, etcd, Redis, PostgreSQL) from the `gms` namespace.

---

## 2. Architecture

### Services & Ports

| Service | Internal Port | Kafka Topic (Input → Output) | Function |
|---------|--------------|------------------------------|---------|
| `opir-ingest` | 8081 | — → `vimi.opir.sensor-data` | SBIRS IR sighting ingestion (simulated) |
| `missile-warning-engine` | 8080 | `vimi.opir.sensor-data` → `vimi.tracks` | Track detection & classification |
| `sensor-fusion` | 8082 | `vimi.tracks` → `vimi.fusion.tracks` | Cross-sensor track correlation |
| `lvc-coordinator` | 8083 | `vimi.fusion.tracks` → `vimi.dis.entity-state` | LVC entity state management |
| `alert-dissemination` | 8084 | `vimi.fusion.tracks` + `vimi.tracks` → `vimi.alerts`, `vimi.c2.alerts` | Alert classification & C2 messaging |
| `env-monitor` | 8085 | — → `vimi.env.events` | Environmental modeling (1° global grid) |
| `replay-engine` | 8086 | `vimi.replay.events` → `.dispcap` files | DIS PDU capture & playback |
| `data-catalog` | 8087 | recordings → REST index | OGC CSW catalog, asset discovery |
| `dis-hla-gateway` | 8090 | `vimi.dis.*` ↔ `vimi.hla.*` | DIS↔HLA bidirectional translation |
| `vimi-plugin` | 8091 | all topics | Cicerone web UI plugin |

### Shared Infrastructure (from `gms` namespace)

Services in `gms` are accessed via Kubernetes `ExternalName` services:

| Service | Address | Purpose |
|---------|---------|---------|
| Kafka | `kafka.gms.svc.cluster.local:9092` | Event streaming |
| etcd | `etcd.gms.svc.cluster.local:2379` | Distributed config/lock |
| Redis | `redis.gms.svc.cluster.local:6379` | Caching, pub/sub |
| PostgreSQL | `postgres.gms.svc.cluster.local:5432` | Persistent storage |

---

## 3. Prerequisites

### Hardware / Cluster

- **Kubernetes cluster** (Kind, K3s, or cloud) with `gms` namespace already deployed
- **kubectl** configured with cluster access
- **Docker** or **Docker Buildx** for building images
- **Kind** (if using local cluster) with sufficient RAM (8GB+ recommended)

### Software Versions

| Component | Version |
|-----------|---------|
| Go | 1.22+ |
| Kubernetes | 1.27+ |
| Docker | 24+ |
| Kafka | (managed, in `gms`) |
| Helm | 3.12+ (optional) |

### Access

- SSH key: `~/.ssh/id_ed25519` (GitLab / IDM access)
- GitLab token: `glpat-...` (for CI/CD and Git operations)
- Kubernetes context: `kind-gms` or your cluster name

---

## 4. Repository Structure

```
git@idm.wezzel.com:crab-meat-repos/trooper-vimi.git
├── apps/                          # 8 mission-processing services
│   ├── opir-ingest/
│   │   ├── main.go
│   │   ├── go.mod
│   │   ├── go.sum
│   │   └── Dockerfile             # go 1.22-alpine → scratch
│   ├── missile-warning-engine/
│   ├── sensor-fusion/
│   ├── lvc-coordinator/
│   ├── alert-dissemination/
│   ├── env-monitor/
│   ├── replay-engine/
│   └── data-catalog/
├── hla-bridge/                    # DIS↔HLA gateway (dis-hla-gateway)
│   ├── main.go
│   ├── go.mod
│   ├── go.sum
│   └── Dockerfile
├── vimi-plugin/                   # Cicerone plugin
│   ├── main.go
│   ├── go.mod
│   ├── go.sum
│   └── Dockerfile
├── FORGE-FOM/
│   └── FOM.xml                    # Federation Object Model (OPIR/Missile/Sensor/Track)
├── k8s/
│   └── vimi-cluster/
│       ├── namespace.yaml         # vimi namespace
│       ├── vimi-cluster.yaml      # ⭐ Full deployment (this is what you apply)
│       ├── vimi-ingress.yaml      # Ingress rules
│       ├── vimi-monitoring.yaml   # Prometheus Operator CRDs
│       └── apps/                  # Individual app manifests (GitOps/ArgoCD)
│           ├── opir-ingest.yaml
│           └── ...
├── cicerone-scripts/
│   ├── cicerone-vimi              # Main CLI tool (bash)
│   ├── vimi-status.sh
│   └── federation-join.sh
├── docs/
│   └── VIMI-INSTALL-GUIDE.md      # This document
└── .gitlab-ci.yml                 # CI/CD pipeline (build + push images)
```

---

## 5. Step 1 — Base Infrastructure

### 5.1 Clone the Repository

```bash
git clone git@idm.wezzel.com:crab-meat-repos/trooper-vimi.git
cd trooper-vimi
```

### 5.2 Verify the `gms` Namespace

VIMI depends on Kafka, etcd, Redis, and PostgreSQL running in the `gms` namespace:

```bash
# Check gms namespace exists and has the required services
kubectl get svc -n gms

# Expected output:
# NAME         TYPE           CLUSTER-IP      EXTERNAL-IP                      PORT(S)
# etcd         ExternalName   <none>          etcd.gms.svc.cluster.local       2379,2380/TCP
# kafka        ExternalName   <none>          kafka.gms.svc.cluster.local      9092,9093/TCP
# postgres     ExternalName   <none>          postgres.gms.svc.cluster.local  5432/TCP
# redis        ExternalName   <none>          redis.gms.svc.cluster.local     6379/TCP
```

If `gms` namespace is missing or services are down, deploy the base GMS stack first before proceeding.

### 5.3 Create the VIMI Namespace

```bash
kubectl create namespace vimi --dry-run=client -o yaml | kubectl apply -f -
```

Or let `vimi-cluster.yaml` create it automatically:

```bash
kubectl apply -f k8s/vimi-cluster/vimi-cluster.yaml --dry-run=server 2>&1 | head -5
```

---

## 6. Step 2 — Build & Load Container Images

VIMI ships 10 container images. You can build them locally for Kind/K3s or push to GHCR for cloud deployments.

### 6.1 Build All Images

```bash
cd trooper-vimi

# Set the registry prefix
REGISTRY="localhost:5000"    # Local Kind registry
# REGISTRY="ghcr.io/YOURORG"  # For cloud / GitHub Container Registry

# Build all apps (apps/*) in parallel via Make or script
for app in apps/*/; do
  name=$(basename "$app")
  echo "Building $name..."
  docker build -t "${REGISTRY}/${name}:latest" -f "${app}Dockerfile" "${app}"
done

# Build hla-bridge (dis-hla-gateway) and vimi-plugin
docker build -t "${REGISTRY}/dis-hla-gateway:latest" -f hla-bridge/Dockerfile hla-bridge/
docker build -t "${REGISTRY}/vimi-plugin:latest"     -f vimi-plugin/Dockerfile vimi-plugin/
```

**Build times:** ~2–5 minutes total on a fast machine. All images use multi-stage build (Go 1.22-alpine builder → `scratch` runtime) to minimize size.

### 6.2 Load Images into Kind (Local)

If using Kind, load images directly onto the node:

```bash
CLUSTER_NAME="gms"  # Your Kind cluster name

for img in opir-ingest missile-warning-engine sensor-fusion lvc-coordinator \
           alert-dissemination env-monitor replay-engine data-catalog \
           dis-hla-gateway vimi-plugin; do
  echo "Loading $img into Kind..."
  kind load docker-image "${REGISTRY}/${img}:latest" --name "$CLUSTER_NAME"
done
```

> **Note:** Use `--name gms` not `--name kind-gms`. The actual Kind cluster name is `gms`.

### 6.3 Push to GHCR (Cloud / Production)

```bash
# Login to GHCR
echo "$GHCR_TOKEN" | docker login ghcr.io -u USERNAME --password-stdin

REGISTRY="ghcr.io/YOURORG/vimi"

for img in opir-ingest missile-warning-engine sensor-fusion lvc-coordinator \
           alert-dissemination env-monitor replay-engine data-catalog \
           dis-hla-gateway vimi-plugin; do
  docker tag "${REGISTRY}/${img}:latest" "${REGISTRY}/${img}:latest"
  docker push "${REGISTRY}/${img}:latest"
done
```

Then update `k8s/vimi-cluster/vimi-cluster.yaml` image references:

```bash
sed -i "s|image: vimi/|image: ${REGISTRY}/vimi/|g" k8s/vimi-cluster/vimi-cluster.yaml
sed -i 's|imagePullPolicy: Never|imagePullPolicy: Always|g' k8s/vimi-cluster/vimi-cluster.yaml
```

### 6.4 Image Tags in YAML

The `vimi-cluster.yaml` uses these image references by default:

| Setting | Local Kind | Production (GHCR) |
|---------|-----------|-------------------|
| Registry prefix | `vimi/` | `ghcr.io/YOURORG/vimi/` |
| Pull policy | `Never` (node-local) | `Always` |
| Tag | `latest` | `latest` or Git SHA |

---

## 7. Step 3 — Deploy to Kubernetes

### 7.1 Single-Command Deploy (Recommended)

```bash
kubectl apply -f k8s/vimi-cluster/vimi-cluster.yaml
```

This deploys everything in one pass:

- `Namespace/vimi`
- 4 ExternalName services (kafka, etcd, redis, postgres → `gms`)
- `ConfigMap/vimi-config`
- 10 Deployments + 10 ClusterIP Services
- `PrometheusRule/vimi-alerts`

### 7.2 Verify Deployment

```bash
# All pods running
kubectl get pods -n vimi

# All deployments ready
kubectl get deployments -n vimi

# Services up
kubectl get svc -n vimi
```

Expected output (all `Running`, `1/1` ready):

```
NAME                                      READY   STATUS
alert-dissemination-xxxxx                1/1     Running
data-catalog-xxxxx                       1/1     Running
dis-hla-gateway-xxxxx                    1/1     Running
env-monitor-xxxxx                        1/1     Running
lvc-coordinator-xxxxx                    1/1     Running
missile-warning-engine-xxxxx             1/1     Running
opir-ingest-xxxxx                        1/1     Running
replay-engine-xxxxx                      1/1     Running
sensor-fusion-xxxxx                      1/1     Running
vimi-plugin-xxxxx                        1/1     Running
```

### 7.3 Rollout Restart

```bash
# Restart a specific service
kubectl rollout restart deployment/opir-ingest -n vimi

# Restart all services
for svc in opir-ingest missile-warning-engine sensor-fusion lvc-coordinator \
           alert-dissemination env-monitor replay-engine data-catalog \
           dis-hla-gateway vimi-plugin; do
  kubectl rollout restart deployment/$svc -n vimi
done

# Watch rollout
kubectl rollout status deployment/opir-ingest -n vimi --timeout=60s
```

### 7.4 Individual App Deployment (GitOps / ArgoCD)

For GitOps workflows, use individual app manifests:

```bash
kubectl apply -f k8s/vimi-cluster/apps/
```

---

## 8. Step 4 — Verify the Pipeline

### 8.1 Check Service Logs

```bash
# Watch opir-ingest (source of pipeline)
kubectl logs -n vimi deploy/opir-ingest --tail=5 -f

# Watch missile-warning-engine (should show track detections)
kubectl logs -n vimi deploy/missile-warning-engine --tail=5 -f

# Watch alert-dissemination (should show alert issuances)
kubectl logs -n vimi deploy/alert-dissemination --tail=5 -f
```

### 8.2 Verify Kafka Topics

```bash
# List VIMI Kafka topics
kubectl exec -n gms kafka-0 -- \
  /opt/kafka/bin/kafka-topics.sh \
  --bootstrap-server localhost:9092 \
  --list 2>/dev/null | grep vimi

# Expected topics:
# vimi.opir.sensor-data
# vimi.alerts
# vimi.tracks
# vimi.fusion.tracks
# vimi.c2.alerts
# vimi.alert-log
# vimi.dis.entity-state
# vimi.dis.entity-state-out
# vimi.hla.object-update
# vimi.hla.interaction
# vimi.env.events
# vimi.replay.events
```

### 8.3 Create Kafka Topics (If Missing)

```bash
KAFKA_CMD="/opt/kafka/bin/kafka-topics.sh --bootstrap-server localhost:9092"

TOPICS=(
  "vimi.opir.sensor-data:3:1"
  "vimi.alerts:3:1"
  "vimi.tracks:3:1"
  "vimi.fusion.tracks:3:1"
  "vimi.c2.alerts:3:1"
  "vimi.alert-log:3:1"
  "vimi.dis.entity-state:3:1"
  "vimi.dis.entity-state-out:3:1"
  "vimi.hla.object-update:3:1"
  "vimi.hla.interaction:3:1"
  "vimi.env.events:3:1"
  "vimi.replay.events:3:1"
)

for topic_spec in "${TOPICS[@]}"; do
  topic="${topic_spec%%:*}"
  parts="${topic_spec#*:}"
  replicas="${parts#*:}"
  
  kubectl exec -n gms kafka-0 -- \
    $KAFKA_CMD --create --topic "$topic" \
    --partitions "${parts%%:*}" \
    --replication-factor "$replicas" \
    --if-not-exists 2>/dev/null
done
```

### 8.4 Health Checks

Each service exposes `/health` on its port:

```bash
for app in opir-ingest missile-warning-engine sensor-fusion lvc-coordinator \
           alert-dissemination env-monitor replay-engine data-catalog \
           dis-hla-gateway vimi-plugin; do
  port=$(kubectl get svc $app -n vimi -o jsonpath='{.spec.ports[0].port}')
  echo -n "$app (:$port): "
  # Use a temporary debug pod if wget/curl not available
  kubectl exec -n vimi deploy/$app -- \
    sh -c "wget -qO- http://localhost:$port/health" 2>/dev/null || \
    echo "(no wget — check logs)"
done
```

---

## 9. Step 5 — CICERONE CLI Integration

The `cicerone-vimi` script provides a unified CLI for VIMI operations.

### 9.1 Install

```bash
# Copy to a location in PATH
sudo cp cicerone-scripts/cicerone-vimi /usr/local/bin/cicerone-vimi
sudo chmod +x /usr/local/bin/cicerone-vimi

# Or add to Cicerone's scripts directory
sudo cp cicerone-scripts/cicerone-vimi /usr/local/bin/cicerone-vimi
```

### 9.2 Configure Environment

```bash
export VIMI_API_HOST="localhost"          # or your cluster API host
export VIMI_KAFKA="kafka.gms.svc.cluster.local:9092"
export KIND_CLUSTER="gms"
export K8S_NAMESPACE="vimi"
export VIMI_PLUGIN_URL="http://localhost:8091"
```

### 9.3 Commands

```bash
# VIMI system status (all services)
cicerone-vimi status

# Real-time track summary
cicerone-vimi tracks

# Alert summary (CONOPREP/IMMINENT/INCOMING/HOSTILE)
cicerone-vimi alerts

# LVC coordinator status
cicerone-vimi lvc

# Environmental events
cicerone-vimi env

# Inject synthetic track event
cicerone-vimi inject --type missile --lat 10.5 --lon -75.0 --speed 2500

# Globe view data (marker colors by alert level)
cicerone-vimi globe

# DIS recording management
cicerone-vimi recording start   # Start DIS PDU capture
cicerone-vimi recording stop    # Stop capture
cicerone-vimi recording list    # List saved recordings
cicerone-vimi recording play    # Playback recording

# HLA federation management
cicerone-vimi federation join   # Join HLA federation
cicerone-vimi federation leave  # Leave federation
cicerone-vimi federation status # Federation state

# Kubernetes cluster info
cicerone-vimi cluster status    # Node/pod status in vimi namespace
```

### 9.4 Kubernetes Dashboard (Alternative)

```bash
kubectl get pods,svc,ingresses -n vimi \
  -o wide --show-labels

kubectl top pods -n vimi            # Resource usage
kubectl describe pods -n vimi       # Events, resource limits
```

---

## 10. Step 6 — Prometheus Monitoring

### 10.1 PrometheusRule

The `PrometheusRule/vimi-alerts` is deployed automatically with `vimi-cluster.yaml`. It defines:

```yaml
alerts:
  - VIMIHostileTrack      # HOSTILE track detected, for 1m
  - VIMIAlertIncoming     # INCOMING alert active, for 10s
  - VIMIAlertHostile      # HOSTILE alert active, for 5s (NCA approval needed)
  - VIMIServiceDown       # Any VIMI service down, for 2m
```

### 10.2 ServiceMonitor (If Prometheus Operator Installed)

```bash
kubectl apply -f k8s/vimi-cluster/vimi-monitoring.yaml
```

This creates:
- `ServiceMonitor/vimi` — scrapes all VIMI service endpoints
- `ConfigMap/vimi-scrape-config` — Prometheus scrape configuration

### 10.3 Port-Metrics Direct Access

```bash
# Query metrics directly from any service
kubectl exec -n vimi deploy/missile-warning-engine -- \
  wget -qO- http://localhost:8080/metrics 2>/dev/null | head -20
```

### 10.4 Prometheus Operator Install (If Not Present)

```bash
# Install Prometheus Operator via Helm (optional)
helm repo add prometheus-community https://prometheus-community.github.io/helm-charts
helm install prometheus prometheus-community/kube-prometheus-stack \
  --namespace vimi \
  --create-namespace \
  --set prometheus.prometheusSpec.serviceMonitorSelector.matchLabels.app=vimi \
  --set grafana.adminPassword=YOUR_PASSWORD
```

---

## Kafka Topics Reference

| Topic | Partitions | Source | Destination |
|-------|-----------|--------|-------------|
| `vimi.opir.sensor-data` | 3 | opir-ingest (simulated SBIRS) | missile-warning-engine |
| `vimi.tracks` | 3 | missile-warning-engine | sensor-fusion, alert-dissemination |
| `vimi.fusion.tracks` | 3 | sensor-fusion | lvc-coordinator, alert-dissemination |
| `vimi.alerts` | 3 | alert-dissemination | lvc-coordinator, vimi-plugin |
| `vimi.c2.alerts` | 3 | alert-dissemination (JTIDS/C2 messages) | External C2 systems |
| `vimi.alert-log` | 3 | alert-dissemination | replay-engine, data-catalog |
| `vimi.dis.entity-state` | 3 | lvc-coordinator | dis-hla-gateway |
| `vimi.dis.entity-state-out` | 3 | dis-hla-gateway | External DIS network |
| `vimi.hla.object-update` | 3 | dis-hla-gateway | HLA RTI (Portico/Mak) |
| `vimi.hla.interaction` | 3 | dis-hla-gateway | HLA RTI |
| `vimi.env.events` | 3 | env-monitor | All services (shared env context) |
| `vimi.replay.events` | 3 | all DIS-enabled services | replay-engine |

---

## Service API Reference

### opir-ingest (`:8081`)

```
GET /health              → 200 OK
GET /metrics             → Prometheus metrics
GET /api/v1/satellites   → List tracked satellites
GET /api/v1/sightings   → Recent IR sightings
```

### missile-warning-engine (`:8080`)

```
GET /health              → 200 OK
GET /metrics             → Prometheus metrics
GET /api/v1/tracks       → Active tracks (MRBM/IRBM/SLBM/CRBM classification)
POST /api/v1/tracks/{id}/classify → Force classification
```

### sensor-fusion (`:8082`)

```
GET /health              → 200 OK
GET /metrics             → Prometheus metrics
GET /api/v1/fused-tracks → Correlated multi-sensor tracks
```

### lvc-coordinator (`:8083`)

```
GET /health              → 200 OK
GET /metrics             → Prometheus metrics
GET /api/v1/lvc-status   → Live/Virtual/Constructive entity counts
GET /api/v1/entities      → Active entity list
```

### alert-dissemination (`:8084`)

```
GET /health               → 200 OK
GET /metrics              → Prometheus metrics
GET /api/v1/alerts        → Active alerts (CONOPREP→IMMINENT→INCOMING→HOSTILE)
GET /api/v1/alerts/history → Alert history
POST /api/v1/inject       → Inject synthetic alert (testing)
```

### env-monitor (`:8085`)

```
GET /health               → 200 OK
GET /metrics              → Prometheus metrics
GET /api/v1/env-status    → Global grid status (cloud/precip/wind/solar/EM)
GET /api/v1/sensor-ratings → SBIRS/AWACS/PATRIOT/THAAD performance ratings
```

### replay-engine (`:8086`)

```
GET /health                → 200 OK
GET /metrics               → Prometheus metrics
GET /api/v1/recordings     → Available .dispcap recordings
POST /api/v1/record/start  → Start recording
POST /api/v1/record/stop   → Stop recording
GET  /api/v1/playback/{id} → Playback control
```

### data-catalog (`:8087`)

```
GET /health                 → 200 OK
GET /metrics                → Prometheus metrics
GET /api/v1/assets          → All indexed assets
GET /api/v1/assets?keyword=missile → Search by keyword
GET /api/v1/assets?bbox=-90,-180,90,180&time=2026-04-01T00:00:00Z → BBox+time filter
GET /api/v1/csw?SERVICE=CSW&REQUEST=GetRecords → OGC CSW GetRecords
```

### dis-hla-gateway (`:8090`)

```
GET /health                 → 200 OK (includes RtiConnected status)
GET /metrics                → Prometheus metrics
GET /api/v1/bridge-status   → DIS↔HLA translation stats
POST /api/v1/inject/dis     → Inject raw DIS PDU
```

### vimi-plugin (`:8091`) — Cicerone Plugin

```
GET /health                 → 200 OK
GET /metrics                → Prometheus metrics
GET /api/v1/vimi/status     → Full VIMI system status
GET /api/v1/vimi/tracks     → Active tracks with alert levels
GET /api/v1/vimi/alerts     → Active alerts
GET /api/v1/vimi/env        → Environmental events
GET /api/v1/vimi/lvc        → LVC coordinator state
POST /api/v1/vimi/inject    → Inject synthetic event
```

---

## Troubleshooting

### Pods Not Starting (ImagePullBackOff)

**Symptom:** `ErrImagePull` or `ImagePullBackOff`

**Cause:** Kind cluster can't reach the image registry.

**Fix:**
```bash
# Verify images are loaded in Kind
kind get nodes --name gms
# Should show: gms-control-plane

# Reload images with correct cluster name
kind load docker-image vimi/opir-ingest:latest --name gms

# If using localhost:5000, deploy a local registry:
docker run -d --name registry --restart=always \
  -p 5000:5000 registry:2

# Then tag and push:
docker tag vimi/opir-ingest:latest localhost:5000/opir-ingest:latest
docker push localhost:5000/opir-ingest:latest
```

### Deployments Not Created (kubectl apply silently skipped)

**Symptom:** Services exist but no Deployments appear.

**Cause:** YAML multi-document formatting issue — `kubectl apply` silently skips documents it can't parse.

**Fix:**
```bash
# Verify YAML parses correctly
python3 -c "
import yaml
with open('k8s/vimi-cluster/vimi-cluster.yaml') as f:
    docs = list(yaml.safe_load_all(f))
print(f'Parsed {len(docs)} YAML docs')
for d in docs:
    if d:
        print(f'  {d.get(\"kind\")} / {d.get(\"metadata\",{}).get(\"name\")}')
"

# Should show 27 docs including 10 Deployments
# If Deployments missing, the YAML is corrupted — pull fresh copy
```

### Kafka Connection Failures

**Symptom:** Services start but log "connection refused" to Kafka.

**Fix:**
```bash
# Verify Kafka is running in gms namespace
kubectl get pods -n gms | grep kafka

# Test Kafka connectivity
kubectl exec -n vimi deploy/opir-ingest -- \
  wget -qO- http://localhost:8081/health

# Check ExternalName service resolution
kubectl run -n vimi --image=busybox:1.36 debug --restart=Never -it -- \
  nslookup kafka.gms.svc.cluster.local

# If Kafka pod is down in gms, restart it:
kubectl rollout restart deployment/kafka -n gms
```

### High Memory / OOM Kills

**Symptom:** Containers killed with exit code 137 (SIGKILL).

**Fix:**
```bash
# Increase memory limits in vimi-cluster.yaml
resources:
  limits:
    memory: "512Mi"  # increase from 256Mi
```

### pod/dis-hla-gateway CrashLoopBackOff

**Cause:** HLA RTI (Portico/Mak) not connected — expected behavior.

**Fix:** The gateway runs fine with `RtiConnected: false`. To connect to a real HLA RTI:
```bash
# Deploy Portico RTI (external)
# Update DIS_HLA_GATEWAY_RTI_HOST env var
kubectl set env deployment/dis-hla-gateway -n vimi \
  RTI_HOST=portico-rt.svc.cluster.local
```

### YAML Apply Fails with Port Name Required

**Symptom:** `spec.ports[0].name: Required value`

**Cause:** ExternalName services need named ports.

**Fix:** Already fixed in `vimi-cluster.yaml` — ensure you're using the latest version:
```bash
git pull origin master
kubectl apply -f k8s/vimi-cluster/vimi-cluster.yaml
```

---

## Development

### Run a Service Locally (Outside Kubernetes)

```bash
# Set environment
export KAFKA_BROKERS="kafka.gms.svc.cluster.local:9092"
export PORT="8080"
export DIS_SITE_ID="1"
export DIS_APP_ID="2"
export REDIS_ADDR="redis.gms.svc.cluster.local:6379"

# Build
cd apps/missile-warning-engine
go build -o /tmp/missile-warning-engine .

# Run
/tmp/missile-warning-engine
```

### Run via kubectl tunnel (for local development)

```bash
# Tunnel Kafka to localhost
kubectl port-forward -n gms svc/kafka 9092:9092 &

# Now local services can connect to localhost:9092
```

### Debug a Running Pod

```bash
# Execute shell in pod (if shell available)
kubectl exec -n vimi deploy/opir-ingest -it -- /bin/sh

# If no shell (scratch image), use wget for debugging
kubectl exec -n vimi deploy/opir-ingest -- \
  wget -qO- http://localhost:8081/api/v1/satellites

# Copy files from pod
kubectl cp vimi/opir-ingest-xxxxx:/var/log/app.log /tmp/app.log
```

### Test the Full Pipeline

```bash
# 1. Start recording
curl -X POST http://localhost:8086/api/v1/record/start

# 2. Watch tracks appear
curl -s http://localhost:8084/api/v1/alerts | jq .

# 3. Watch sensor fusion
curl -s http://localhost:8082/api/v1/fused-tracks | jq .

# 4. Check env monitor
curl -s http://localhost:8085/api/v1/env-status | jq .

# 5. Stop recording
curl -X POST http://localhost:8086/api/v1/record/stop

# 6. List recordings
curl -s http://localhost:8086/api/v1/recordings | jq .
```

### CI/CD Pipeline

The `.gitlab-ci.yml` in the trooper-vimi repo automatically:

1. **Build** all 10 Docker images on every push to `master`
2. **Push** to GHCR (`ghcr.io/vimic/vimi/*`)
3. **Deploy** to Kind (if runner has kubeconfig)

To enable CI/CD:
1. Add a GitLab runner with Docker or Kaniko capability
2. Set `DOCKER_REGISTRY` and `KUBECONFIG` CI variables
3. The pipeline runs `make docker-build docker-push` on merge to `master`

### Add a New Service

```bash
# 1. Create app directory
mkdir -p apps/my-new-service

# 2. Add Go module
cd apps/my-new-service
go mod init github.com/vimic/vimi/my-new-service

# 3. Add kafka consumer/producer
# See apps/opir-ingest/main.go as reference

# 4. Add Dockerfile
cp ../opir-ingest/Dockerfile .

# 5. Add deployment to vimi-cluster.yaml
# See apps/opir-ingest section in the YAML generator

# 6. Test locally
kubectl apply -f k8s/vimi-cluster/vimi-cluster.yaml
```

---

## Appendix: Alert Levels

| Level | Meaning | NCA Required | JTIDS Net |
|-------|---------|-------------|-----------|
| `CONOPREP` | Pre-conflict preparation | No | — |
| `IMMINENT` | Launch detected, impact pending | No | 1 |
| `INCOMING` | Missile in flight, tracking | No | 1 |
| `HOSTILE` | Confirmed hostile intent | **Yes** | 2 |

## Appendix: DIS Exercise IDs

| Exercise | Site ID | App IDs |
|----------|--------|--------|
| 1 | 1 | 1–10 (one per service) |

## Appendix: Key Files

| File | Purpose |
|------|---------|
| `k8s/vimi-cluster/vimi-cluster.yaml` | **Main deployment manifest** |
| `k8s/vimi-cluster/vimi-ingress.yaml` | External access via Ingress |
| `k8s/vimi-cluster/vimi-monitoring.yaml` | Prometheus Operator CRDs |
| `cicerone-scripts/cicerone-vimi` | CLI tool |
| `FORGE-FOM/FOM.xml` | HLA Federation Object Model |

---

**VIMI Version:** 1.0 (Trooper-VIMI)
**Latest Commit:** `5a647d1`
**Repository:** `git@idm.wezzel.com:crab-meat-repos/trooper-vimi.git`
**Deployed Namespace:** `vimi`
**Deployed Cluster:** Kind/GMS (darth)
