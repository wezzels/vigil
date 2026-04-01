# TROOPER-FORGE

**FORGE MDPAF Conversion Project** — DoD LVC Simulation Federation

DoD-aligned simulation and mission processing framework supporting Live, Virtual, and Constructive (LVC) training, OPIR satellite data fusion, missile warning workflows, and multi-federation interoperability via DIS/HLA/TENA/NETN protocols.

## Project Structure

```
trooper-forge/
├── FORGE-FOM/              # HLA Federation Object Model (IEEE 1516-2010)
│   └── FOM.xml             # Object/interaction class definitions
├── Dockerfiles/            # Base container images
├── k8s/                    # Kubernetes manifests
│   └── forge-cluster/      # Kind/K8s namespace + services
├── vm/                     # VM templates + cloud-init
│   └── cloud-init/         # cloud-init configs for FORGE VMs
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

- Kubernetes/Kind cluster (namespace: `forge`)
- Kafka + etcd + Redis (available in `gms` namespace)
- Docker for building app images
- VIMIC for VM lifecycle management

### Deploy Base Services

```bash
kubectl apply -f k8s/forge-cluster/namespace.yaml
kubectl apply -f k8s/forge-cluster/base-services.yaml
```

### Build an App

```bash
cd apps/opir-ingest
docker build -t registry.stsgym.com/forge-opir-ingest:latest .
docker push registry.stsgym.com/forge-opir-ingest:latest
```

### Deploy to K8s

```bash
kubectl apply -f k8s/forge-cluster/ -n forge
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

## Phases

- **Phase 0**: Foundation — Monorepo, FOM, CI/CD, K8s cluster
- **Phase 1**: Core Infrastructure — OPIR ingest, missile warning, sensor fusion
- **Phase 2**: Mission Processing — Alerts, env monitor, LVC coordinator, replay
- **Phase 3**: Advanced Integration — DIS/HLA gateway, VIMIC plugin, cross-domain
- **Phase 4**: Operational Federation — Coalition, DoD certification

## Repository

**GitLab:** `git@idm.wezzel.com:crab-meat-repos/trooper-forge.git`

*Note: Repo must be created manually (bot token has Guest access — cannot create via API). Create via web UI at https://idm.wezzel.com/crab-meat-repos/trooper-forge*
