# VIGIL Architecture

## System Overview

VIGIL is a distributed LVC (Live, Virtual, Constructive) simulation federation platform designed for DoD missile warning and sensor fusion operations. It provides real-time track correlation, threat assessment, and alert dissemination capabilities.

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                           VIGIL SYSTEM ARCHITECTURE                          │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│   ┌──────────────┐     ┌──────────────┐     ┌──────────────┐               │
│   │   OPIR       │     │   RADAR      │     │   IFF/SSR    │               │
│   │   Sensors    │     │   Sensors    │     │   Sensors    │               │
│   └──────┬───────┘     └──────┬───────┘     └──────┬───────┘               │
│          │                    │                    │                        │
│          └────────────────────┼────────────────────┘                        │
│                               │                                             │
│                               ▼                                             │
│   ┌───────────────────────────────────────────────────────────────────┐     │
│   │                    OPIR-INGEST / SENSOR-INGEST                    │     │
│   │                    (Kafka Producers)                              │     │
│   └───────────────────────────────┬───────────────────────────────────┘     │
│                                   │                                          │
│                                   ▼                                          │
│   ┌───────────────────────────────────────────────────────────────────┐     │
│   │                         KAFKA CLUSTER                              │     │
│   │  ┌─────────┐  ┌─────────┐  ┌─────────┐  ┌─────────┐  ┌─────────┐  │     │
│   │  │ opir-   │  │ radar-  │  │ track-  │  │correlat│  │ alerts  │  │     │
│   │  │detections│  │ tracks  │  │ updates │  │ed-tracks│  │         │  │     │
│   │  └─────────┘  └─────────┘  └─────────┘  └─────────┘  └─────────┘  │     │
│   └───────────────────────────────┬───────────────────────────────────┘     │
│                                   │                                          │
│          ┌────────────────────────┼────────────────────────┐                 │
│          │                        │                        │                 │
│          ▼                        ▼                        ▼                 │
│   ┌──────────────┐     ┌──────────────────┐     ┌──────────────────┐         │
│   │SENSOR-FUSION │     │ MISSILE-WARNING  │     │  LVC-COORDINATOR │         │
│   │              │     │     ENGINE       │     │                  │         │
│   │ - JPDA       │     │                  │     │ - DIS Protocol  │         │
│   │ - Kalman     │     │ - Alert Doctrine │     │ - Dead Reckoning│         │
│   │ - Track Mgmt │     │ - Threat Assess  │     │ - Entity Mgmt   │         │
│   └──────┬───────┘     └────────┬─────────┘     └────────┬─────────┘         │
│          │                      │                        │                  │
│          └──────────────────────┼────────────────────────┘                  │
│                                 │                                           │
│                                 ▼                                           │
│   ┌───────────────────────────────────────────────────────────────────┐     │
│   │                    ALERT-DISSEMINATION                             │     │
│   │                    (NCA/Pentagon Distribution)                     │     │
│   └───────────────────────────────────────────────────────────────────┘     │
│                                                                              │
│   ┌───────────────────────────────────────────────────────────────────┐     │
│   │                    REPLAY-ENGINE                                   │     │
│   │                    (Mission Recording & Playback)                   │     │
│   └───────────────────────────────────────────────────────────────────┘     │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘
```

## Components

### Core Services

| Service | Port | Description |
|---------|------|-------------|
| `opir-ingest` | 8080 | OPIR satellite data ingestion |
| `sensor-fusion` | 8081 | Multi-source track fusion |
| `missile-warning-engine` | 8082 | Threat detection & alert generation |
| `alert-dissemination` | 8083 | Alert distribution to C2 systems |
| `lvc-coordinator` | 8084 | DIS entity management |
| `replay-engine` | 8085 | Mission recording & playback |

### Data Flow

```
Sensors → OPIR/Radar Ingest → Kafka → Sensor Fusion → Track Correlation
                                              ↓
                                    Missile Warning Engine
                                              ↓
                                    Alert Dissemination
                                              ↓
                                    C2 Systems / Displays
