# VIGIL Roadmap to 100% Production

## Executive Summary

Current state: **~60% real implementations, ~40% stubs/simulation**

Target: **100% production-ready LVC simulation federation**

Estimated timeline: **8-10 weeks** for full production fidelity

---

## Phase Overview

| Phase | Focus | Duration | Exit Criteria |
|-------|-------|----------|---------------|
| **Phase 0** | Foundation & Tests | 1 week | 80% test coverage, CI/CD |
| **Phase 1** | Real Data Feeds | 2 weeks | OPIR adapter, sensor integration |
| **Phase 2** | Protocol Bridges | 2 weeks | HLA RTI, JREAP, Link 16 |
| **Phase 3** | C2 Integration | 2 weeks | C2BMC, alert dissemination |
| **Phase 4** | Persistence & HA | 1 week | PostgreSQL, multi-node |
| **Phase 5** | Security & Certification | 1 week | mTLS, DoD compliance |
| **Phase 6** | Integration Testing | 1 week | End-to-end validation |

---

## Detailed Breakdown

### Phase 0: Foundation & Tests (Week 1)

**Goal:** Establish test coverage and CI/CD pipeline

| Task | Subtasks | Effort | Dependencies |
|------|----------|--------|--------------|
| **0.1 Unit Tests** | | | |
| | 0.1.1 Add tests for DIS PDU encoding | 4h | None |
| | 0.1.2 Add tests for coordinate transforms (ECEF↔Geodetic) | 2h | None |
| | 0.1.3 Add tests for JPDA association | 4h | None |
| | 0.1.4 Add tests for Kalman filter | 3h | None |
| | 0.1.5 Add tests for track manager | 4h | None |
| | 0.1.6 Add tests for dead reckoning algorithms | 3h | None |
| | 0.1.7 Add tests for alert doctrine rules | 2h | None |
| | 0.1.8 Add tests for replay engine PDU parsing | 3h | None |
| **0.2 Integration Tests** | | | |
| | 0.2.1 Create test Kafka cluster (docker-compose.test.yaml) | 2h | None |
| | 0.2.2 Add integration tests for opir-ingest→Kafka | 3h | 0.2.1 |
| | 0.2.3 Add integration tests for Kafka→missile-warning | 3h | 0.2.1 |
| | 0.2.4 Add integration tests for sensor-fusion pipeline | 4h | 0.2.1 |
| | 0.2.5 Add integration tests for LVC coordinator DIS output | 3h | 0.2.1 |
| **0.3 CI/CD Pipeline** | | | |
| | 0.3.1 Create GitHub Actions workflow | 2h | None |
| | 0.3.2 Add golangci-lint configuration | 1h | None |
| | 0.3.3 Add gosec security scanner | 1h | None |
| | 0.3.4 Add test coverage reporting | 1h | 0.1.x |
| | 0.3.5 Add Docker image build & push | 2h | None |
| **0.4 Documentation** | | | |
| | 0.4.1 Create ARCHITECTURE.md | 3h | None |
| | 0.4.2 Create API.md with endpoint docs | 2h | None |
| | 0.4.3 Create DEPLOYMENT.md | 2h | None |

**Phase 0 Total:** ~40 hours (1 week)

---

### Phase 1: Real Data Feeds (Weeks 2-3)

**Goal:** Replace simulation with real sensor data

