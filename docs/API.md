# VIGIL API Documentation

## Overview

VIGIL provides RESTful APIs for all microservices. Each service exposes health, metrics, and service-specific endpoints.

## Common Endpoints

All services implement these common endpoints:

### Health Check

```http
GET /health
```

Returns service health status.

**Response:**
```json
{
  "status": "healthy",
  "uptime": 3600,
  "checks": {
    "kafka": "ok",
    "redis": "ok",
    "postgres": "ok"
  },
  "version": "0.0.1"
}
```

### Metrics

```http
GET /metrics
```

Returns Prometheus-formatted metrics.

**Response:**
```
# HELP vigil_tracks_processed Total tracks processed
# TYPE vigil_tracks_processed counter
vigil_tracks_processed{service="sensor-fusion"} 12345

# HELP vigil_processing_latency Processing latency in milliseconds
# TYPE vigil_processing_latency histogram
vigil_processing_latency_bucket{le="10"} 100
vigil_processing_latency_bucket{le="50"} 450
vigil_processing_latency_bucket{le="100"} 800
```

## OPIR Ingest API

### POST /api/v1/detections

Submit an OPIR detection.

**Request:**
```json
{
  "sensor_id": "SBIRS-GEO-1",
  "timestamp": 1700000000000,
  "latitude": 38.8977,
  "longitude": -77.0365,
  "altitude": 50000.0,
  "velocity": 3000.0,
  "heading": 45.0,
  "confidence": 0.95,
  "signature": "IR-BOOSTER-001"
}
```

**Response:**
```json
{
  "status": "accepted",
  "detection_id": "DET-001",
  "timestamp": 1700000000000
}
```

### GET /api/v1/status

Get ingestion status.

**Response:**
```json
{
  "detections_received": 12345,
  "detections_published": 12340,
  "errors": 5,
  "last_detection": 1700000000000,
  "processing_rate": 100.5
}
```

## Sensor Fusion API

### POST /api/v1/tracks/correlate

Submit tracks for correlation.

**Request:**
```json
{
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
}
```

**Response:**
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
    "source_count": 2,
    "variance": 0.001
  }
}
```

### GET /api/v1/tracks

Get all active tracks.

**Query Parameters:**
- `limit`: Maximum tracks to return (default: 100)
- `offset`: Pagination offset
- `source`: Filter by source

**Response:**
```json
{
  "tracks": [
    {
      "track_id": "TRACK-001",
      "track_number": 1001,
      "latitude": 38.8978,
      "longitude": -77.0367,
      "altitude": 50025.0,
      "heading": 45.0,
      "speed": 300.0,
      "confidence": 0.94,
      "source_count": 2,
      "last_update": 1700000000000
    }
  ],
  "total": 1,
  "limit": 100,
  "offset": 0
}
```

### GET /api/v1/tracks/{track_id}

Get specific track by ID.

**Response:**
```json
{
  "track_id": "TRACK-001",
  "track_number": 1001,
  "latitude": 38.8978,
  "longitude": -77.0367,
  "altitude": 50025.0,
  "velocity": {
    "x": 297.5,
    "y": 2.5,
    "z": 0.0
  },
  "heading": 45.0,
  "speed": 300.0,
  "confidence": 0.94,
  "source_count": 2,
  "sources": ["SBIRS-GEO-1", "RADAR-1"],
  "variance": {
    "lat": 0.001,
    "lon": 0.001,
    "alt": 10.0
  },
  "last_update": 1700000000000,
  "created_at": 1699999000000
}
```

## Missile Warning Engine API

### GET /api/v1/alerts

Get active alerts.

**Query Parameters:**
- `level`: Filter by alert level (CONOPREP, IMMINENT, INCOMING, HOSTILE)
- `threat_type`: Filter by threat type
- `limit`: Maximum alerts to return

**Response:**
```json
{
  "alerts": [
    {
      "alert_id": "ALERT-001",
      "track_id": "TRACK-001",
      "alert_level": "INCOMING",
      "threat_type": "BALLISTIC",
      "launch_point": {
        "lat": 38.0,
        "lon": -77.0
      },
      "impact_point": {
        "lat": 39.0,
        "lon": -78.0
      },
      "time_to_impact": 90.0,
      "confidence": 0.85,
      "created_at": 1700000000000
    }
  ],
  "total": 1
}
```

### POST /api/v1/alerts/evaluate

Evaluate track for alert generation.

**Request:**
```json
{
  "track_id": "TRACK-001",
  "threat_type": "BALLISTIC"
}
```

**Response:**
```json
{
  "alert_level": "INCOMING",
  "threat_type": "BALLISTIC",
  "time_to_impact": 90.0,
  "confidence": 0.85,
  "should_alert": true
}
```

### GET /api/v1/threats

Get threat summary.

**Response:**
```json
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

