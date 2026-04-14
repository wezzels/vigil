# VIGIL Data Flow Diagram

## Overview

This document describes the data flows within VIGIL system.

---

## 1. Sensor Data Ingestion Flow

```
┌─────────────┐        ┌─────────────┐        ┌─────────────┐
│   OPIR      │        │   OPIR      │        │   OPIR      │
│   Sensor    │───────▶│   Ingest    │───────▶│   Kafka     │
│   (1)       │        │   Service   │        │   Topic     │
└─────────────┘        └─────────────┘        └─────────────┘
       │                      │                      │
       │ Data:                │ Process:             │ Store:
       │ - Timestamp          │ - Validate           │ - opir-detections
       │ - Location           │ - Transform          │ - track-updates
       │ - Signature          │ - Timestamp          │ - correlated-tracks
       │ - Confidence         │ - Publish            │
       │                      │                      │
       └──────────────────────┴──────────────────────┘
```

### Data Elements

| Element | Type | Classification | Encryption |
|---------|------|----------------|------------|
| Timestamp | int64 | FOUO | Transit |
| Location | float64[] | FOUO | Transit |
| Signature | string | FOUO | Transit |
| Confidence | float64 | FOUO | Transit |

---

## 2. Track Correlation Flow

```
┌─────────────┐        ┌─────────────┐        ┌─────────────┐
│   Kafka     │        │   Sensor    │        │ PostgreSQL │
│   Topics    │───────▶│   Fusion    │───────▶│ TimescaleDB│
│   (2)       │        │   Service   │        │   (4)      │
└─────────────┘        └─────────────┘        └─────────────┘
       │                      │                      │
       │ Sources:             │ Process:             │ Store:
       │ - OPIR tracks        │ - Correlation        │ - tracks
       │ - Radar tracks       │ - Bayesian fusion    │ - track_history
       │ - SBIRS tracks       │ - Quality check      │ - correlations
       │                      │ - State update       │
       │                      │                      │
       └──────────────────────┴──────────────────────┘
```

### Correlation Process

1. **Ingest**: Receive tracks from multiple sources
2. **Gate**: Apply spatial/velocity gates
3. **Associate**: Munkres algorithm for optimal assignment
4. **Fuse**: Bayesian belief propagation
5. **Store**: Persist correlated tracks

---

## 3. Alert Generation Flow

```
┌─────────────┐        ┌─────────────┐        ┌─────────────┐
│ Correlated  │        │   Missile   │        │   Alert     │
│   Tracks    │───────▶│   Warning   │───────▶│   Queue     │
│   (3)       │        │   Service   │        │   (5)       │
└─────────────┘        └─────────────┘        └─────────────┘
       │                      │                      │
       │ Input:               │ Process:             │ Output:
       │ - Track ID           │ - Threat assess      │ - Alert ID
       │ - Velocity            │ - Priority rank     │ - Priority
       │ - Location            │ - Alert create      │ - Format
       │ - Intent             │ - Queue             │ - Recipients
       │                      │                      │
       └──────────────────────┴──────────────────────┘
```

### Alert Priorities

| Priority | Description | Response Time |
|----------|-------------|---------------|
| Critical | Immediate threat | < 30 seconds |
| Imminent | High probability threat | < 2 minutes |
| Warning | Potential threat | < 5 minutes |
| Watch | Monitoring | < 15 minutes |

---

## 4. Alert Dissemination Flow

```
┌─────────────┐        ┌─────────────┐        ┌─────────────┐
│   Alert     │        │    LVC      │        │   C2BMC     │
│   Queue     │───────▶│ Coordinator │───────▶│   Interface │
│   (5)       │        │   (6)       │        │   (7)       │
└─────────────┘        └─────────────┘        └─────────────┘
       │                      │                      │
       │ Process:             │ Process:             │ Output:
       │ - Dequeue           │ - Format (NCA)      │ - CONOPREP
       │ - Route              │ - Ack/Timeout       │ - IMMINENT
       │ - Retry              │ - Escalate          │ - INCOMING
       │                      │ - Log delivery      │
       │                      │                      │
       └──────────────────────┴──────────────────────┘
```

