# VIGIL Usage Guide

## Quick Start

### Starting Services

```bash
# Start all services (Docker Compose)
docker compose up -d

# Start specific service
docker compose up -d opir-ingest

# View logs
docker compose logs -f opir-ingest
```

### Verify Services

```bash
# Check health
curl http://localhost:8080/health

# Expected response
{
  "status": "healthy",
  "uptime": 3600,
  "checks": {
    "kafka": "ok",
    "redis": "ok"
  }
}
```

## OPIR Ingest Service

### Submit Detection

```bash
curl -X POST http://localhost:8080/api/v1/detections \
  -H "Content-Type: application/json" \
  -d '{
    "sensor_id": "SBIRS-GEO-1",
    "timestamp": 1700000000000,
    "latitude": 38.8977,
    "longitude": -77.0365,
    "altitude": 50000.0,
    "velocity": 3000.0,
    "heading": 45.0,
    "confidence": 0.95,
    "signature": "IR-BOOSTER-001"
  }'
```

### Response

```json
{
  "status": "accepted",
  "detection_id": "DET-001",
  "timestamp": 1700000000000
}
```

### Check Status

```bash
curl http://localhost:8080/api/v1/status

# Response
{
  "detections_received": 12345,
  "detections_published": 12340,
  "errors": 5,
  "last_detection": 1700000000000,
  "processing_rate": 100.5
}
```

## Sensor Fusion Service

### Submit Tracks for Correlation

```bash
curl -X POST http://localhost:8081/api/v1/tracks/correlate \
  -H "Content-Type: application/json" \
  -d '{
    "tracks": [
      {
        "source_id": "SBIRS-GEO-1",
        "latitude": 38.8977,
        "longitude": -77.0365,
        "altitude": 50000.0,
        "velocity_x": 300.0,
        "velocity_y": 0.0,
        "velocity_z": 0.0,
        "confidence": 0.95
      },
      {
        "source_id": "RADAR-1",
        "latitude": 38.8980,
        "longitude": -77.0370,
        "altitude": 50050.0,
        "velocity_x": 295.0,
        "velocity_y": 5.0,
        "velocity_z": 0.0,
        "confidence": 0.92
      }
    ]
  }'
```

### Response

```json
{
  "correlated_track": {
    "track_id": "TRACK-001",
    "track_number": 1001,
    "latitude": 38.8978,
    "longitude": -77.0367,
    "altitude": 50025.0,
    "velocity_x": 297.5,
    "velocity_y": 2.5,
    "velocity_z": 0.0,
    "confidence": 0.94,
    "source_count": 2
  }
}
```

### Get All Tracks

```bash
curl http://localhost:8081/api/v1/tracks

# Query parameters
curl "http://localhost:8081/api/v1/tracks?limit=10&offset=0&source=SBIRS"
```

### Get Specific Track

```bash
curl http://localhost:8081/api/v1/tracks/TRACK-001
```

## Missile Warning Engine

### Get Active Alerts

```bash
curl http://localhost:8082/api/v1/alerts

# Filter by level
curl "http://localhost:8082/api/v1/alerts?level=INCOMING"

# Filter by threat type
curl "http://localhost:8082/api/v1/alerts?threat_type=BALLISTIC"
```

### Evaluate Track for Alert

```bash
curl -X POST http://localhost:8082/api/v1/alerts/evaluate \
  -H "Content-Type: application/json" \
  -d '{
    "track_id": "TRACK-001",
    "threat_type": "BALLISTIC"
  }'
```

### Response

```json
{
  "alert_level": "INCOMING",
  "threat_type": "BALLISTIC",
  "time_to_impact": 90.0,
  "confidence": 0.85,
  "should_alert": true
}
```

### Get Threat Summary

```bash
curl http://localhost:8082/api/v1/threats

# Response
{
  "total_threats": 5,
  "by_type": {
    "BALLISTIC": 2,
    "CRUISE": 1,
    "AIRCRAFT": 1,
    "UAV": 1
  },
  "by_level": {
    "CONOPREP": 1,
    "IMMINENT": 2,
    "INCOMING": 1,
    "HOSTILE": 1
  }
}
```

## LVC Coordinator

### Create Entity