| Task | Subtasks | Effort | Dependencies |
|------|----------|--------|--------------|
| **1.1 OPIR Adapter** | | | |
| | 1.1.1 Define OPIRDataFeed interface | 2h | None |
| | 1.1.2 Implement SBIRS-High data adapter | 8h | 1.1.1 |
| | 1.1.3 Implement NG-OPIR data adapter | 8h | 1.1.1 |
| | 1.1.4 Add configuration for sensor endpoints | 2h | 1.1.2, 1.1.3 |
| | 1.1.5 Add reconnection logic with backoff | 3h | 1.1.2, 1.1.3 |
| | 1.1.6 Add data validation and sanitization | 3h | 1.1.2, 1.1.3 |
| | 1.1.7 Unit tests for OPIR adapters | 4h | 1.1.2-1.1.6 |
| **1.2 Radar Integration** | | | |
| | 1.2.1 Define RadarDataFeed interface | 2h | None |
| | 1.2.2 Implement AN/TPY-2 radar adapter | 6h | 1.2.1 |
| | 1.2.3 Implement SBX radar adapter | 6h | 1.2.1 |
| | 1.2.4 Implement UEWR radar adapter | 4h | 1.2.1 |
| | 1.2.5 Add radar track correlation | 4h | 1.2.2-1.2.4 |
| | 1.2.6 Unit tests for radar adapters | 3h | 1.2.2-1.2.5 |
| **1.3 Sensor Fusion Enhancement** | | | |
| | 1.3.1 Add multi-hypothesis tracking (MHT) | 8h | None |
| | 1.3.2 Add track scoring and confidence | 4h | 1.3.1 |
| | 1.3.3 Add sensor registration correction | 4h | None |
| | 1.3.4 Add time alignment for asynchronous sensors | 3h | None |
| | 1.3.5 Unit tests for MHT | 4h | 1.3.1-1.3.4 |
| **1.4 Mode Switching** | | | |
| | 1.4.1 Implement mode=real vs mode=simulate switch | 2h | None |
| | 1.4.2 Add replay mode for recorded data | 4h | None |
| | 1.4.3 Add hybrid mode (real + simulated) | 3h | 1.4.1, 1.4.2 |
| | 1.4.4 Configuration for mode selection | 1h | 1.4.1-1.4.3 |

**Phase 1 Total:** ~80 hours (2 weeks)

---

### Phase 2: Protocol Bridges (Weeks 4-5)

**Goal:** Enable multi-federation interoperability

| Task | Subtasks | Effort | Dependencies |
|------|----------|--------|--------------|
| **2.1 HLA RTI Integration** | | | |
| | 2.1.1 Evaluate RTI options (Portico vs makRTI) | 4h | None |
| | 2.1.2 Add Portico RTI dependency | 1h | 2.1.1 |
| | 2.1.3 Implement FOM parsing from XML | 6h | None |
| | 2.1.4 Implement HLA object class publishing | 8h | 2.1.3 |
| | 2.1.5 Implement HLA object class subscription | 6h | 2.1.4 |
| | 2.1.6 Implement HLA interaction publishing | 4h | 2.1.3 |
| | 2.1.7 Implement HLA interaction subscription | 4h | 2.1.6 |
| | 2.1.8 Add federation join/ resign logic | 3h | 2.1.4-2.1.7 |
| | 2.1.9 Unit tests for HLA bridge | 4h | 2.1.4-2.1.8 |
| **2.2 DIS Gateway** | | | |
| | 2.2.1 Implement DIS UDP receiver | 3h | None |
| | 2.2.2 Implement DIS UDP transmitter | 3h | None |
| | 2.2.3 Add DIS exercise management | 4h | 2.2.1, 2.2.2 |
| | 2.2.4 Add DIS entity state PDU full implementation | 4h | None |
| | 2.2.5 Add DIS fire PDU implementation | 3h | None |
| | 2.2.6 Add DIS detonation PDU implementation | 3h | None |
| | 2.2.7 Add DIS electromagnetic emission PDU | 4h | None |
| | 2.2.8 Unit tests for DIS gateway | 4h | 2.2.1-2.2.7 |
| **2.3 JREAP Bridge** | | | |
| | 2.3.1 Implement JREAP-A (Serial) adapter | 6h | None |
| | 2.3.2 Implement JREAP-B (IP) adapter | 6h | None |
| | 2.3.3 Implement JREAP-C (Satellite) adapter | 8h | None |
| | 2.3.4 Add JREAP message parsing (MIL-STD-3011) | 8h | None |
| | 2.3.5 Add JREAP message generation | 6h | 2.3.4 |
| | 2.3.6 Unit tests for JREAP bridge | 4h | 2.3.1-2.3.5 |
| **2.4 Link 16 Bridge** | | | |
| | 2.4.1 Implement J-series message parsing | 8h | None |
| | 2.4.2 Implement J-series message generation | 6h | 2.4.1 |
| | 2.4.3 Add J3.2 (Air Track) support | 4h | 2.4.1 |
| | 2.4.4 Add J7.0 (Track Management) support | 4h | 2.4.1 |
| | 2.4.5 Add J12.0 (Mission Assignment) support | 3h | 2.4.1 |
| | 2.4.6 Unit tests for Link 16 bridge | 4h | 2.4.1-2.4.5 |

