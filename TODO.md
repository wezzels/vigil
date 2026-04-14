# VIGIL TODO

## Status Legend
- 🔴 Not started
- 🟡 In progress
- ✅ Complete
- ⏸️ Blocked

---

## Phase 0: Foundation & Tests (Week 1)

### 0.1 Unit Tests

#### 0.1.1 DIS PDU Encoding Tests
- ✅ Create `dis-pdu/pdu_test.go`
- ✅ Add test for Entity State PDU encoding
- ✅ Add test for Entity State PDU decoding
- ✅ Add test for Fire PDU encoding
- ✅ Add test for Detonation PDU encoding
- ✅ Add test for PDU timestamp conversion
- ✅ Add test for ECEF↔Geodetic coordinate conversion
- ✅ Add benchmark for PDU encoding performance

#### 0.1.2 Coordinate Transform Tests
- 🔴 Create `pkg/coords/coords_test.go`
- 🔴 Add test for geodetic to ECEF conversion
- 🔴 Add test for ECEF to geodetic conversion
- 🔴 Add test for LLA to MGRS conversion
- 🔴 Add test for coordinate precision (<1m error)
- 🔴 Add benchmark for coordinate transforms

#### 0.1.3 JPDA Association Tests
- ✅ Create `apps/sensor-fusion/jpda_test.go`
- ✅ Add test for single-target single-measurement association
- ✅ Add test for single-target multi-measurement association
- ✅ Add test for multi-target single-measurement association
- ✅ Add test for multi-target multi-measurement association
- ✅ Add test for clutter rejection
- ✅ Add test for track initiation
- ✅ Add test for track deletion
- ✅ Add benchmark for JPDA with 100+ tracks

#### 0.1.4 Kalman Filter Tests
- ✅ Create `apps/sensor-fusion/kalman_test.go`
- ✅ Add test for Kalman predict step
- ✅ Add test for Kalman update step
- ✅ Add test for extended Kalman (EKF) predict
- ✅ Add test for extended Kalman (EKF) update
- 🔴 Add test for unscented Kalman (UKF) sigma points
- ✅ Add test for measurement noise handling
- ✅ Add test for process noise handling
- ✅ Add benchmark for Kalman update rate

#### 0.1.5 Track Manager Tests
- 🔴 Create `apps/missile-warning-engine/trackmanager_test.go`
- 🔴 Add test for track creation
- 🔴 Add test for track update
- 🔴 Add test for track deletion
- 🔴 Add test for track correlation
- 🔴 Add test for threat type estimation
- 🔴 Add test for alert level escalation
- 🔴 Add test for track aging
- 🔴 Add test for track cleanup

#### 0.1.6 Dead Reckoning Tests
- ✅ Create `dis-pdu/deadreckoning_test.go`
- ✅ Add test for DRM_FVM (Fixed Velocity)
- ✅ Add test for DRM_RPW (Rest of World)
- ✅ Add test for DRM_RVW (Relative Velocity)
- ✅ Add test for position extrapolation accuracy
- ✅ Add test for velocity decay
- ✅ Add test for orientation interpolation
- ✅ Add benchmark for dead reckoning calculations

#### 0.1.7 Alert Doctrine Tests
- ✅ Create `pkg/doctrine/doctrine_test.go`
- ✅ Add test for CONOPREP alert rule
- ✅ Add test for IMMINENT alert rule
- ✅ Add test for INCOMING alert rule
- ✅ Add test for HOSTILE alert rule
- ✅ Add test for alert escalation
- ✅ Add test for alert de-escalation
- ✅ Add test for multiple threat types

#### 0.1.8 Replay Engine Tests
- ✅ Create `apps/replay-engine/replay_test.go`
- ✅ Add test for PDU serialization
- ✅ Add test for recording metadata
- ✅ Add test for time compression
- ✅ Add test for byte formatting
- 🟡 Add test for Entity State PDU parsing (partial)
- 🟡 Add test for recording start/stop (partial)
- 🔴 Add test for playback with time scaling
- 🔴 Add benchmark for PDU parsing rate

### 0.2 Integration Tests

#### 0.2.1 Test Infrastructure
- ✅ Create `docker-compose.test.yaml`
- ✅ Add test Kafka cluster
- ✅ Add test Redis
- ✅ Add test PostgreSQL
- 🔴 Add wait-for-it scripts
- 🔴 Create Makefile test targets

#### 0.2.2 OPIR Ingest Integration
- ✅ Create `apps/opir-ingest/integration_test.go`
- 🔴 Add test for Kafka topic creation
- ✅ Add test for message publishing
- ✅ Add test for message serialization
- ✅ Add test for health endpoint
- 🔴 Add test for metrics endpoint