### Delivery Status

| Status | Description |
|--------|-------------|
| Pending | Waiting for delivery |
| Sent | Sent to recipient |
| Acknowledged | Recipient acknowledged |
| Rejected | Recipient rejected |
| Timeout | No acknowledgment |

---

## 5. C2 Interface Flow

```
┌─────────────┐        ┌─────────────┐        ┌─────────────┐
│   VIGIL     │  REST  │   C2BMC     │  mTLS  │   C2BMC     │
│   Services  │◀──────▶│   Interface │◀──────▶│   System    │
│   (8)       │        │   (7)       │        │   (9)       │
└─────────────┘        └─────────────┘        └─────────────┘
       │                      │                      │
       │ Endpoints:           │ Protocol:            │ Operations:
       │ - /api/tracks       │ - REST over TLS     │ - Create alert
       │ - /api/alerts       │ - mTLS auth         │ - Get tracks
       │ - /api/events       │ - JWT tokens        │ - Update tracks
       │                      │ - API keys          │ - Delete tracks
       │                      │                      │
       └──────────────────────┴──────────────────────┘
```

### API Endpoints

| Endpoint | Method | Auth | Description |
|----------|--------|------|-------------|
| /api/tracks | GET | mTLS/JWT | Get correlated tracks |
| /api/tracks/:id | GET | mTLS/JWT | Get track by ID |
| /api/alerts | POST | mTLS/JWT | Create alert |
| /api/alerts/:id/ack | POST | mTLS/JWT | Acknowledge alert |
| /api/events | GET | mTLS/JWT | Get events |

---

## 6. Simulation Integration Flow

```
┌─────────────┐        ┌─────────────┐        ┌─────────────┐
│   HLA/DIS   │        │    LVC      │        │   VIGIL     │
│ Federates   │◀──────▶│ Coordinator │◀──────▶│   Services  │
│   (10)      │        │   (6)       │        │   (8)       │
└─────────────┘        └─────────────┘        └─────────────┘
       │                      │                      │
       │ Protocols:           │ Functions:           │ Data:
       │ - HLA (IEEE 1516)   │ - Protocol bridge    │ - Entity state
       │ - DIS (IEEE 1278)   │ - Entity mapping     │ - Fire/Detonation
       │ - JREAP (MIL-STD)   │ - Time sync          │ - Track updates
       │ - Link 16 (J-Ser)   │ - Coordinate conv    │ - Correlations
       │                      │                      │
       └──────────────────────┴──────────────────────┘
```

---

## Data Flow Summary

| Flow | Source | Destination | Data | Protocol |
|------|--------|-------------|------|----------|
| 1 | OPIR Sensors | OPIR Ingest | Detections | Kafka |
| 2 | Kafka Topics | Sensor Fusion | Tracks | Kafka |
| 3 | Sensor Fusion | PostgreSQL | Correlated Tracks | SQL |
| 4 | Correlated Tracks | Missile Warning | Threat Data | Kafka |
| 5 | Missile Warning | Alert Queue | Alerts | Kafka |
| 6 | Alert Queue | LVC Coordinator | Formatted Alerts | Kafka |
| 7 | LVC Coordinator | C2BMC Interface | NCA Alerts | REST |
| 8 | C2BMC Interface | C2BMC System | Commands | REST/mTLS |
| 9 | HLA/DIS Federates | LVC Coordinator | Simulation Data | HLA/DIS |
| 10 | LVC Coordinator | VIGIL Services | Entity Data | REST |

---

## Security Controls by Flow

| Flow | Encryption | Authentication | Authorization |
|------|------------|----------------|---------------|
| 1-2 | TLS | Certificates | N/A |
| 3-4 | TLS | Service Account | RBAC |
| 5-6 | TLS | Service Account | RBAC |
| 7-8 | mTLS | PKI | RBAC |
| 9-10 | TLS | Certificates | N/A |

---

**Document Version**: 1.0
**Date**: 2026-04-14