**Phase 2 Total:** ~100 hours (2.5 weeks)

---

### Phase 3: C2 Integration (Weeks 6-7)

**Goal:** Connect to real C2 systems

| Task | Subtasks | Effort | Dependencies |
|------|----------|--------|--------------|
| **3.1 C2BMC Interface** | | | |
| | 3.1.1 Define C2BMC API interface | 4h | None |
| | 3.1.2 Implement C2BMC REST client | 6h | 3.1.1 |
| | 3.1.3 Implement alert submission to C2BMC | 4h | 3.1.2 |
| | 3.1.4 Implement track correlation with C2BMC | 6h | 3.1.2 |
| | 3.1.5 Add C2BMC authentication (PKI) | 4h | 3.1.2 |
| | 3.1.6 Unit tests for C2BMC interface | 3h | 3.1.2-3.1.5 |
| **3.2 Alert Dissemination** | | | |
| | 3.2.1 Implement NCA alert formatting | 4h | None |
| | 3.2.2 Implement alert prioritization queue | 3h | None |
| | 3.2.3 Implement alert acknowledgment handling | 3h | 3.2.2 |
| | 3.2.4 Add alert escalation logic | 3h | 3.2.2 |
| | 3.2.5 Implement alert delivery confirmation | 3h | 3.2.2 |
| | 3.2.6 Add multiple recipient handling | 3h | None |
| | 3.2.7 Unit tests for alert dissemination | 3h | 3.2.1-3.2.6 |
| **3.3 Tactical Data Links** | | | |
| | 3.3.1 Implement TADIL-A message formatting | 4h | None |
| | 3.3.2 Implement TADIL-J message formatting | 6h | None |
| | 3.3.3 Add VMF (Variable Message Format) support | 4h | None |
| | 3.3.4 Unit tests for tactical data links | 3h | 3.3.1-3.3.3 |
| **3.4 External System Interfaces** | | | |
| | 3.4.1 Implement JTAGS interface | 6h | None |
| | 3.4.2 Implement USMTF message generation | 4h | None |
| | 3.4.3 Implement ADatP-3 message generation | 4h | None |
| | 3.4.4 Unit tests for external interfaces | 3h | 3.4.1-3.4.3 |

**Phase 3 Total:** ~80 hours (2 weeks)

---

### Phase 4: Persistence & HA (Week 8)

**Goal:** Production-grade data storage and high availability

