# VIMI — Complete User & Implementation Guide

**VIMI** (VIMI Integrated Mission Infrastructure) — DoD-grade LVC (Live/Virtual/Constructive) mission processing system for OPIR satellite data fusion, missile warning, and multi-federation interoperability.

**Repository:** `git@idm.wezzel.com:crab-meat-repos/trooper-vimi.git`
**Namespace:** `vimi`
**Cluster:** Kind (`gms`) on darth (100.92.94.92)
**CI/CD:** GitLab CI on darth (runner `vimi-runner-2`)

---

## Table of Contents

1. [What VIMI Does](#1-what-vimi-does)
2. [Architecture](#2-architecture)
3. [Quick Start](#3-quick-start)
4. [Prerequisites](#4-prerequisites)
5. [Repository Structure](#5-repository-structure)
6. [Building Images](#6-building-images)
7. [Loading Images into Kind](#7-loading-images-into-kind)
8. [Deploying to Kubernetes](#8-deploying-to-kubernetes)
9. [CI/CD Pipeline](#9-cicd-pipeline)
10. [Service Reference](#10-service-reference)
11. [Kafka Topics](#11-kafka-topics)
12. [ CICERONE CLI](#12-cicerone-cli)
13. [Development](#13-development)
14. [Troubleshooting](#14-troubleshooting)

---

## 1. What VIMI Does

VIMI processes infrared (IR) satellite data through a real-time streaming pipeline:

```
SBIRS/AWACS Sensors (simulated)
         ↓
   opir-ingest         → Ingests IR sightings, publishes to Kafka
         ↓
missile-warning-engine → Detects missile tracks, classifies type/speed
         ↓
   sensor-fusion        → Correlates tracks across multiple sensors
         ↓
 alert-dissemination   → Issues CONOPREP→IMMINENT→INCOMING→HOSTILE alerts
         ↓
   lvc-coordinator      → Manages LVC entity state, dead reckoning
         ↓
   env-monitor          → Tracks environmental conditions (grid-based)
         ↓
   replay-engine        → Records DIS PDU events to .dispcap files
         ↓
   data-catalog         → OGC CSW catalog for asset discovery
         ↓
   dis-hla-gateway      → Bridges DIS ↔ HLA ↔ TENA ↔ NETN protocols
```

**10 microservices** run in the `vimi` namespace, consuming shared infrastructure from the `gms` namespace (Kafka, etcd, Redis, PostgreSQL).

---

## 2. Architecture

### 2.1 Service Map

```
┌─────────────────────────────────────────────────────────────┐
│                     gms namespace (shared infra)             │
│  Kafka:9092  etcd:2379  Redis:6379  PostgreSQL:5432        │
└─────────────────────────────────────────────────────────────┘
                              ↓
┌─────────────────────────────────────────────────────────────┐
│                     vimi namespace                           │
│                                                              │
│  opir-ingest:8080 ───────────────→ Kafka ──────────────────→ │
│  missile-warning:8080 ←─────────────┘                         │
│  sensor-fusion:8082 ───────────────→ Kafka ────────────────→│
│  lvc-coordinator:8083 ←────────────┘                         │
│  alert-dissemination:8084 ────────→ Kafka ────────────────→ │
│  env-monitor:8085 ─────────────────→ Kafka ─────────────────→ │
│  replay-engine:8086 ←──────────────┘                         │
│  data-catalog:8087                                          │
│  dis-hla-gateway:8090 ←─────────── DIS ↔ HLA/TENA/NETN      │
│  vimi-plugin:8091 (Cicerone UI plugin)                      │
└─────────────────────────────────────────────────────────────┘
```

### 2.2 Service Ports

| Service | Port | HTTP API Path | Kafka Input | Kafka Output |
|---------|------|-------------|-------------|--------------|
| `opir-ingest` | 8080 | `/api/v1/satellites` | — | `vimi.opir.sensor-data` |
| `missile-warning-engine` | 8080 | `/api/v1/tracks` | `vimi.opir.sensor-data` | `vimi.tracks` |
| `sensor-fusion` | 8082 | `/api/v1/fused-tracks` | `vimi.tracks` | `vimi.fusion.tracks` |
| `lvc-coordinator` | 8083 | `/api/v1/lvc-status` | `vimi.fusion.tracks` | `vimi.dis.entity-state` |
| `alert-dissemination` | 8084 | `/api/v1/alerts` | `vimi.fusion.tracks` + `vimi.tracks` | `vimi.alerts`, `vimi.c2.alerts` |
| `env-monitor` | 8085 | `/api/v1/env-status` | — | `vimi.env.events` |
| `replay-engine` | 8086 | `/api/v1/recordings` | `vimi.replay.events` | `.dispcap` files |
| `data-catalog` | 8087 | `/api/v1/assets` | recordings | REST index |
| `dis-hla-gateway` | 8090 | `/api/v1/bridge-status` | `vimi.dis.*` ↔ | `vimi.hla.*` |
| `vimi-plugin` | 8091 | `/api/v1/vimi/*` | all topics | — |

### 2.3 Alert Levels

| Level | NCA Required | JTIDS Net | Description |
|-------|-------------|-----------|-------------|
| `CONOPREP` | No | — | Pre-conflict preparation |
| `IMMINENT` | No | 1 | Launch detected, impact pending |
| `INCOMING` | No | 1 | Missile in flight, tracking |
| `HOSTILE` | **Yes** | 2 | Confirmed hostile intent |

---

## 3. Quick Start

### One-Command Deploy (after images are loaded)

```bash
kubectl apply -f k8s/vimi-cluster/
kubectl get pods -n vimi -w
```

### Full Rebuild + Deploy Cycle

```bash
# 1. Pull latest
git clone git@idm.wezzel.com:crab-meat-repos/trooper-vimi.git
cd trooper-vimi

# 2. Build all images
REGISTRY=darth:5000
for app in apps/*/ hla-bridge vimi-plugin; do
  name=$(basename "$app")
  docker build -t "${REGISTRY}/vimi-${name}:latest" -f "${app}Dockerfile" "${app}"
done

# 3. Load into Kind
for app in opir-ingest missile-warning-engine sensor-fusion lvc-coordinator \
           alert-dissemination env-monitor replay-engine data-catalog \
           dis-hla-gateway vimi-plugin; do
  docker save "${REGISTRY}/vimi-${app}:latest" | \
    docker exec -i gms-control-plane ctr -n k8s.io images import -
done

# 4. Deploy
kubectl apply -f k8s/vimi-cluster/ -n vimi

# 5. Verify
kubectl get pods -n vimi
```

---

## 4. Prerequisites

### Hardware/Software

- **Kind cluster** named `gms` running on darth (100.92.94.92)
- **kubectl** configured for `kind-gms` context
- **Docker** 24+ for building images
- **gms namespace** deployed with Kafka, etcd, Redis, PostgreSQL

### Verify gms Namespace

```bash
kubectl get svc -n gms

# Expected:
# NAME      TYPE           CLUSTER-IP   EXTERNAL-IP                      PORT(S)
# etcd      ExternalName   <none>       etcd.gms.svc.cluster.local       2379,2380/TCP
# kafka     ExternalName   <none>       kafka.gms.svc.cluster.local      9092,9093/TCP
# postgres  ExternalName   <none>       postgres.gms.svc.cluster.local   5432/TCP
# redis     ExternalName   <none>       redis.gms.svc.cluster.local      6379/TCP
```

### GitLab Runner

Runner `vimi-runner-2` (ID=17) on darth handles CI/CD:
- Executor: Docker with DinD (`docker:24-dind`)
- Tags: `vimi`, `k8s`, `docker`
- Project-agnostic: works for both VIMI and FORGE

---

## 5. Repository Structure

```
trooper-vimi/
├── apps/                          # 8 mission-processing microservices
│   ├── opir-ingest/               # SBIRS IR sighting ingestion
│   │   ├── main.go
│   │   ├── go.mod
│   │   ├── go.sum
│   │   └── Dockerfile             # go 1.22-alpine → scratch
│   ├── missile-warning-engine/     # Track detection + classification
│   ├── sensor-fusion/             # Cross-sensor track correlation
│   ├── lvc-coordinator/           # DIS entity state + dead reckoning
│   ├── alert-dissemination/        # Alert classification + C2 messaging
│   ├── env-monitor/               # Environmental modeling (grid)
│   ├── replay-engine/              # DIS PDU capture + playback
│   └── data-catalog/              # OGC CSW catalog
├── hla-bridge/                    # DIS↔HLA↔TENA gateway
│   ├── main.go
│   ├── go.mod
│   ├── go.sum
│   └── Dockerfile
├── vimi-plugin/                   # Cicerone web UI plugin
│   ├── main.go
│   ├── go.mod
│   ├── go.sum
│   └── Dockerfile
├── VIMI-FOM/                      # HLA Federation Object Model
│   └── FOM.xml                    # IEEE 1516-2010 FOM definitions
├── k8s/
│   └── vimi-cluster/
│       ├── vimi-cluster.yaml      # ⭐ Full deployment (Namespace + Svcs + Deployments)
│       ├── vimi-ingress.yaml      # External access via Ingress
│       ├── vimi-monitoring.yaml   # Prometheus Operator CRDs
│       └── apps/                  # Individual app manifests
│           ├── opir-ingest.yaml
│           └── ... (10 files)
├── cicerone-scripts/
│   └── cicerone-vimi              # CLI tool
├── docs/
│   ├── VIMI-USER-GUIDE.md        # This document
│   └── VIMI-INSTALL-GUIDE.md      # Installation reference
├── .gitlab-ci.yml                 # CI/CD pipeline
└── README.md
```

---

## 6. Building Images

### Local Build (for Kind)

```bash
cd trooper-vimi
REGISTRY="darth:5000"

# Build apps
for app in apps/*/; do
  name=$(basename "$app")
  echo "Building vimi-${name}..."
  docker build -t "${REGISTRY}/vimi-${name}:latest" \
    -f "${app}Dockerfile" "${app}"
done

# Build hla-bridge and vimi-plugin
docker build -t "${REGISTRY}/vimi-dis-hla-gateway:latest" \
  -f hla-bridge/Dockerfile hla-bridge/
docker build -t "${REGISTRY}/vimi-vimi-plugin:latest" \
  -f vimi-plugin/Dockerfile vimi-plugin/
```

### Image Tags Used

| Image | Tag | Purpose |
|-------|-----|---------|
| `darth:5000/vimi-opir-ingest` | `latest` | OPIR ingest service |
| `darth:5000/vimi-missile-warning-engine` | `latest` | Missile warning |
| `darth:5000/vimi-sensor-fusion` | `latest` | Sensor fusion |
| `darth:5000/vimi-lvc-coordinator` | `latest` | LVC coordinator |
| `darth:5000/vimi-alert-dissemination` | `latest` | Alert dissemination |
| `darth:5000/vimi-env-monitor` | `latest` | Environment monitor |
| `darth:5000/vimi-replay-engine` | `latest` | Replay engine |
| `darth:5000/vimi-data-catalog` | `latest` | Data catalog |
| `darth:5000/vimi-dis-hla-gateway` | `latest` | DIS/HLA gateway |
| `darth:5000/vimi-vimi-plugin` | `latest` | Cicerone plugin |

### Multi-Platform Build (for GHCR)

```bash
REGISTRY="ghcr.io/YOURORG/vimi"
docker buildx create --use
docker buildx build --platforms linux/amd64,linux/arm64 \
  -t "${REGISTRY}/vimi-opir-ingest:latest" \
  -f apps/opir-ingest/Dockerfile apps/opir-ingest/ \
  --push
```

---

## 7. Loading Images into Kind

### Step 1: Start a Local Registry (Required)

```bash
# On darth, create a registry accessible from inside Kind
docker run -d --name vimi-registry --restart=always \
  -p 5000:5000 \
  -v /var/lib/registry:/var/lib/registry \
  registry:2

# Verify it's running
curl http://darth:5000/v2/
# Should return: {}
```

### Step 2: Load Images into Kind Node

Kind nodes run containerd, not docker. Use `ctr` to import images:

```bash
# Get the kind node name
kubectl get nodes -o jsonpath='{.items[0].metadata.name}'
# Output: gms-control-plane

# For each app, save the image and import into containerd
for app in opir-ingest missile-warning-engine sensor-fusion lvc-coordinator \
           alert-dissemination env-monitor replay-engine data-catalog \
           dis-hla-gateway vimi-plugin; do
  echo "Loading vimi-${app}..."
  docker save "darth:5000/vimi-${app}:latest" | \
    docker exec -i gms-control-plane \
    ctr -n k8s.io images import -
done

# Verify images are loaded
docker exec gms-control-plane ctr -n k8s.io images ls | grep vimi
```

### Step 3: Verify Images in CRI

```bash
# kubelet uses containerd's CRI interface (crictl)
docker exec gms-control-plane crictl images | grep vimi
```

### Important: Why `ctr import` Instead of `kind load`?

- `kind load` uses Docker inside the Kind node, but Kind nodes run containerd
- `ctr -n k8s.io images import` loads directly into the Kubernetes container runtime
- `kind load` fails with containerd 1.7+ due to multi-platform manifest handling issues

---

## 8. Deploying to Kubernetes

### 8.1 Full Deployment

```bash
kubectl apply -f k8s/vimi-cluster/vimi-cluster.yaml

# Or deploy individual apps:
kubectl apply -f k8s/vimi-cluster/apps/
```

### 8.2 Verify Deployment

```bash
# Watch pods
kubectl get pods -n vimi -w

# All should be Running 1/1
kubectl get pods -n vimi --field-selector=status.phase=Running

# Services
kubectl get svc -n vimi
```

### 8.3 Restart Services

```bash
# Restart one service
kubectl rollout restart deployment/opir-ingest -n vimi

# Restart all services
for svc in opir-ingest missile-warning-engine sensor-fusion lvc-coordinator \
           alert-dissemination env-monitor replay-engine data-catalog \
           dis-hla-gateway vimi-plugin; do
  kubectl rollout restart deployment/$svc -n vimi
done

# Watch status
kubectl rollout status deployment/opir-ingest -n vimi --timeout=60s
```

### 8.4 Access Logs

```bash
# Single service
kubectl logs -n vimi deploy/opir-ingest --tail=20 -f

# All services
for svc in opir-ingest missile-warning-engine sensor-fusion lvc-coordinator \
           alert-dissemination env-monitor replay-engine data-catalog \
           dis-hla-gateway vimi-plugin; do
  echo "=== $svc ==="
  kubectl logs -n vimi deploy/$svc --tail=3
done
```

---

## 9. CI/CD Pipeline

### 9.1 Pipeline Overview

The `.gitlab-ci.yml` defines 14 stages:

```
build (10 jobs) → test → security-scan → publish → deploy-k8s
```

| Job | Stage | Description |
|-----|-------|-------------|
| `build-opir-ingest` | build | Docker build + push to darth:5000 |
| `build-missile-warning-engine` | build | Docker build + push |
| `build-sensor-fusion` | build | Docker build + push |
| `build-lvc-coordinator` | build | Docker build + push |
| `build-alert-dissemination` | build | Docker build + push |
| `build-env-monitor` | build | Docker build + push |
| `build-replay-engine` | build | Docker build + push |
| `build-data-catalog` | build | Docker build + push |
| `build-hla-bridge` | build | Docker build + push (as dis-hla-gateway) |
| `build-vimi-plugin` | build | Docker build + push |
| `test-all` | test | `go test ./...` for all apps |
| `security-scan` | security-scan | Trivy vulnerability scan |
| `publish` | publish | Tag and push final images |
| `deploy-k8s` | deploy | Load images into Kind, apply YAML, rollout restart |

### 9.2 CI/CD Variables

Set these in **Settings → CI/CD → Variables**:

| Variable | Value | Description |
|----------|-------|-------------|
| `REGISTRY` | `darth:5000` | Local Docker registry |
| `KUBECONFIG_BASE64` | `(base64 encoded kubeconfig)` | Kind cluster kubeconfig |

### 9.3 deploy-k8s Script

```bash
# 1. Decode kubeconfig
echo "$KUBECONFIG_BASE64" | base64 -d > /tmp/kubeconfig

# 2. Fix API server address (kind binds to 0.0.0.0 internally)
sed -i 's|https://[^:]*:6443|https://172.17.0.1:6443|g' /tmp/kubeconfig

# 3. Load images from registry into Kind
for app in opir-ingest missile-warning-engine sensor-fusion ...; do
  docker save "$REGISTRY/vimi-$app:latest" | \
    docker exec -i "$(kubectl --kubeconfig /tmp/kubeconfig get nodes -o jsonpath='{.items[0].metadata.name}')" \
    ctr -n k8s.io images import -
done

# 4. Apply Kubernetes manifests
kubectl --kubeconfig /tmp/kubeconfig apply -f k8s/vimi-cluster/apps/ --namespace vimi

# 5. Update image tags
for app in ...; do
  kubectl --kubeconfig /tmp/kubeconfig set image deployment/$app $app=$REGISTRY/vimi-$app:latest -n vimi
done

# 6. Rollout restart
for app in ...; do
  kubectl --kubeconfig /tmp/kubeconfig rollout restart deployment/$app -n vimi
done
```

### 9.4 Trigger a Pipeline

```bash
# Push to master (automatic)
git add .
git commit -m "fix: ..."
git push origin master

# Or trigger manually via API (if you have access)
curl -X POST "https://idm.wezzel.com/api/v4/projects/48/pipeline" \
  -H "PRIVATE-TOKEN: $GITLAB_TOKEN"
```

### 9.5 View Pipeline Status

```bash
# Via GitLab API
curl -s "https://idm.wezzel.com/api/v4/projects/48/pipelines" \
  -H "PRIVATE-TOKEN: $GITLAB_TOKEN" | python3 -m json.tool

# Via kubectl
kubectl get pods -n vimi
kubectl get deployments -n vimi
```

---

## 10. Service Reference

### Common Endpoints (All Services)

| Endpoint | Description |
|----------|-------------|
| `GET /health` | Health check (200 OK) |
| `GET /metrics` | Prometheus metrics |

### 10.1 opir-ingest (:8080)

```bash
kubectl exec -n vimi deploy/opir-ingest -- \
  wget -qO- http://localhost:8080/api/v1/satellites
```

Ingests simulated IR satellite sightings from SBIRS/AWACS sensors. Publishes to `vimi.opir.sensor-data` Kafka topic.

### 10.2 missile-warning-engine (:8080)

```bash
kubectl exec -n vimi deploy/missile-warning-engine -- \
  wget -qO- http://localhost:8080/api/v1/tracks
```

Detects missile tracks from IR sightings. Classifies as MRBM/IRBM/SLBM/CRBM. Publishes to `vimi.tracks`.

### 10.3 sensor-fusion (:8082)

```bash
kubectl exec -n vimi deploy/sensor-fusion -- \
  wget -qO- http://localhost:8082/api/v1/fused-tracks
```

Correlates tracks across multiple sensors. Publishes to `vimi.fusion.tracks`.

### 10.4 lvc-coordinator (:8083)

```bash
kubectl exec -n vimi deploy/lvc-coordinator -- \
  wget -qO- http://localhost:8083/api/v1/lvc-status
```

Manages Live/Virtual/Constructive entity state using DIS dead reckoning. Publishes to `vimi.dis.entity-state`.

### 10.5 alert-dissemination (:8084)

```bash
kubectl exec -n vimi deploy/alert-dissemination -- \
  wget -qO- http://localhost:8084/api/v1/alerts
```

Issues alerts: CONOPREP → IMMINENT → INCOMING → HOSTILE. Publishes to `vimi.alerts` and `vimi.c2.alerts`.

### 10.6 env-monitor (:8085)

```bash
kubectl exec -n vimi deploy/env-monitor -- \
  wget -qO- http://localhost:8085/api/v1/env-status
```

Environmental modeling using a 1° global grid. Tracks cloud, precipitation, wind, solar, EM conditions. Publishes to `vimi.env.events`.

### 10.7 replay-engine (:8086)

```bash
kubectl exec -n vimi deploy/replay-engine -- \
  wget -qO- http://localhost:8086/api/v1/recordings
```

Records DIS PDU events to `.dispcap` files. Supports playback.

### 10.8 data-catalog (:8087)

```bash
kubectl exec -n vimi deploy/data-catalog -- \
  wget -qO- http://localhost:8087/api/v1/assets
```

OGC CSW catalog for asset discovery. Indexes recordings and provides spatial/temporal queries.

### 10.9 dis-hla-gateway (:8090)

```bash
kubectl exec -n vimi deploy/dis-hla-gateway -- \
  wget -qO- http://localhost:8090/api/v1/bridge-status
```

Bridges DIS ↔ HLA ↔ TENA ↔ NETN. Translates entity state PDUs bidirectionally.

### 10.10 vimi-plugin (:8091) — Cicerone UI Plugin

```bash
kubectl exec -n vimi deploy/vimi-plugin -- \
  wget -qO- http://localhost:8091/api/v1/vimi/status
```

Web UI plugin for Cicerone. Aggregates tracks, alerts, LVC status for the CICERONE web interface.

---

## 11. Kafka Topics

### Create Topics (If Missing)

```bash
KAFKA_POD=$(kubectl get pods -n gms -l app.kubernetes.io/name=kafka -o jsonpath='{.items[0].metadata.name}')

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

for spec in "${TOPICS[@]}"; do
  topic="${spec%%:*}"
  parts="${spec##*:}"
  kubectl exec -n gms "$KAFKA_POD" -- \
    /opt/kafka/bin/kafka-topics.sh \
    --bootstrap-server localhost:9092 \
    --create --topic "$topic" \
    --partitions "${spec%%:*}" \
    --replication-factor "${spec##*:}" \
    --if-not-exists 2>/dev/null
  echo "Ensured topic: $topic"
done
```

### List Topics

```bash
kubectl exec -n gms kafka-0 -- \
  /opt/kafka/bin/kafka-topics.sh \
  --bootstrap-server localhost:9092 --list 2>/dev/null | grep vimi
```

---

## 12. CICERONE CLI

The `cicerone-vimi` script provides a unified CLI for VIMI operations.

### Install

```bash
sudo cp cicerone-scripts/cicerone-vimi /usr/local/bin/
sudo chmod +x /usr/local/bin/cicerone-vimi
```

### Commands

```bash
cicerone-vimi status        # VIMI system status (all services)
cicerone-vimi tracks        # Real-time track summary
cicerone-vimi alerts        # Alert summary (CONOPREP/IMMINENT/INCOMING/HOSTILE)
cicerone-vimi lvc           # LVC coordinator status
cicerone-vimi env           # Environmental events
cicerone-vimi inject        # Inject synthetic track event
cicerone-vimi globe         # Globe view data
cicerone-vimi recording      # DIS recording management
cicerone-vimi federation    # HLA federation management
cicerone-vimi cluster       # Kubernetes cluster info
```

---

## 13. Development

### Run a Service Locally

```bash
# Set environment
export KAFKA_BROKERS="kafka.gms.svc.cluster.local:9092"
export PORT="8080"
export DIS_SITE_ID="1"
export DIS_APP_ID="2"
export REDIS_ADDR="redis.gms.svc.cluster.local:6379"

# Build
cd apps/missile-warning-engine
go build -o /tmp/mwe .

# Run
/tmp/mwe
```

### Debug a Running Pod

```bash
# Shell into pod (if available)
kubectl exec -n vimi deploy/opir-ingest -it -- /bin/sh

# Or use wget (scratch images have no shell)
kubectl exec -n vimi deploy/opir-ingest -- \
  wget -qO- http://localhost:8080/api/v1/satellites

# Copy logs
kubectl cp vimi/$(kubectl get pods -n vimi -l app=opir-ingest -o jsonpath='{.items[0].metadata.name}'):/var/log/app.log /tmp/app.log
```

### Add a New Service

1. Create `apps/my-service/` with `main.go`, `go.mod`, `Dockerfile`
2. Add build job to `.gitlab-ci.yml`
3. Add deployment to `k8s/vimi-cluster/apps/my-service.yaml`
4. Update `vimi-cluster.yaml` to include the new app
5. Push — CI builds, tests, and deploys automatically

---

## 14. Troubleshooting

### Pods Not Starting (ImagePullBackOff)

**Symptom:** `ErrImagePull` or `ImagePullBackOff` — kubelet can't pull images.

**Diagnosis:**
```bash
# Check kubelet logs on the Kind node
docker logs gms-control-plane | grep -i "pull\|image\|error" | tail -20

# Common error: "dial tcp <IP>:5000: i/o timeout"
# This means the Kind node can't reach the registry
```

**Fix:**
```bash
# Verify images are in containerd
docker exec gms-control-plane ctr -n k8s.io images ls | grep vimi

# Verify registry is reachable from inside Kind
docker exec gms-control-plane sh -c 'curl -s http://darth:5000/v2/'

# If registry is unreachable, either:
# 1. Fix network connectivity (add route)
# 2. Run a registry inside the Kind network
# 3. Use imagePullPolicy: Never with pre-loaded images
```

### Deployments Not Created

**Symptom:** `kubectl apply` runs but no Deployments appear.

**Diagnosis:**
```bash
# Check YAML parses correctly
python3 -c "
import yaml
with open('k8s/vimi-cluster/vimi-cluster.yaml') as f:
    docs = list(yaml.safe_load_all(f))
for d in docs:
    if d and d.get('kind') == 'Deployment':
        print(f\"Deployment: {d['metadata']['name']}\")
"
```

### Services CrashLoopBackOff (Port Conflicts)

**Symptom:** Container exits immediately, `CrashLoopBackOff`.

**Diagnosis:**
```bash
# Check logs for port already in use
kubectl logs -n vimi deploy/opir-ingest --previous | grep -i "listen\|bind\|address"

# Common cause: two pods trying to use same port
# Fix: Each app uses unique port (8080-8087)
```

### Kafka Connection Failures

**Symptom:** "connection refused" to kafka:9092.

**Fix:**
```bash
# Verify Kafka is running
kubectl get pods -n gms | grep kafka

# Test DNS resolution
kubectl run -n vimi --image=busybox:1.36 debug --restart=Never -it -- \
  nslookup kafka.gms.svc.cluster.local

# Restart Kafka if needed
kubectl rollout restart deployment/kafka -n gms
```

### High Memory / OOM Kills

**Symptom:** Exit code 137 (SIGKILL).

**Fix:**
```yaml
# Increase memory limits in the Deployment spec
resources:
  limits:
    memory: "512Mi"  # default is 256Mi
```

### Pipeline Fails at deploy-k8s

**Symptom:** CI pipeline succeeds through `publish` but fails at `deploy-k8s`.

**Diagnosis:**
```bash
# Check the job log in GitLab UI
# Common causes:
# 1. KUBECONFIG_BASE64 not set → "error: current-context is not set"
# 2. Kind API server unreachable → "connection refused"
# 3. Images not in registry → ImagePullBackOff on pods
```

### YAML Port Value Errors

**Symptom:** `json: cannot unmarshal string into Go struct field ... of type int32`

**Cause:** `containerPort: 9092` is written as string instead of integer.

**Fix:** Ensure YAML has:
```yaml
ports:
  - containerPort: 9092    # integer, not quoted
```

---

## Quick Reference Card

```bash
# Clone
git clone git@idm.wezzel.com:crab-meat-repos/trooper-vimi.git
cd trooper-vimi

# Build all images
REGISTRY=darth:5000
for app in apps/*/ hla-bridge vimi-plugin; do
  name=$(basename "$app")
  docker build -t "${REGISTRY}/vimi-${name}:latest" -f "${app}Dockerfile" "${app}"
done

# Load into Kind
for app in opir-ingest missile-warning-engine sensor-fusion lvc-coordinator \
           alert-dissemination env-monitor replay-engine data-catalog \
           dis-hla-gateway vimi-plugin; do
  docker save "${REGISTRY}/vimi-${app}:latest" | \
    docker exec -i gms-control-plane ctr -n k8s.io images import -
done

# Deploy
kubectl apply -f k8s/vimi-cluster/vimi-cluster.yaml

# Verify
kubectl get pods -n vimi
kubectl logs -n vimi deploy/opir-ingest --tail=5 -f

# Restart all
for svc in opir-ingest missile-warning-engine sensor-fusion lvc-coordinator \
           alert-dissemination env-monitor replay-engine data-catalog \
           dis-hla-gateway vimi-plugin; do
  kubectl rollout restart deployment/$svc -n vimi
done
```

---

**VIMI Version:** 1.0
**Repository:** `git@idm.wezzel.com:crab-meat-repos/trooper-vimi.git`
**Latest Commit:** `6674b28` (port fixes)
**CI/CD:** GitLab CI on darth, runner `vimi-runner-2`
**Deployed:** `vimi` namespace on Kind `gms` (darth)