#### 0.2.3 Missile Warning Integration
- ✅ Create `apps/missile-warning-engine/integration_test.go`
- ✅ Add test for Kafka consumer setup
- ✅ Add test for track creation flow
- ✅ Add test for alert generation
- ✅ Add test for health endpoint

#### 0.2.4 Sensor Fusion Integration
- ✅ Create `apps/sensor-fusion/integration_test.go`
- ✅ Add test for multi-source track input
- ✅ Add test for track correlation output
- ✅ Add test for fused track publication
- ✅ Add test for health endpoint

#### 0.2.5 LVC Coordinator Integration
- ✅ Create `apps/lvc-coordinator/integration_test.go`
- ✅ Add test for entity creation
- ✅ Add test for DIS PDU publication
- ✅ Add test for dead reckoning
- ✅ Add test for health endpoint

### 0.3 CI/CD Pipeline

#### 0.3.1 GitHub Actions
- ✅ Create `.github/workflows/ci.yaml`
- ✅ Add checkout step
- ✅ Add Go setup step
- ✅ Add cache configuration
- ✅ Add test step
- ✅ Add coverage upload

#### 0.3.2 Linting
- ✅ Create `.golangci.yml`
- ✅ Enable errcheck linter
- ✅ Enable govet linter
- ✅ Enable staticcheck linter
- ✅ Enable unused linter
- ✅ Enable ineffassign linter
- ✅ Add lint step to CI

#### 0.3.3 Security Scanning
- ✅ Add gosec to CI
- 🔴 Add dependency scanning
- 🔴 Add SAST scanning
- ✅ Configure security policy

#### 0.3.4 Coverage Reporting
- ✅ Add coverage calculation
- ✅ Add codecov integration
- ✅ Add coverage badge to README
- ✅ Set coverage threshold (80%)

#### 0.3.5 Docker Builds
- 🔴 Create `.github/workflows/docker.yaml`
- 🔴 Add build step for each service
- 🔴 Add tag strategy (sha, branch, latest)
- 🔴 Add registry push
- 🔴 Add build caching

### 0.4 Documentation

#### 0.4.1 Architecture Doc
- 🔴 Create `docs/ARCHITECTURE.md`
- 🔴 Add system overview diagram
- 🔴 Add component diagram
- 🔴 Add data flow diagram
- 🔴 Add deployment diagram
- 🔴 Add sequence diagrams for key flows

#### 0.4.2 API Documentation
- 🔴 Create `docs/API.md`
- 🔴 Document opir-ingest endpoints
- 🔴 Document missile-warning endpoints
- 🔴 Document sensor-fusion endpoints
- 🔴 Document lvc-coordinator endpoints
- 🔴 Add OpenAPI/Swagger spec

#### 0.4.3 Deployment Documentation
- 🔴 Create `docs/DEPLOYMENT.md`
- 🔴 Add prerequisites
- 🔴 Add Docker Compose deployment
- 🔴 Add Kubernetes deployment
- 🔴 Add configuration reference
- 🔴 Add troubleshooting section

---

## Phase 1: Real Data Feeds (Weeks 2-3)

### 1.1 OPIR Adapter

#### 1.1.1 Interface Definition
- ✅ Create `pkg/sensors/opir/interface.go`
- ✅ Define OPIRDataFeed interface
- ✅ Define OPIRSighting struct
- ✅ Define OPIRConfig struct
- ✅ Define OPIRError types

#### 1.1.2 SBIRS-High Adapter
- ✅ Create `pkg/sensors/opir/sbirs.go`
- ✅ Implement connection establishment
- ✅ Implement TLS configuration
- ✅ Implement data stream reading
- ✅ Implement sighting parsing
- ✅ Implement error handling
- ✅ Implement reconnection logic
- ✅ Add unit tests

#### 1.1.3 NG-OPIR Adapter
- ✅ Create `pkg/sensors/opir/ngopir.go`
- ✅ Implement connection establishment
- ✅ Implement data stream reading
- ✅ Implement sighting parsing
- ✅ Implement error handling
- ✅ Add unit tests

#### 1.1.4 Configuration
- ✅ Create `pkg/sensors/opir/config.go`
- ✅ Add endpoint configuration
- ✅ Add authentication configuration
- ✅ Add timeout configuration
- ✅ Add retry configuration
- ✅ Add environment variable support

#### 1.1.5 Reconnection Logic
- ✅ Create `pkg/sensors/opir/reconnect.go`
- ✅ Implement exponential backoff
- ✅ Implement max retry limit
- ✅ Implement connection health check
- ✅ Add circuit breaker pattern