| Task | Subtasks | Effort | Dependencies |
|------|----------|--------|--------------|
| **4.1 PostgreSQL Schema** | | | |
| | 4.1.1 Design track database schema | 3h | None |
| | 4.1.2 Create migrations for tracks table | 2h | 4.1.1 |
| | 4.1.3 Create migrations for alerts table | 1h | 4.1.1 |
| | 4.1.4 Create migrations for events table | 1h | 4.1.1 |
| | 4.1.5 Create migrations for entities table | 1h | 4.1.1 |
| | 4.1.6 Add indexes for common queries | 2h | 4.1.2-4.1.5 |
| | 4.1.7 Create database repository layer | 4h | 4.1.2-4.1.5 |
| **4.2 Time-Series Storage** | | | |
| | 4.2.1 Integrate TimescaleDB for track history | 4h | 4.1.7 |
| | 4.2.2 Add continuous aggregates for metrics | 3h | 4.2.1 |
| | 4.2.3 Create retention policies | 1h | 4.2.1 |
| **4.3 Redis Enhancement** | | | |
| | 4.3.1 Add Redis for track state caching | 2h | None |
| | 4.3.2 Add Redis for session management | 2h | None |
| | 4.3.3 Add Redis for pub/sub coordination | 2h | None |
| **4.4 High Availability** | | | |
| | 4.4.1 Add Kafka consumer groups | 2h | None |
| | 4.4.2 Add leader election for coordinators | 4h | None |
| | 4.4.3 Add health check endpoints | 2h | None |
| | 4.4.4 Add graceful shutdown handling | 2h | None |
| | 4.4.5 Add Kubernetes deployment manifests | 3h | 4.4.1-4.4.4 |
| | 4.4.6 Add horizontal pod autoscaling | 2h | 4.4.5 |

**Phase 4 Total:** ~40 hours (1 week)

---

### Phase 5: Security & Certification (Week 9)

**Goal:** DoD-compliant security posture

| Task | Subtasks | Effort | Dependencies |
|------|----------|--------|--------------|
| **5.1 Authentication** | | | |
| | 5.1.1 Add mTLS for service-to-service | 4h | None |
| | 5.1.2 Add PKI certificate management | 3h | 5.1.1 |
| | 5.1.3 Add JWT token validation | 3h | None |
| | 5.1.4 Add API key management | 2h | None |
| **5.2 Authorization** | | | |
| | 5.2.1 Add RBAC middleware | 4h | None |
| | 5.2.2 Add role definitions (admin, operator, viewer) | 2h | 5.2.1 |
| | 5.2.3 Add audit logging | 3h | 5.2.1 |
| **5.3 Network Security** | | | |
| | 5.3.1 Add network policies (K8s) | 2h | None |
| | 5.3.2 Add service mesh (Istio) integration | 4h | None |
| | 5.3.3 Add secrets management (Vault) | 4h | None |
| **5.4 Compliance** | | | |
| | 5.4.1 Add STIG checklist validation | 4h | None |
| | 5.4.2 Add security scanning to CI/CD | 2h | None |
| | 5.4.3 Add vulnerability reporting | 2h | None |
| | 5.4.4 Create Authority to Operate (ATO) package | 8h | 5.1-5.4 |

**Phase 5 Total:** ~50 hours (1.25 weeks)

---

### Phase 6: Integration Testing (Week 10)

**Goal:** End-to-end validation

| Task | Subtasks | Effort | Dependencies |
|------|----------|--------|--------------|
| **6.1 End-to-End Tests** | | | |
| | 6.1.1 Create sensor-to-alert E2E test | 4h | Phase 1-5 |
| | 6.1.2 Create track lifecycle E2E test | 4h | Phase 1-5 |
| | 6.1.3 Create federation E2E test | 4h | Phase 2 |
| | 6.1.4 Create C2 E2E test | 4h | Phase 3 |
| **6.2 Performance Testing** | | | |
| | 6.2.1 Create load test for OPIR ingest | 4h | None |
| | 6.2.2 Create load test for track correlation | 4h | None |
| | 6.2.3 Create latency benchmarks | 4h | 6.2.1, 6.2.2 |
| | 6.2.4 Tune for target latency (<100ms) | 4h | 6.2.3 |
| **6.3 Chaos Engineering** | | | |
| | 6.3.1 Add network partition tests | 3h | None |
| | 6.3.2 Add node failure tests | 3h | None |
| | 6.3.3 Add Kafka failure tests | 3h | None |
| **6.4 Documentation Finalization** | | | |
| | 6.4.1 Update API documentation | 2h | None |
| | 6.4.2 Create operator runbook | 4h | None |
| | 6.4.3 Create troubleshooting guide | 3h | None |
| | 6.4.4 Create deployment checklist | 2h | None |