```bash
curl -X POST http://localhost:8084/api/v1/entities \
  -H "Content-Type: application/json" \
  -d '{
    "force_id": 1,
    "entity_type": "F-16C",
    "marking": "VIPER01",
    "location": {
      "lat": 38.8977,
      "lon": -77.0365,
      "alt": 10000.0
    },
    "orientation": {
      "psi": 0.785,
      "theta": 0.0,
      "phi": 0.0
    },
    "velocity": {
      "x": 300.0,
      "y": 0.0,
      "z": 0.0
    },
    "dead_reckoning_model": 2
  }'
```

### Response

```json
{
  "entity_id": {
    "site_id": 1,
    "application_id": 1,
    "entity_id": 100
  },
  "force_id": 1,
  "marking": "VIPER01",
  "location": {
    "x": 1000000.0,
    "y": 2000000.0,
    "z": 10000.0
  },
  "created_at": 1700000000000
}
```

### Get Entities

```bash
# All entities
curl http://localhost:8084/api/v1/entities

# Filter by force
curl "http://localhost:8084/api/v1/entities?force_id=1"

# Filter by marking
curl "http://localhost:8084/api/v1/entities?marking=VIPER"
```

### Remove Entity

```bash
curl -X DELETE http://localhost:8084/api/v1/entities/100
```

## Replay Engine

### Start Recording

```bash
curl -X POST http://localhost:8085/api/v1/recordings \
  -H "Content-Type: application/json" \
  -d '{
    "name": "EXERCISE-2024-001",
    "description": "Training exercise recording",
    "tags": ["training", "exercise"]
  }'
```

### Response

```json
{
  "recording_id": "REC-001",
  "name": "EXERCISE-2024-001",
  "status": "recording",
  "started_at": 1700000000000
}
```

### List Recordings

```bash
curl http://localhost:8085/api/v1/recordings

# Response
{
  "recordings": [
    {
      "id": "REC-001",
      "name": "EXERCISE-2024-001",
      "start_time": 1700000000000,
      "end_time": 1700003600000,
      "pdu_count": 12345,
      "file_size": 10485760
    }
  ],
  "total": 1
}
```

### Start Playback

```bash
curl -X POST http://localhost:8085/api/v1/recordings/REC-001/playback \
  -H "Content-Type: application/json" \
  -d '{
    "speed": 2.0,
    "start_time": 1700000000000
  }'
```

### Stop Playback

```bash
curl -X DELETE http://localhost:8085/api/v1/playback/PLAY-001
```

## Command Line Interface

### vigilctl

VIGIL includes a CLI tool for common operations:

```bash
# Install
go install github.com/wezzels/vigil/cmd/vigilctl@latest

# Check service health
vigilctl health --all

# Get tracks
vigilctl tracks list --limit 10

# Get alerts
vigilctl alerts list --level INCOMING

# Create entity
vigilctl entities create --force-id 1 --type F-16C --marking VIPER01

# Start recording
vigilctl recordings start --name "Test Recording"

# Configure
vigilctl config set kafka.brokers localhost:9092
```

## Example Workflows

### Track Processing Pipeline

```bash
# 1. Submit detection to OPIR ingest
curl -X POST http://localhost:8080/api/v1/detections \
  -H "Content-Type: application/json" \
  -d '{"sensor_id": "SBIRS-GEO-1", "latitude": 38.8977, "longitude": -77.0365, "altitude": 50000, "confidence": 0.95}'

# 2. Check sensor fusion for correlated track
curl http://localhost:8081/api/v1/tracks?source=SBIRS

# 3. Evaluate for alert
curl -X POST http://localhost:8082/api/v1/alerts/evaluate \
  -H "Content-Type: application/json" \
  -d '{"track_id": "TRACK-001", "threat_type": "BALLISTIC"}'

# 4. Get alert status
curl http://localhost:8082/api/v1/alerts?track_id=TRACK-001
```

### DIS Entity Lifecycle

```bash
# 1. Create entity
ENTITY_ID=$(curl -s -X POST http://localhost:8084/api/v1/entities \
  -H "Content-Type: application/json" \
  -d '{"force_id": 1, "entity_type": "F-16C", "marking": "VIPER01"}' | jq -r '.entity_id.entity_id')

# 2. Update entity (via Entity State PDUs - automatic)

# 3. Monitor entity
curl http://localhost:8084/api/v1/entities/$ENTITY_ID

# 4. Remove entity
curl -X DELETE http://localhost:8084/api/v1/entities/$ENTITY_ID
```