```

### Topics

| Topic | Purpose |
|-------|---------|
| `opir-detections` | Raw OPIR sensor detections |
| `radar-tracks` | Radar track data |
| `track-updates` | Track state updates |
| `correlated-tracks` | Fused track output |
| `c2-messages` | C2 system messages |
| `alerts` | Alert notifications |
| `link16-reports` | Link 16 J-Series messages |
| `jreap-messages` | JREAP messages |

## Technology Stack

### Core Technologies

- **Go 1.21+**: Primary programming language
- **Apache Kafka**: Message broker for event streaming
- **Redis**: Caching and state management
- **PostgreSQL**: Persistent data storage

### Protocols

- **DIS (IEEE 1278.1-2012)**: Distributed Interactive Simulation
- **Link 16 (MIL-STD-6016)**: Tactical data link
- **JREAP (MIL-STD-3011)**: Joint Range Extension Applications Protocol

### Deployment

- **Docker**: Container runtime
- **Kubernetes**: Container orchestration
- **Helm**: Kubernetes package manager

## Data Models

### Track

```go
type Track struct {
    ID           uint64    // Unique track ID
    TrackNumber  uint32    // Sequential track number
    Lat          float64   // Latitude (degrees)
    Lon          float64   // Longitude (degrees)
    Alt          float64   // Altitude (meters)
    Velocity     Vector3   // Velocity vector (m/s)
    Heading      float64   // Heading (degrees)
    Confidence   float64   // Track confidence (0.0-1.0)
    SourceCount  int       // Number of contributing sources
    VarLat       float64   // Latitude variance
    VarLon       float64   // Longitude variance
    VarAlt       float64   // Altitude variance
    LastUpdate   int64     // Unix milliseconds
    Sources      []string  // Sensor sources
}
```

### Alert

```go
type Alert struct {
    ID           uint64    // Unique alert ID
    TrackNumber  uint32    // Associated track
    AlertLevel   int       // CONOPREP=1, IMMINENT=2, INCOMING=3, HOSTILE=4
    ThreatType   int       // BALLISTIC, CRUISE, AIRCRAFT, UAV, ARTILLERY
    LaunchPoint  LatLonAlt // Launch location
    ImpactPoint  LatLonAlt // Predicted impact
    LaunchTime   int64     // Launch time (Unix ms)
    ImpactTime   int64     // Predicted impact time (Unix ms)
    Confidence   float64   // Alert confidence
    SourceCount  int       // Number of sources
    Heading      float64   // Heading (degrees)
    Speed        float64   // Speed (m/s)
}
```

## Performance Requirements

| Metric | Requirement |
|--------|-------------|
| Track Processing Latency | < 100ms |
| Alert Generation Latency | < 50ms |
| Track Correlation Capacity | 10,000+ tracks |
| Alert Throughput | 1,000+ alerts/second |
| DIS PDU Rate | 50 PDUs/second/entity |
| System Uptime | 99.9% |

## Security

### Authentication

- mTLS for inter-service communication
- JWT tokens for API access
- Role-based access control (RBAC)

### Network Segmentation

- Management network (secure)
- Sensor network (air-gapped)
- C2 network (classified)

### Data Classification

- UNCLASSIFIED: Training data
- SECRET: Operational data
- TOP SECRET: Sensor parameters

## Dependencies

```
vigil/
├── dis-pdu/           # DIS protocol implementation
├── pkg/
│   ├── fusion/        # Track fusion algorithms
│   ├── doctrine/      # Alert doctrine
│   └── coords/        # Coordinate transformations
├── apps/
│   ├── opir-ingest/   # OPIR data ingestion
│   ├── sensor-fusion/ # Track fusion
│   ├── missile-warning-engine/ # Threat detection
│   ├── alert-dissemination/ # Alert distribution
│   ├── lvc-coordinator/ # DIS management
│   └── replay-engine/ # Mission replay
└── docs/              # Documentation
```

## References

- [IEEE 1278.1-2012](https://standards.ieee.org/): DIS Protocol
- [MIL-STD-6016](https://quicksearch.dla.mil/): Link 16
- [MIL-STD-3011](https://quicksearch.dla.mil/): JREAP
- [STANAG 4609](https://www.nato.int/): NATO ISR Standards