## LVC Coordinator API

### POST /api/v1/entities

Create a new DIS entity.

**Request:**
```json
{
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
}
```

**Response:**
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

### GET /api/v1/entities

Get all active entities.

**Query Parameters:**
- `force_id`: Filter by force (Friendly=1, Opposing=2, Neutral=3)
- `marking`: Filter by marking

**Response:**
```json
{
  "entities": [
    {
      "entity_id": {
        "site_id": 1,
        "application_id": 1,
        "entity_id": 100
      },
      "force_id": 1,
      "marking": "VIPER01",
      "entity_type": "F-16C",
      "location": {
        "lat": 38.8977,
        "lon": -77.0365,
        "alt": 10000.0
      },
      "velocity": 300.0,
      "heading": 45.0,
      "last_update": 1700000000000
    }
  ],
  "total": 1
}
```

### DELETE /api/v1/entities/{entity_id}

Remove an entity.

**Response:**
```json
{
  "status": "removed",
  "entity_id": 100
}
```

## Replay Engine API

### POST /api/v1/recordings

Start a new recording.

**Request:**
```json
{
  "name": "EXERCISE-2024-001",
  "description": "Training exercise recording",
  "tags": ["training", "exercise"]
}
```

**Response:**
```json
{
  "recording_id": "REC-001",
  "name": "EXERCISE-2024-001",
  "status": "recording",
  "started_at": 1700000000000
}
```

### GET /api/v1/recordings

List all recordings.

**Response:**
```json
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

### POST /api/v1/recordings/{id}/playback

Start playback of a recording.

**Request:**
```json
{
  "speed": 1.0,
  "start_time": 1700000000000
}
```

**Response:**
```json
{
  "playback_id": "PLAY-001",
  "recording_id": "REC-001",
  "status": "playing",
  "speed": 1.0,
  "current_time": 1700000000000
}
```

## Error Responses

All endpoints use standard HTTP status codes and return error details:

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

### Common Error Codes

| Code | HTTP Status | Description |
|------|-------------|-------------|
| `VALIDATION_ERROR` | 400 | Invalid request data |
| `NOT_FOUND` | 404 | Resource not found |
| `CONFLICT` | 409 | Resource already exists |
| `INTERNAL_ERROR` | 500 | Internal server error |
| `SERVICE_UNAVAILABLE` | 503 | Dependency unavailable |

## Rate Limiting

All APIs are rate-limited:

| Service | Limit | Window |
|---------|-------|--------|
| Health | 60 | 1 minute |
| Metrics | 60 | 1 minute |
| API | 1000 | 1 minute |

Rate limit headers are included in responses:

```http
X-RateLimit-Limit: 1000
X-RateLimit-Remaining: 999
X-RateLimit-Reset: 1700000060
```

## Authentication

API requests require JWT authentication:

```http
Authorization: Bearer <token>
```

Tokens are obtained from the auth service:

```http
POST /api/v1/auth/token
Content-Type: application/json

{
  "client_id": "your-client-id",
  "client_secret": "your-client-secret"
}
```

**Response:**
```json
{
  "access_token": "eyJhbGciOiJSUzI1NiIs...",
  "token_type": "Bearer",
  "expires_in": 3600
}
```