### Mission Replay

```bash
# 1. Start recording
curl -X POST http://localhost:8085/api/v1/recordings \
  -H "Content-Type: application/json" \
  -d '{"name": "Mission-Alpha"}'

# 2. ... exercise operations ...

# 3. Stop recording
curl -X DELETE http://localhost:8085/api/v1/recordings/REC-001

# 4. List recordings
curl http://localhost:8085/api/v1/recordings

# 5. Playback recording
curl -X POST http://localhost:8085/api/v1/recordings/REC-001/playback \
  -H "Content-Type: application/json" \
  -d '{"speed": 2.0}'
```

## Error Handling

### Common Errors

#### 400 Bad Request

```json
{
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "Invalid latitude value",
    "field": "latitude",
    "value": -100.0
  }
}
```

#### 404 Not Found

```json
{
  "error": {
    "code": "NOT_FOUND",
    "message": "Track not found",
    "track_id": "TRACK-999"
  }
}
```

#### 503 Service Unavailable

```json
{
  "error": {
    "code": "SERVICE_UNAVAILABLE",
    "message": "Kafka connection failed",
    "retry_after": 30
  }
}
```

### Rate Limiting

```bash
# Response headers
X-RateLimit-Limit: 1000
X-RateLimit-Remaining: 999
X-RateLimit-Reset: 1700000060

# When rate limited (429 Too Many Requests)
{
  "error": {
    "code": "RATE_LIMITED",
    "message": "Rate limit exceeded",
    "retry_after": 60
  }
}
```

## Monitoring

### Prometheus Metrics

```bash
# Get all metrics
curl http://localhost:8080/metrics

# Key metrics
vigil_tracks_processed_total{service="sensor-fusion"} 12345
vigil_processing_latency_ms{service="sensor-fusion"} 15.5
vigil_alerts_generated_total{service="missile-warning"} 42
vigil_kafka_lag{service="opir-ingest"} 0
vigil_errors_total{service="sensor-fusion"} 5
```

### Grafana Dashboards

Import dashboards from `docs/dashboards/`:

```bash
# Import dashboard
curl -X POST http://grafana:3000/api/dashboards/db \
  -H "Content-Type: application/json" \
  -d @docs/dashboards/vigil-overview.json
```

### Log Analysis

```bash
# Tail logs
docker compose logs -f opir-ingest

# Search logs
docker compose logs opir-ingest | grep ERROR

# JSON logs (if LOG_FORMAT=json)
docker compose logs opir-ingest | jq 'select(.level=="error")'
```

## Performance Tuning

### Kafka Optimization

```yaml
# config/kafka.yaml
producer:
  batch_size: 32768      # Increase batch size
  linger_ms: 10          # Small delay for batching
  compression: snappy    # Enable compression
  acks: all              # Wait for all replicas

consumer:
  fetch_min_bytes: 1024  # Minimum bytes per fetch
  fetch_max_wait_ms: 500 # Maximum wait time
```

### Redis Optimization

```yaml
# config/redis.yaml
pool_size: 200          # Connection pool size
min_idle_conns: 20      # Minimum idle connections
max_retries: 5          # Retry attempts
```

### Processing Optimization

```yaml
# config/processing.yaml
max_tracks: 50000       # Maximum tracks in memory
batch_size: 500         # Processing batch size
workers: 16             # Worker goroutines
processing_timeout: 50ms # Processing timeout
```

## Security

### Authentication

```bash
# Get JWT token
curl -X POST http://localhost:8080/api/v1/auth/token \
  -H "Content-Type: application/json" \
  -d '{"client_id": "your-id", "client_secret": "your-secret"}'

# Use token in requests
curl http://localhost:8081/api/v1/tracks \
  -H "Authorization: Bearer YOUR_TOKEN"
```

### TLS

```bash
# Enable TLS
curl --cacert /path/to/ca.crt \
  --cert /path/to/client.crt \
  --key /path/to/client.key \
  https://vigil.example.com/api/v1/tracks
```