#### 1.1.6 Data Validation
- ✅ Create `pkg/sensors/opir/validate.go`
- ✅ Add latitude range validation
- ✅ Add longitude range validation
- ✅ Add intensity range validation
- ✅ Add timestamp validation
- ✅ Add duplicate detection

#### 1.1.7 Unit Tests
- ✅ Create `pkg/sensors/opir/opir_test.go`
- ✅ Add mock feed tests
- ✅ Add parsing tests
- ✅ Add validation tests
- ✅ Add reconnection tests

### 1.2 Radar Integration

#### 1.2.1 Interface Definition
- ✅ Create `pkg/sensors/radar/interface.go`
- ✅ Define RadarDataFeed interface
- ✅ Define RadarTrack struct
- ✅ Define RadarConfig struct

#### 1.2.2 AN/TPY-2 Adapter
- ✅ Create `pkg/sensors/radar/tpy2.go`
- ✅ Implement connection establishment
- ✅ Implement track parsing
- ✅ Implement coordinate conversion
- ✅ Implement error handling
- ✅ Add unit tests

#### 1.2.3 SBX Radar Adapter
- ✅ Create `pkg/sensors/radar/sbx.go`
- ✅ Implement connection establishment
- ✅ Implement track parsing
- ✅ Add unit tests

#### 1.2.4 UEWR Adapter
- ✅ Create `pkg/sensors/radar/uewr.go`
- ✅ Implement connection establishment
- ✅ Implement track parsing
- ✅ Add unit tests

#### 1.2.5 Track Correlation
- ✅ Create `pkg/sensors/radar/correlation.go`
- ✅ Implement track merging logic
- ✅ Implement track scoring
- ✅ Add unit tests

#### 1.2.6 Unit Tests
- ✅ Create `pkg/sensors/radar/radar_test.go`
- ✅ Create `pkg/sensors/radar/sbx_uewr_test.go`
- ✅ Create `pkg/sensors/radar/correlation_test.go`
- ✅ Add mock feed tests
- ✅ Add correlation tests

### 1.3 Sensor Fusion Enhancement

#### 1.3.1 Multi-Hypothesis Tracking
- ✅ Create `pkg/mht/`
- ✅ Implement hypothesis generation
- ✅ Implement hypothesis scoring
- ✅ Implement hypothesis pruning
- ✅ Implement track confirmation
- ✅ Add unit tests

#### 1.3.2 Track Scoring
- ✅ Create `pkg/fusion/scoring.go`
- ✅ Implement score calculation
- ✅ Implement score decay
- ✅ Implement confidence bounds
- ✅ Add unit tests

#### 1.3.3 Sensor Registration
- ✅ Create `pkg/geo/registration.go`
- ✅ Implement bias estimation
- ✅ Implement bias correction
- ✅ Add unit tests

#### 1.3.4 Time Alignment
- ✅ Create `pkg/geo/timealign.go`
- ✅ Implement interpolation
- ✅ Implement extrapolation
- ✅ Add unit tests

#### 1.3.5 Unit Tests
- ✅ Create `pkg/mht/mht_test.go`
- ✅ Add MHT tests
- ✅ Add scoring tests

### 1.4 Mode Switching

#### 1.4.1 Mode Implementation
- ✅ Create `pkg/mode/mode.go`
- ✅ Implement mode enum
- ✅ Implement mode switching
- ✅ Add configuration

#### 1.4.2 Replay Mode
- ✅ Create `pkg/mode/replay.go`
- ✅ Implement replay from file
- ✅ Implement time scaling
- ✅ Add unit tests

#### 1.4.3 Hybrid Mode
- ✅ Create `pkg/mode/hybrid.go`
- ✅ Implement real + simulated mixing
- ✅ Add unit tests

#### 1.4.4 Configuration
- ✅ Add mode configuration to each service
- ✅ Add environment variable support
- ✅ Add hot-reload support

---

## Phase 2: Protocol Bridges (Weeks 4-5)

### 2.1 HLA RTI Integration

#### 2.1.1 RTI Evaluation
- ✅ Evaluate Portico RTI
- ✅ Evaluate makRTI
- ✅ Create comparison matrix
- ✅ Document selection decision

#### 2.1.2 Portico Integration
- ✅ Add Portico dependency to go.mod
- ✅ Create RTI wrapper interface
- ✅ Add license handling
- ✅ Document configuration

#### 2.1.3 FOM Parsing
- ✅ Create `pkg/hla/fom/parser.go`
- ✅ Implement XML parsing
- ✅ Implement object class extraction
- ✅ Implement interaction class extraction
- ✅ Add unit tests

