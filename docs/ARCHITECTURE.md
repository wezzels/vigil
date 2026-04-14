# VIGIL Architecture

## System Overview

VIGIL is a distributed missile warning and sensor fusion platform designed for real-time tracking, correlation, and alerting.

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                            VIGIL SYSTEM ARCHITECTURE                          │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                               │
│  ┌────────────────────────────────────────────────────────────────────────┐ │
│  │                         SENSOR INPUT LAYER                               │ │
│  │  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐   │ │
│  │  │   OPIR   │  │  RADAR   │  │  SBIRS   │  │   DSP   │  │  Other   │   │ │
│  │  │  Sensor  │  │  Feeds   │  │  Data    │  │  Sats   │  │ Sources  │   │ │
│  │  └────┬─────┘  └────┬─────┘  └────┬─────┘  └────┬─────┘  └────┬─────┘   │ │
│  └───────┼─────────────┼─────────────┼─────────────┼─────────────┼─────────┘ │
│          └─────────────┴─────────────┼─────────────┴─────────────┘           │
│                                      │                                       │
│  ┌───────────────────────────────────┼───────────────────────────────────┐  │
│  │                         INGEST LAYER │                                │  │
│  │  ┌──────────────────────────────────┴──────────────────────────────┐ │  │
│  │  │                     Kafka Event Streaming                         │ │  │
│  │  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐              │ │  │
│  │  │  │  opir-detect│  │ radar-tracks│  │  sbirs-data │              │ │  │
│  │  │  └─────────────┘  └─────────────┘  └─────────────┘              │ │  │
│  │  └──────────────────────────────────────────────────────────────────┘ │  │
│  └───────────────────────────────────┬───────────────────────────────────┘  │
│                                      │                                       │
│  ┌───────────────────────────────────┼───────────────────────────────────┐  │
│  │                    PROCESSING LAYER  │                                │  │
│  │  ┌──────────────────┐  ┌───────────┴──────────┐  ┌──────────────────┐ │  │
│  │  │   OPIR Ingest    │  │   Sensor Fusion      │  │ Missile Warning  │ │  │
│  │  │  ┌────────────┐  │  │  ┌────────────────┐  │  │  ┌────────────┐  │ │  │
│  │  │  │  Detection │  │  │  │ Track Correlate │  │  │  │ Threat Est │  │ │  │
│  │  │  │  Parsing   │  │  │  │ MHT Fusion     │  │  │  │ Alert Gen  │  │ │  │
│  │  │  │  Geolocate │  │  │  │ State Estim   │  │  │  │ CONOPREP   │  │ │  │
│  │  │  └────────────┘  │  │  └────────────────┘  │  │  └────────────┘  │ │  │
│  │  └──────────────────┘  └─────────────────────┘  └──────────────────┘ │  │
│  └───────────────────────────────────┬───────────────────────────────────┘  │
│                                      │                                       │
│  ┌───────────────────────────────────┼───────────────────────────────────┐  │
│  │                       LAYER LAYER   │                                │  │
│  │  ┌──────────────────┐  ┌───────────┴──────────┐  ┌──────────────────┐ │  │
│  │  │  HLA Federate    │  │  DIS Gateway         │  │  LVC Coordinator │ │  │
│  │  │  ┌────────────┐  │  │  ┌────────────────┐  │  │  ┌────────────┐  │ │  │
│  │  │  │ RPR FOM    │  │  │  │ PDU Encode     │  │  │  │ Time Sync  │  │ │  │
│  │  │  │ Entity State│ │  │  │ PDU Decode     │  │  │  │ Entity Mgr │  │ │  │
│  │  │  └────────────┘  │  │  └────────────────┘  │  │  └────────────┘  │ │  │
│  │  └──────────────────┘  └─────────────────────┘  └──────────────────┘ │  │
│  └───────────────────────────────────┬───────────────────────────────────┘  │
│                                      │                                       │
│  ┌───────────────────────────────────┼───────────────────────────────────┐  │
│  │                          C2 LAYER   │                                │  │
│  │  ┌────────────────────────────────┴────────────────────────────────┐ │  │
│  │  │                    C2BMC Interface                               │ │  │
│  │  │  ┌───────────┐  ┌───────────┐  ┌───────────┐  ┌───────────┐    │ │  │
│  │  │  │  Link 16  │  │   JREAP   │  │  VMF      │  │  REST API │    │ │  │
│  │  │  └───────────┘  └───────────┘  └───────────┘  └───────────┘    │ │  │
│  │  └──────────────────────────────────────────────────────────────────┘ │  │
│  └───────────────────────────────────────────────────────────────────────┘  │
│                                                                               │
│  ┌────────────────────────────────────────────────────────────────────────┐ │
│  │                         DATA LAYER                                      │ │
│  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐    │ │
│  │  │  PostgreSQL │  │  TimescaleDB│  │    Redis    │  │    Kafka    │    │ │
│  │  │  (Tracks)   │  │  (Time Ser) │  │   (Cache)   │  │  (Events)   │    │ │
│  │  └─────────────┘  └─────────────┘  └─────────────┘  └─────────────┘    │ │
│  └────────────────────────────────────────────────────────────────────────┘ │
│                                                                               │
└─────────────────────────────────────────────────────────────────────────────┘
```

## Components

### 1. Sensor Input Layer

| Component | Protocol | Rate | Purpose |
|-----------|----------|------|---------|
| OPIR Sensor | Binary UDP | 1000 msg/s | Infrared detection |
| RADAR Feeds | ASTERIX | 500 msg/s | Track data |
| SBIRS Data | Binary | 100 msg/s | Strategic warning |
| DSP Sats | Binary | 50 msg/s | Launch detection |

### 2. Ingest Layer

| Component | Input | Output | Purpose |
|-----------|-------|--------|---------|
| OPIR Ingest | Binary UDP | Kafka | Detection parsing |
| Radar Gateway | ASTERIX | Kafka | Track conversion |
| SBIRS Processor | Binary | Kafka | Warning data |

### 3. Processing Layer

| Component | Input | Output | Purpose |
|-----------|-------|--------|---------|
| Sensor Fusion | Kafka tracks | Correlated tracks | MHT fusion |
| Missile Warning | Correlated tracks | Alerts | Threat estimation |
| Track Manager | All tracks | Track DB | Lifecycle management |

### 4. LVC Layer

| Component | Protocol | Purpose |
|-----------|----------|---------|
| HLA Federate | IEEE 1516 | Simulation integration |
| DIS Gateway | IEEE 1278 | PDU routing |
| LVC Coordinator | Custom | Time/entity sync |

### 5. C2 Layer

| Interface | Protocol | Purpose |
|-----------|----------|---------|
| Link 16 | MIL-STD-6016 | Tactical data |
| JREAP | MIL-STD-3011 | IP relay |
| VMF | VMF | Army C2 |
| REST API | HTTP/JSON | Modern clients |

### 6. Data Layer

| Database | Purpose | Retention |
|----------|---------|-----------|
| PostgreSQL | Track metadata | 30 days |
| TimescaleDB | Time series | 90 days |
| Redis | Cache/Session | 1 hour |
| Kafka | Event streaming | 7 days |

---

## Data Flow

```
Sensor Detection → Kafka → Fusion → Track DB → Alert → C2 Interface
                    ↓
              TimescaleDB (archive)