**Phase 6 Total:** ~50 hours (1.25 weeks)

---

## Resource Requirements

### Personnel
| Role | Weeks 1-2 | Weeks 3-5 | Weeks 6-7 | Weeks 8-10 |
|------|-----------|-----------|-----------|------------|
| Backend Engineer | 1 | 1 | 1 | 1 |
| Protocol Specialist | - | 1 | 0.5 | - |
| DevOps Engineer | 0.5 | 0.5 | 0.5 | 0.5 |
| Security Engineer | - | - | - | 0.5 |
| QA Engineer | 0.5 | 0.5 | 0.5 | 1 |

### Infrastructure
- Kafka cluster (3 nodes)
- PostgreSQL cluster (3 nodes)
- Redis cluster (3 nodes)
- Kubernetes cluster (3+ nodes)
- HLA RTI license (if makRTI)

---

## Risk Assessment

| Risk | Probability | Impact | Mitigation |
|------|-------------|--------|------------|
| HLA RTI licensing issues | Medium | High | Use Portico (open source) as fallback |
| Real sensor data unavailable | Low | Critical | Use high-fidelity simulation mode |
| C2BMC interface changes | Medium | Medium | Abstract interface layer |
| Performance under load | Medium | High | Early load testing, optimize incrementally |
| Security certification delays | Medium | High | Start ATO process early |

---

## Success Metrics

| Metric | Target | Measurement |
|--------|--------|-------------|
| Test coverage | >80% | `go test -cover` |
| End-to-end latency | <100ms | P99 latency |
| Track correlation accuracy | >95% | Sim vs real comparison |
| Uptime | >99.9% | 30-day rolling |
| Alert delivery time | <5s | P99 |
| Concurrent tracks | >10,000 | Load test |

---

## Timeline Visualization

```
Week 1:  [Phase 0] Foundation & Tests
Week 2:  [Phase 1] OPIR Adapter + Radar Integration
Week 3:  [Phase 1] Sensor Fusion Enhancement
Week 4:  [Phase 2] HLA RTI Integration
Week 5:  [Phase 2] DIS + JREAP + Link 16 Bridges
Week 6:  [Phase 3] C2BMC Interface
Week 7:  [Phase 3] Alert Dissemination + Tactical Links
Week 8:  [Phase 4] PostgreSQL + HA
Week 9:  [Phase 5] Security + Certification
Week 10: [Phase 6] Integration Testing
```

---

## Appendix: Technology Stack

| Layer | Technology | Purpose |
|-------|------------|---------|
| Language | Go 1.21+ | Performance, concurrency |
| Messaging | Kafka | Event streaming |
| Database | PostgreSQL + TimescaleDB | Persistent storage |
| Cache | Redis | State management |
| HLA RTI | Portico / makRTI | Federation |
| Deployment | Kubernetes | Orchestration |
| Monitoring | Prometheus + Grafana | Observability |
| Logging | Loki | Log aggregation |
| Service Mesh | Istio (optional) | mTLS, traffic management |

---

## Appendix: Milestones

| Milestone | Date | Deliverable |
|-----------|------|-------------|
| M1: Foundation Complete | Week 1 | Tests passing, CI/CD green |
| M2: Real Data Flowing | Week 3 | OPIR/Radar feeding Kafka |
| M3: Federation Ready | Week 5 | HLA + DIS bridges operational |
| M4: C2 Connected | Week 7 | Alerts reaching C2BMC |
| M5: Production Ready | Week 9 | Security certified |
| M6: Operational | Week 10 | ATO approved, deployed |