#### 2.1.4 Object Publishing
- 🔴 Create `pkg/hla/publish.go`
- 🔴 Implement object class publishing
- 🔴 Implement attribute updates
- 🔴 Implement ownership management
- 🔴 Add unit tests

#### 2.1.5 Object Subscription
- 🔴 Create `pkg/hla/subscribe.go`
- 🔴 Implement object class subscription
- 🔴 Implement attribute reflection
- 🔴 Implement discovery handling
- 🔴 Add unit tests

#### 2.1.6 Interaction Publishing
- 🔴 Create `pkg/hla/interaction.go`
- 🔴 Implement interaction publishing
- 🔴 Implement parameter handling
- 🔴 Add unit tests

#### 2.1.7 Interaction Subscription
- 🔴 Create `pkg/hla/interaction_sub.go`
- 🔴 Implement interaction subscription
- 🔴 Implement parameter extraction
- 🔴 Add unit tests

#### 2.1.8 Federation Management
- 🔴 Create `pkg/hla/federation.go`
- 🔴 Implement federation join
- 🔴 Implement federation resign
- 🔴 Implement synchronization points
- 🔴 Add unit tests

#### 2.1.9 HLA Tests
- 🔴 Create `pkg/hla/hla_test.go`
- 🔴 Add mock RTI tests
- 🔴 Add integration tests with Portico

### 2.2 DIS Gateway

#### 2.2.1 UDP Receiver
- ✅ Create `pkg/dis/receiver.go`
- ✅ Implement UDP socket binding
- ✅ Implement multicast support
- ✅ Implement buffer management
- ✅ Add unit tests

#### 2.2.2 UDP Transmitter
- ✅ Create `pkg/dis/transmitter.go`
- ✅ Implement UDP socket creation
- ✅ Implement multicast support
- ✅ Implement broadcast support
- ✅ Add unit tests

#### 2.2.3 Exercise Management
- ✅ Create `pkg/dis/exercise.go`
- ✅ Implement exercise ID handling
- ✅ Implement site/application ID management
- ✅ Implement entity ID allocation
- ✅ Add unit tests

#### 2.2.4 Entity State PDU
- ✅ Create `pkg/dis/pdu/entity_state.go` (already in dis-pdu)
- ✅ Implement full PDU encoding
- ✅ Implement full PDU decoding
- ✅ Implement all dead reckoning algorithms
- ✅ Add unit tests

#### 2.2.5 Fire PDU
- ✅ Create `pkg/dis/pdu/fire.go` (already in dis-pdu)
- ✅ Implement Fire PDU encoding
- ✅ Implement Fire PDU decoding
- ✅ Add unit tests

#### 2.2.6 Detonation PDU
- ✅ Create `pkg/dis/pdu/detonation.go` (already in dis-pdu)
- ✅ Implement Detonation PDU encoding
- ✅ Implement Detonation PDU decoding
- ✅ Add unit tests

#### 2.2.7 Electromagnetic Emission PDU
- 🔴 Create `pkg/dis/pdu/emission.go`
- 🔴 Implement Emission PDU encoding
- 🔴 Implement Emission PDU decoding
- 🔴 Add unit tests

#### 2.2.8 DIS Tests
- 🔴 Create `pkg/dis/dis_test.go`
- 🔴 Add PDU roundtrip tests
- 🔴 Add network tests

### 2.3 JREAP Bridge

#### 2.3.1 JREAP-A Adapter
- ✅ Create `pkg/jreap/jreap_a.go` (in jreap.go)
- ✅ Implement serial connection
- ✅ Implement message framing
- ✅ Implement error handling
- ✅ Add unit tests

#### 2.3.2 JREAP-B Adapter
- ✅ Create `pkg/jreap/jreap_b.go` (in jreap.go)
- ✅ Implement TCP/IP connection
- ✅ Implement message framing
- ✅ Implement error handling
- ✅ Add unit tests

#### 2.3.3 JREAP-C Adapter
- ✅ Create `pkg/jreap/jreap_c.go` (in jreap.go)
- ✅ Implement satellite link handling
- ✅ Implement message framing
- ✅ Implement error handling
- ✅ Add unit tests

#### 2.3.4 Message Parsing
- ✅ Create `pkg/jreap/message.go` (in jreap.go)
- ✅ Implement MIL-STD-3011 message parsing
- ✅ Implement message generation
- ✅ Add unit tests

#### 2.3.5 Message Generation
- ✅ Create `pkg/jreap/generate.go` (in jreap.go)
- ✅ Implement JREAP message encoding
- ✅ Implement checksum calculation
- ✅ Add unit tests

