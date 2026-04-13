# VIGIL

**LVC Simulation Federation** — DoD Missile Warning & Sensor Fusion Platform

VIGIL is a DoD-aligned simulation and mission processing framework supporting Live, Virtual, and Constructive (LVC) training, OPIR satellite data fusion, missile warning workflows, and multi-federation interoperability via DIS/HLA/TENA/NETN protocols.

## Project Structure

```
vigil/
├── VIMI-FOM/              # HLA Federation Object Model (IEEE 1516-2010)
│   └── FOM.xml             # Object/interaction class definitions
├── Dockerfiles/            # Base container images
├── k8s/                    # Kubernetes manifests
│   └── vimi-cluster/      # Kind/K8s namespace + services
├── vm/                     # VM templates + cloud-init
│   └── cloud-init/         # cloud-init configs for VIGIL VMs
├── apps/                   # Mission processing microservices
│   ├── opir-ingest/        # OPIR satellite data ingestion
│   ├── missile-warning-engine/  # Threat detection + trajectory
│   ├── sensor-fusion/      # Multi-source track fusion
│   ├── alert-dissemination/ # NCA/Pentagon alert distribution
│   ├── env-monitor/        # Fire, volcano, agricultural monitoring
│   ├── replay-engine/      # Event recording + playback
│   ├── data-catalog/       # JFCDS data discovery service
│   └── lvc-coordinator/    # DIS entity management + dead reckoning
├── dis-pdu/                # DIS protocol implementation (IEEE 1278.1)
├── hla-bridge/             # DIS↔HLA↔TENA gateway
├── cicerone-scripts/       # CICERONE CLI extensions
├── vimic-plugin/           # VIMIC integration module
└── docs/                   # Documentation
```

## Quick Start

### Prerequisites

- Docker + Docker Compose
- Kafka + Redis (or use docker-compose.local.yaml)

### Deploy Base Services (Local)

```bash
docker compose -f docker-compose.local.yaml up -d
```

### Build Apps

```bash
cd apps/opir-ingest
docker build -t vigil-opir-ingest:latest .
```

### Run Services

```bash
docker run -d --name vigil-opir-ingest \
  --network host \
  -e KAFKA_BROKERS=localhost:9092 \
  -e PORT=8081 \
  vigil-opir-ingest:latest
```

### Check Health

```bash
curl http://localhost:8081/health | jq
```

## Key Standards

| Standard | Purpose |
|----------|---------|
| DIS (IEEE 1278.1) | Entity state, fire, detonation PDUs |
| HLA (IEEE 1516-2010) | Federation object model, RTI |
| TENA 2015 | Range middleware, object reuse |
| NETN FOM (STANREC 4800) | NATO coalition interoperability |
| JFCDS (CSIAC/DTIC) | Joint Federated Common Data Services |
| DoD Cloud IaC | DevSecOps pipeline + infrastructure |

## Services

| Service | Port | Purpose |
|---------|------|---------|
| opir-ingest | 8081 | OPIR satellite IR data ingestion |
| missile-warning | 8082 | Threat detection & trajectory prediction |
| sensor-fusion | 8083 | Multi-source track correlation |
| lvc-coordinator | 8084 | DIS entity management |

## Phases

- **Phase 0**: Foundation — Monorepo, FOM, CI/CD, K8s cluster ✅
- **Phase 1**: Core Infrastructure — OPIR ingest, missile warning, sensor fusion ✅
- **Phase 2**: Mission Processing — Alerts, env monitor, LVC coordinator, replay
- **Phase 3**: Advanced Integration — DIS/HLA gateway, VIMIC plugin, cross-domain
- **Phase 4**: Operational Federation — Coalition, DoD certification

## Repository

**GitLab:** `git@idm.wezzel.com:crab-meat-repos/vigil.git`
**Local:** `/home/wez/vigil`