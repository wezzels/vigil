# VIGIL System Description

## Authority to Operate (ATO) Package
### System Name: VIGIL - Vital Intelligence Gathering and Integration Layer

---

## 1. Executive Summary

VIGIL (Vital Intelligence Gathering and Integration Layer) is a real-time sensor fusion and alert dissemination platform designed for missile warning and space domain awareness. The system integrates data from multiple sensor sources (OPIR, radar, SBIRS) and provides correlated tracks and alerts to command and control systems.

### Key Capabilities

- **Sensor Fusion**: Real-time correlation of tracks from multiple sensor types
- **Alert Dissemination**: NCA-formatted alerts (CONOPREP, IMMINENT, INCOMING)
- **Protocol Bridges**: HLA (IEEE 1516), DIS (IEEE 1278), JREAP (MIL-STD-3011), Link 16 (MIL-STD-6016)
- **C2 Integration**: C2BMC interface for command and control
- **High Availability**: Kubernetes-based deployment with auto-scaling

### System Classification

- **Data Classification**: UNCLASSIFIED // FOR OFFICIAL USE ONLY
- **System Type**: Mission Critical
- **Availability Requirement**: 99.9% uptime

---

## 2. System Architecture

### 2.1 High-Level Architecture

```
                    ┌─────────────────────────────────────────┐
                    │            SENSOR INPUTS                 │
                    │  ┌─────────┐  ┌─────────┐  ┌─────────┐  │
                    │  │  OPIR   │  │  Radar  │  │  SBIRS  │  │
                    │  └────┬────┘  └────┬────┘  └────┬────┘  │
                    └───────┼────────────┼────────────┼───────┘
                            │            │            │
                    ┌───────┴────────────┴────────────┴───────┐
                    │          INGESTION LAYER                │
                    │  ┌────────────────────────────────────┐ │
                    │  │  OPIR Ingest Service (Kafka)       │ │
                    │  └────────────────────────────────────┘ │
                    └───────────────────┬─────────────────────┘
                                        │
                    ┌───────────────────┴─────────────────────┐
                    │        PROCESSING LAYER                   │
                    │  ┌───────────────┐  ┌───────────────────┐ │
                    │  │ Sensor Fusion │  │ Missile Warning   │ │
                    │  │   Service     │  │    Service        │ │
                    │  └───────────────┘  └───────────────────┘ │
                    └───────────────────┬─────────────────────┘
                                        │
                    ┌───────────────────┴─────────────────────┐
                    │     DISSEMINATION LAYER                   │
                    │  ┌───────────────┐  ┌───────────────────┐ │
                    │  │ LVC           │  │   C2 Interface    │ │
                    │  │ Coordinator   │  │    (C2BMC)        │ │
                    │  └───────────────┘  └───────────────────┘ │
                    └───────────────────────────────────────────┘
```

### 2.2 Components

| Component | Function | Technology |
|-----------|----------|------------|
| OPIR Ingest | Sensor data ingestion | Go, Kafka |
| Missile Warning | Threat assessment | Go, TimescaleDB |
| Sensor Fusion | Track correlation | Go, Bayesian fusion |
| LVC Coordinator | Simulation integration | Go, HLA/DIS |
| C2 Interface | Command integration | Go, REST API |

### 2.3 Data Flow

1. **Sensor Input**: OPIR/Radar data received via Kafka
2. **Ingestion**: Data validated, transformed, and published
3. **Fusion**: Tracks correlated using Bayesian inference
4. **Warning**: Threat assessment and alert generation
5. **Dissemination**: Alerts sent via NCA format to C2BMC

---

## 3. Technical Specifications

### 3.1 Infrastructure

| Resource | Specification |
|----------|---------------|
| Platform | Kubernetes (bare-metal) |
| Nodes | 5 nodes (3 masters, 2 workers) |
| CPU | 64 cores per node |
| Memory | 256 GB per node |
| Storage | 2 TB NVMe per node |

### 3.2 Software Stack

| Layer | Technology |
|-------|------------|
| Language | Go 1.22+ |
| Database | PostgreSQL 15 + TimescaleDB |
| Cache | Redis 7 |
| Message Queue | Kafka 3.x |
| Container Runtime | containerd |
| Orchestration | Kubernetes 1.28+ |

### 3.3 Network Architecture

| Network | CIDR | Purpose |
|---------|------|---------|
| Management | 10.0.0.0/24 | Cluster management |
| Services | 10.0.1.0/24 | Service network |
| Data | 10.0.2.0/24 | Database network |

---

## 4. Security Controls

### 4.1 Access Control

| Control | Implementation |
|---------|---------------|
| Authentication | mTLS + JWT + API Keys |
| Authorization | RBAC (viewer, operator, supervisor, admin) |
| Network Segmentation | Kubernetes Network Policies |
| Secrets Management | HashiCorp Vault |

### 4.2 Data Protection

| Control | Implementation |
|---------|---------------|
| Encryption in Transit | TLS 1.2+ (mTLS internal) |
| Encryption at Rest | AES-256-GCM |
| Key Management | Vault Transit Engine |
| Audit Logging | Comprehensive security events |

### 4.3 Compliance

| Framework | Status |
|-----------|--------|
| NIST SP 800-53 | 100% compliant |
| DISA STIG | 100% compliant |
| DoD Cloud SRG | IL5 compliant |

---

## 5. Interfaces

### 5.1 External Interfaces

| Interface | Protocol | Purpose |
|-----------|----------|---------|
| OPIR Sensors | Kafka | Infrared sensor data |
| Radar Systems | Kafka | Radar track data |
| C2BMC | REST API | Command and control |
| HLA Federates | IEEE 1516 | Simulation integration |
| DIS Entities | IEEE 1278 | Distributed simulation |

### 5.2 Data Formats

| Format | Usage |
|--------|-------|
| NCA CONOPREP | Critical alerts |
| NCA IMMINENT | Warning alerts |
| NCA INCOMING | Launch notifications |
| JREAP | Tactical data links |
| Link 16 J-Series | Military messaging |

---

## 6. Personnel

### 6.1 Roles

| Role | Responsibility |
|------|----------------|
| System Owner | Overall system accountability |
| ISSO | Information System Security Officer |
| Administrator | System administration |
| Operator | Day-to-day operations |
| Viewer | Read-only access |

### 6.2 Training Requirements

| Role | Training Required |
|------|-------------------|
| All | Security Awareness (annual) |
| Administrator | Kubernetes Administration |
| ISSO | Security Management (CISSP) |

---

## 7. Appendix

### 7.1 References

- NIST SP 800-53 Rev. 5
- DISA Application STIG
- DoD Cloud SRG
- MIL-STD-3011 (JREAP)
- MIL-STD-6016 (Link 16)

### 1.2 Acronyms

| Acronym | Definition |
|---------|------------|
| ATO | Authority to Operate |
| C2BMC | Command and Control, Battle Management, and Communications |
| DIS | Distributed Interactive Simulation |
| HLA | High Level Architecture |
| ISSO | Information System Security Officer |
| JREAP | Joint Range Extension Applications Protocol |
| NCA | National Command Authority |
| OPIR | Overhead Persistent Infrared |
| RBAC | Role-Based Access Control |
| STIG | Security Technical Implementation Guide |

---

**Document Version**: 1.0
**Date**: 2026-04-14
**Classification**: UNCLASSIFIED // FOR OFFICIAL USE ONLY