#### 2.3.6 JREAP Tests
- ✅ Create `pkg/jreap/jreap_test.go`
- ✅ Add roundtrip tests
- ✅ Add MIL-STD-3011 compliance tests

### 2.4 Link 16 Bridge

#### 2.4.1 J-Series Parsing
- ✅ Create `pkg/link16/jseries/parser.go`
- ✅ Implement J-series header parsing
- ✅ Implement J-series body parsing
- ✅ Add unit tests

#### 2.4.2 J-Series Generation
- ✅ Create `pkg/link16/jseries/generate.go` (in parser.go)
- ✅ Implement J-series encoding
- ✅ Add unit tests

#### 2.4.3 J3.2 Support
- ✅ Create `pkg/link16/j32.go`
- ✅ Implement J3.2 (Air Track) encoding
- ✅ Implement J3.2 decoding
- ✅ Add unit tests

#### 2.4.4 J7.0 Support
- ✅ Create `pkg/link16/j70.go`
- ✅ Implement J7.0 (Track Management) encoding
- ✅ Implement J7.0 decoding
- ✅ Add unit tests

#### 2.4.5 J12.0 Support
- ✅ Create `pkg/link16/j120.go`
- ✅ Implement J12.0 (Mission Assignment) encoding
- ✅ Implement J12.0 decoding
- ✅ Add unit tests

#### 2.4.6 Link 16 Tests
- ✅ Create `pkg/link16/link16_test.go` (in jseries/parser_test.go, j32_test.go, j70_test.go, j120_test.go)
- ✅ Add roundtrip tests
- 🔴 Add MIL-STD-6016 compliance tests

---

## Phase 3: C2 Integration (Weeks 6-7)

### 3.1 C2BMC Interface

#### 3.1.1 Interface Definition
- 🔴 Create `pkg/c2/c2bmc/interface.go`
- 🔴 Define C2BMCClient interface
- 🔴 Define AlertRequest struct
- 🔴 Define TrackData struct

#### 3.1.2 REST Client
- 🔴 Create `pkg/c2/c2bmc/client.go`
- 🔴 Implement HTTP client
- 🔴 Implement request serialization
- 🔴 Implement response parsing
- 🔴 Implement error handling

#### 3.1.3 Alert Submission
- 🔴 Create `pkg/c2/c2bmc/alert.go`
- 🔴 Implement alert formatting
- 🔴 Implement alert submission
- 🔴 Implement acknowledgment handling

#### 3.1.4 Track Correlation
- 🔴 Create `pkg/c2/c2bmc/track.go`
- 🔴 Implement track submission
- 🔴 Implement track correlation
- 🔴 Implement status query

#### 3.1.5 PKI Authentication
- 🔴 Create `pkg/c2/c2bmc/auth.go`
- 🔴 Implement certificate loading
- 🔴 Implement TLS configuration
- 🔴 Implement mutual TLS

#### 3.1.6 C2BMC Tests
- 🔴 Create `pkg/c2/c2bmc/c2bmc_test.go`
- 🔴 Add mock server tests
- 🔴 Add integration tests

### 3.2 Alert Dissemination

#### 3.2.1 NCA Formatting
- 🔴 Create `apps/alert-dissemination/nca/format.go`
- 🔴 Implement CONOPREP format
- 🔴 Implement IMMINENT format
- 🔴 Implement INCOMING format
- 🔴 Add unit tests

#### 3.2.2 Alert Queue
- 🔴 Create `apps/alert-dissemination/queue/queue.go`
- 🔴 Implement priority queue
- 🔴 Implement FIFO for same priority
- 🔴 Add unit tests

#### 3.2.3 Acknowledgment Handling
- 🔴 Create `apps/alert-dissemination/ack/ack.go`
- 🔴 Implement ACK handling
- 🔴 Implement NACK handling
- 🔴 Implement timeout handling
- 🔴 Add unit tests

#### 3.2.4 Escalation Logic
- 🔴 Create `apps/alert-dissemination/escalation.go`
- 🔴 Implement escalation rules
- 🔴 Implement de-escalation rules
- 🔴 Add unit tests

#### 3.2.5 Delivery Confirmation
- 🔴 Create `apps/alert-dissemination/delivery.go`
- 🔴 Implement delivery tracking
- 🔴 Implement retry logic
- 🔴 Add unit tests

#### 3.2.6 Multiple Recipients
- 🔴 Create `apps/alert-dissemination/recipients.go`
- 🔴 Implement recipient list management
- 🔴 Implement delivery status tracking
- 🔴 Add unit tests

#### 3.2.7 Alert Tests
- 🔴 Create `apps/alert-dissemination/alert_test.go`
- 🔴 Add end-to-end tests