```

---

## Deployment

### Kubernetes Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                     KUBERNETES CLUSTER                          │
├─────────────────────────────────────────────────────────────────┤
│  Namespace: vigil                                               │
│                                                                 │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │  Deployment: opir-ingest (3 replicas)                    │   │
│  │  ┌─────┐ ┌─────┐ ┌─────┐                                 │   │
│  │  │ Pod │ │ Pod │ │ Pod │                                 │   │
│  │  └─────┘ └─────┘ └─────┘                                 │   │
│  └─────────────────────────────────────────────────────────┘   │
│                                                                 │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │  Deployment: sensor-fusion (3 replicas)                   │   │
│  │  ┌─────┐ ┌─────┐ ┌─────┐                                 │   │
│  │  │ Pod │ │ Pod │ │ Pod │                                 │   │
│  │  └─────┘ └─────┘ └─────┘                                 │   │
│  └─────────────────────────────────────────────────────────┘   │
│                                                                 │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │  Deployment: missile-warning (2 replicas)                 │   │
│  │  ┌─────┐ ┌─────┐                                         │   │
│  │  │ Pod │ │ Pod │                                         │   │
│  │  └─────┘ └─────┘                                         │   │
│  └─────────────────────────────────────────────────────────┘   │
│                                                                 │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │  StatefulSet: postgresql (3 replicas)                     │   │
│  │  ┌─────┐ ┌─────┐ ┌─────┐                                 │   │
│  │  │ P0  │ │ P1  │ │ P2  │ (Patroni HA)                    │   │
│  │  └─────┘ └─────┘ └─────┘                                 │   │
│  └─────────────────────────────────────────────────────────┘   │
│                                                                 │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │  StatefulSet: kafka (3 brokers)                           │   │
│  │  ┌─────┐ ┌─────┐ ┌─────┐                                 │   │
│  │  │ K0  │ │ K1  │ │ K2  │ (KRaft mode)                    │   │
│  │  └─────┘ └─────┘ └─────┘                                 │   │
│  └─────────────────────────────────────────────────────────┘   │
│                                                                 │
│  Services: opir-ingest, sensor-fusion, missile-warning,        │
│            postgresql, kafka, redis, grafana, prometheus       │
│                                                                 │
│  HPA: opir-ingest-hpa, sensor-fusion-hpa, missile-warning-hpa  │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

---

**Last Updated:** 2026-04-14
**Version:** 1.0