### 3.3 Tactical Data Links

#### 3.3.1 TADIL-A
- 🔴 Create `pkg/tadil/tadil_a.go`
- 🔴 Implement message formatting
- 🔴 Implement message parsing
- 🔴 Add unit tests

#### 3.3.2 TADIL-J
- 🔴 Create `pkg/tadil/tadil_j.go`
- 🔴 Implement message formatting
- 🔴 Implement message parsing
- 🔴 Add unit tests

#### 3.3.3 VMF Support
- 🔴 Create `pkg/tadil/vmf.go`
- 🔴 Implement VMF message formatting
- 🔴 Implement VMF message parsing
- 🔴 Add unit tests

#### 3.3.4 TADIL Tests
- 🔴 Create `pkg/tadil/tadil_test.go`
- 🔴 Add roundtrip tests

### 3.4 External System Interfaces

#### 3.4.1 JTAGS Interface
- 🔴 Create `pkg/external/jtags.go`
- 🔴 Implement JTAGS message formatting
- 🔴 Implement JTAGS connection handling
- 🔴 Add unit tests

#### 3.4.2 USMTF Generation
- 🔴 Create `pkg/external/usmtf.go`
- 🔴 Implement USMTF message formatting
- 🔴 Add unit tests

#### 3.4.3 ADatP-3 Generation
- 🔴 Create `pkg/external/adatp3.go`
- 🔴 Implement ADatP-3 message formatting
- 🔴 Add unit tests

#### 3.4.4 External Tests
- 🔴 Create `pkg/external/external_test.go`
- 🔴 Add format compliance tests

---

## Phase 4: Persistence & HA (Week 8)

### 4.1 PostgreSQL Schema

#### 4.1.1 Schema Design
- 🔴 Create `db/schema/design.sql`
- 🔴 Design tracks table
- 🔴 Design alerts table
- 🔴 Design events table
- 🔴 Design entities table

#### 4.1.2 Migrations
- 🔴 Create `db/migrations/001_tracks.up.sql`
- 🔴 Create `db/migrations/001_tracks.down.sql`
- 🔴 Create `db/migrations/002_alerts.up.sql`
- 🔴 Create `db/migrations/002_alerts.down.sql`
- 🔴 Create `db/migrations/003_events.up.sql`
- 🔴 Create `db/migrations/003_events.down.sql`
- 🔴 Create `db/migrations/004_entities.up.sql`
- 🔴 Create `db/migrations/004_entities.down.sql`

#### 4.1.3 Indexes
- 🔴 Create `db/migrations/005_indexes.up.sql`
- 🔴 Add track_id index
- 🔴 Add timestamp index
- 🔴 Add spatial index (PostGIS)
- 🔴 Add composite indexes

#### 4.1.4 Repository Layer
- 🔴 Create `pkg/db/repository/interface.go`
- 🔴 Create `pkg/db/repository/tracks.go`
- 🔴 Create `pkg/db/repository/alerts.go`
- 🔴 Create `pkg/db/repository/events.go`
- 🔴 Create `pkg/db/repository/entities.go`

### 4.2 Time-Series Storage

#### 4.2.1 TimescaleDB Integration
- 🔴 Create `db/timescaledb/setup.sql`
- 🔴 Create hypertable for track_history
- 🔴 Create continuous aggregates
- 🔴 Test time-series queries

#### 4.2.2 Continuous Aggregates
- 🔴 Create `db/timescaledb/aggregates.sql`
- 🔴 Add minute aggregate
- 🔴 Add hour aggregate
- 🔴 Add day aggregate

#### 4.2.3 Retention Policies
- 🔴 Create `db/timescaledb/retention.sql`
- 🔴 Add raw data retention (30 days)
- 🔴 Add aggregate retention (1 year)

### 4.3 Redis Enhancement

#### 4.3.1 Track State Caching
- 🔴 Create `pkg/cache/tracks.go`
- 🔴 Implement track state caching
- 🔴 Implement cache invalidation
- 🔴 Add unit tests

#### 4.3.2 Session Management
- 🔴 Create `pkg/cache/session.go`
- 🔴 Implement session storage
- 🔴 Implement session expiration
- 🔴 Add unit tests

#### 4.3.3 Pub/Sub Coordination
- 🔴 Create `pkg/cache/pubsub.go`
- 🔴 Implement pub/sub for coordination
- 🔴 Implement leader election
- 🔴 Add unit tests

### 4.4 High Availability

#### 4.4.1 Kafka Consumer Groups
- 🔴 Update Kafka consumer configuration
- 🔴 Add consumer group balancing
- 🔴 Add rebalance handling
- 🔴 Test with multiple consumers

#### 4.4.2 Leader Election
- 🔴 Create `pkg/ha/election.go`
- 🔴 Implement etcd-based election
- 🔴 Implement Redis-based election
- 🔴 Add unit tests

#### 4.4.3 Health Endpoints
- 🔴 Create `pkg/ha/health.go`
- 🔴 Implement liveness probe
- 🔴 Implement readiness probe
- 🔴 Add startup probe

#### 4.4.4 Graceful Shutdown
- 🔴 Create `pkg/ha/shutdown.go`
- 🔴 Implement signal handling
- 🔴 Implement connection draining
- 🔴 Test graceful shutdown

#### 4.4.5 Kubernetes Manifests
- 🔴 Create `k8s/vigil/namespace.yaml`
- 🔴 Create `k8s/vigil/opir-ingest.yaml`
- 🔴 Create `k8s/vigil/missile-warning.yaml`
- 🔴 Create `k8s/vigil/sensor-fusion.yaml`
- 🔴 Create `k8s/vigil/lvc-coordinator.yaml`

#### 4.4.6 Horizontal Autoscaling
- 🔴 Create `k8s/vigil/hpa.yaml`
- 🔴 Add CPU-based scaling
- 🔴 Add custom metrics scaling
- 🔴 Test autoscaling

---

## Phase 5: Security & Certification (Week 9)

### 5.1 Authentication

#### 5.1.1 mTLS
- 🔴 Create `pkg/auth/mtls.go`
- 🔴 Implement certificate loading
- 🔴 Implement certificate verification
- 🔴 Implement mTLS server
- 🔴 Implement mTLS client
- 🔴 Add unit tests

#### 5.1.2 PKI Management
- 🔴 Create `pkg/auth/pki.go`
- 🔴 Implement certificate generation
- 🔴 Implement certificate rotation
- 🔴 Implement CRL checking
- 🔴 Add unit tests

#### 5.1.3 JWT Validation
- 🔴 Create `pkg/auth/jwt.go`
- 🔴 Implement JWT parsing
- 🔴 Implement JWT validation
- 🔴 Implement token refresh
- 🔴 Add unit tests

#### 5.1.4 API Key Management
- 🔴 Create `pkg/auth/apikey.go`
- 🔴 Implement API key generation
- 🔴 Implement API key validation
- 🔴 Implement key rotation
- 🔴 Add unit tests

### 5.2 Authorization

#### 5.2.1 RBAC Middleware
- 🔴 Create `pkg/auth/rbac.go`
- 🔴 Implement role checking
- 🔴 Implement permission checking
- 🔴 Implement role inheritance
- 🔴 Add unit tests

#### 5.2.2 Role Definitions
- 🔴 Create `pkg/auth/roles.go`
- 🔴 Define admin role
- 🔴 Define operator role
- 🔴 Define viewer role
- 🔴 Add role tests

#### 5.2.3 Audit Logging
- 🔴 Create `pkg/auth/audit.go`
- 🔴 Implement request logging
- 🔴 Implement access logging
- 🔴 Implement change logging
- 🔴 Add unit tests

### 5.3 Network Security

#### 5.3.1 Network Policies
- 🔴 Create `k8s/vigil/networkpolicy.yaml`
- 🔴 Implement ingress policies
- 🔴 Implement egress policies
- 🔴 Test policies

#### 5.3.2 Service Mesh
- 🔴 Create `k8s/vigil/istio.yaml`
- 🔴 Add Istio sidecar
- 🔴 Configure mTLS
- 🔴 Configure traffic policies

#### 5.3.3 Secrets Management
- 🔴 Create `k8s/vigil/vault.yaml`
- 🔴 Add Vault integration
- 🔴 Implement secret injection
- 🔴 Test secrets

### 5.4 Compliance

#### 5.4.1 STIG Checklist
- 🔴 Create `docs/STIG.md`
- 🔴 Document STIG requirements
- 🔴 Create validation scripts
- 🔴 Test compliance

#### 5.4.2 Security Scanning
- 🔴 Add Trivy to CI
- 🔴 Add Snyk to CI
- 🔴 Add dependency scanning
- 🔴 Fix vulnerabilities

#### 5.4.3 Vulnerability Reporting
- 🔴 Create `docs/SECURITY.md`
- 🔴 Document reporting process
- 🔴 Document response SLAs
- 🔴 Document disclosure policy

#### 5.4.4 ATO Package
- 🔴 Create `docs/ATO/`
- 🔴 Add system description
- 🔴 Add network diagram
- 🔴 Add data flow diagram
- 🔴 Add risk assessment
- 🔴 Add contingency plan

---

## Phase 6: Integration Testing (Week 10)

### 6.1 End-to-End Tests

#### 6.1.1 Sensor-to-Alert E2E
- 🔴 Create `tests/e2e/sensor_alert_test.go`
- 🔴 Add OPIR ingest test
- 🔴 Add missile warning test
- 🔴 Add alert generation test
- 🔴 Verify end-to-end flow

#### 6.1.2 Track Lifecycle E2E
- 🔴 Create `tests/e2e/track_lifecycle_test.go`
- 🔴 Add track creation test
- 🔴 Add track update test
- 🔴 Add track fusion test
- 🔴 Add track deletion test

#### 6.1.3 Federation E2E
- 🔴 Create `tests/e2e/federation_test.go`
- 🔴 Add HLA federation test
- 🔴 Add DIS gateway test
- 🔴 Add entity state test

#### 6.1.4 C2 E2E
- 🔴 Create `tests/e2e/c2_test.go`
- 🔴 Add alert delivery test
- 🔴 Add track correlation test
- 🔴 Add acknowledgment test

### 6.2 Performance Testing

#### 6.2.1 OPIR Load Test
- 🔴 Create `tests/load/opir_load_test.go`
- 🔴 Add 1000 msg/s test
- 🔴 Add 5000 msg/s test
- 🔴 Add 10000 msg/s test
- 🔴 Measure latency

#### 6.2.2 Track Correlation Load
- 🔴 Create `tests/load/correlation_load_test.go`
- 🔴 Add 1000 tracks test
- 🔴 Add 10000 tracks test
- 🔴 Add 100000 tracks test
- 🔴 Measure correlation time

#### 6.2.3 Latency Benchmarks
- 🔴 Create `tests/benchmarks/latency_test.go`
- 🔴 Measure P99 latency
- 🔴 Measure P95 latency
- 🔴 Measure P50 latency
- 🔴 Document results

#### 6.2.4 Latency Tuning
- 🔴 Profile hot paths
- 🔴 Optimize allocations
- 🔴 Optimize GC pressure
- 🔴 Add caching where beneficial
- 🔴 Retest latency

### 6.3 Chaos Engineering

#### 6.3.1 Network Partition Tests
- 🔴 Create `tests/chaos/network_test.go`
- 🔴 Add Kafka partition test
- 🔴 Add Redis partition test
- 🔴 Verify recovery

#### 6.3.2 Node Failure Tests
- 🔴 Create `tests/chaos/node_test.go`
- 🔴 Add pod kill test
- 🔴 Add node drain test
- 🔴 Verify failover

#### 6.3.3 Kafka Failure Tests
- 🔴 Create `tests/chaos/kafka_test.go`
- 🔴 Add broker kill test
- 🔴 Add topic deletion test
- 🔴 Verify recovery

### 6.4 Documentation Finalization

#### 6.4.1 API Documentation
- 🔴 Update API.md
- 🔴 Add all endpoints
- 🔴 Add all request/response schemas
- 🔴 Add all error codes

#### 6.4.2 Operator Runbook
- 🔴 Create `docs/RUNBOOK.md`
- 🔴 Add deployment procedures
- 🔴 Add monitoring procedures
- 🔴 Add incident procedures

#### 6.4.3 Troubleshooting Guide
- 🔴 Create `docs/TROUBLESHOOTING.md`
- 🔴 Add common issues
- 🔴 Add diagnostic commands
- 🔴 Add resolution steps

#### 6.4.4 Deployment Checklist
- 🔴 Create `docs/CHECKLIST.md`
- 🔴 Add pre-deployment checklist
- 🔴 Add deployment checklist
- 🔴 Add post-deployment checklist

---

## Summary Statistics

| Phase | Tasks | Subtasks |
|-------|-------|----------|
| Phase 0 | 4 | 32 |
| Phase 1 | 4 | 28 |
| Phase 2 | 4 | 42 |
| Phase 3 | 4 | 26 |
| Phase 4 | 4 | 24 |
| Phase 5 | 4 | 17 |
| Phase 6 | 4 | 28 |
| **Total** | **28** | **197** |

---

## Quick Reference

### Priority Legend
- **P0**: Blocking, must complete first
- **P1**: High priority, core functionality
- **P2**: Medium priority, important features
- **P3**: Low priority, nice to have

### Effort Estimates
- **Hours**: Estimated effort per subtask
- **Days**: Estimated calendar time
- **Dependencies**: Required predecessors

### Progress Tracking
```bash
# Count completed tasks
grep -c "✅" TODO.md

# Count remaining tasks
grep -c "🔴" TODO.md

# Count in-progress
grep -c "🟡" TODO.md
```