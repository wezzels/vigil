# VIGIL API Documentation

## Base URL

```
https://vigil.local/api/v1
```

## Authentication

All endpoints require authentication via one of:
- **mTLS**: Client certificate validation
- **JWT**: Bearer token in Authorization header
- **API Key**: X-API-Key header

## Endpoints

### Tracks

#### List Tracks

```http
GET /tracks
```

**Parameters:**

| Name | Type | Location | Description |
|------|------|----------|-------------|
| limit | int | query | Maximum tracks to return (default: 100) |
| offset | int | query | Pagination offset |
| source | string | query | Filter by source (OPIR, RADAR, SBIRS) |
| status | string | query | Filter by status (active, dropped) |

**Response:**

```json
{
  "tracks": [
    {
      "id": "track-001",
      "track_number": "TN001",
      "source": "OPIR",
      "position": {
        "latitude": 34.0522,
        "longitude": -118.2437,
        "altitude": 10000.0
      },
      "velocity": {
        "x": 100.0,
        "y": 200.0,
        "z": 50.0
      },
      "identity": "hostile",
      "quality": "high",
      "confidence": 0.95,
      "created_at": "2026-04-14T12:00:00Z",
      "updated_at": "2026-04-14T12:01:00Z"
    }
  ],
  "total": 150,
  "limit": 100,
  "offset": 0
}
```

#### Get Track

```http
GET /tracks/{id}
```

**Response:**

```json
{
  "id": "track-001",
  "track_number": "TN001",
  "source": "OPIR",
  "position": {
    "latitude": 34.0522,
    "longitude": -118.2437,
    "altitude": 10000.0
  },
  "velocity": {
    "x": 100.0,
    "y": 200.0,
    "z": 50.0
  },
  "identity": "hostile",
  "quality": "high",
  "confidence": 0.95,
  "created_at": "2026-04-14T12:00:00Z",
  "updated_at": "2026-04-14T12:01:00Z"
}
```

#### Create Track

```http
POST /tracks
```

**Request:**

```json
{
  "track_number": "TN001",
  "source": "OPIR",
  "position": {
    "latitude": 34.0522,
    "longitude": -118.2437,
    "altitude": 10000.0
  },
  "velocity": {
    "x": 100.0,
    "y": 200.0,
    "z": 50.0
  },
  "identity": "unknown"
}
```

**Response:** `201 Created`

#### Update Track

```http
PUT /tracks/{id}
```

**Request:**

```json
{
  "position": {
    "latitude": 34.0530,
    "longitude": -118.2440,
    "altitude": 10100.0
  },
  "velocity": {
    "x": 105.0,
    "y": 205.0,
    "z": 55.0
  }
}
```

**Response:** `200 OK`

#### Delete Track

```http
DELETE /tracks/{id}
```

**Response:** `204 No Content`

---

### Alerts

#### List Alerts

```http
GET /alerts
```

**Parameters:**

| Name | Type | Location | Description |
|------|------|----------|-------------|
| limit | int | query | Maximum alerts to return |
| offset | int | query | Pagination offset |
| priority | string | query | Filter by priority |
| status | string | query | Filter by status |

**Response:**

```json
{
  "alerts": [
    {
      "id": "alert-001",
      "type": "CONOPREP",
      "priority": "critical",
      "track_id": "track-001",
      "status": "pending",
      "created_at": "2026-04-14T12:00:00Z"
    }
  ],
  "total": 50
}
```

#### Get Alert

```http
GET /alerts/{id}
```

#### Create Alert

```http
POST /alerts
```

**Request:**

```json
{
  "type": "IMMINENT",
  "priority": "high",
  "track_id": "track-001",
  "message": "Missile launch detected"
}
```

#### Acknowledge Alert

```http
POST /alerts/{id}/acknowledge
```

**Request:**

```json
{
  "acknowledged_by": "operator-001",
  "notes": "Confirming alert"
}
```

#### Complete Alert

```http
POST /alerts/{id}/complete
```

**Request:**

```json
{
  "completed_by": "operator-001",
  "resolution": "Threat mitigated"
}
```

---

### Events

#### List Events

```http
GET /events
```

**Parameters:**

| Name | Type | Location | Description |
|------|------|----------|-------------|
| limit | int | query | Maximum events to return |
| type | string | query | Filter by event type |
| start_time | string | query | Start time (RFC3339) |
| end_time | string | query | End time (RFC3339) |

**Response:**

```json
{
  "events": [
    {
      "id": "event-001",
      "type": "track_created",
      "track_id": "track-001",
      "timestamp": "2026-04-14T12:00:00Z",
      "data": {}
    }
  ]
}
```

---

### Health

#### Liveness

```http
GET /healthz/liveness
```

**Response:**

```json
{
  "status": "healthy"
}
```

#### Readiness

```http
GET /healthz/readiness
```

**Response:**

```json
{
  "status": "healthy",
  "checks": {
    "database": "healthy",
    "redis": "healthy",
    "kafka": "healthy"
  }
}
```

#### Startup

```http
GET /healthz/startup
```

---

## Error Responses

All errors follow this format:

```json
{
  "error": {
    "code": "TRACK_NOT_FOUND",
    "message": "Track with ID track-001 not found",
    "details": {}
  }
}
```

### Error Codes

| Code | HTTP Status | Description |
|------|-------------|-------------|
| UNAUTHORIZED | 401 | Authentication required |
| FORBIDDEN | 403 | Insufficient permissions |
| NOT_FOUND | 404 | Resource not found |
| VALIDATION_ERROR | 400 | Invalid request data |
| INTERNAL_ERROR | 500 | Internal server error |
| TRACK_NOT_FOUND | 404 | Track not found |
| ALERT_NOT_FOUND | 404 | Alert not found |
| INVALID_PRIORITY | 400 | Invalid alert priority |
| DUPLICATE_TRACK | 409 | Track already exists |

---

## Rate Limiting

| Endpoint | Limit |
|----------|-------|
| GET /tracks | 1000/min |
| POST /tracks | 100/min |
| GET /alerts | 1000/min |
| POST /alerts | 50/min |

Rate limit headers:

```
X-RateLimit-Limit: 1000
X-RateLimit-Remaining: 999
X-RateLimit-Reset: 1681466400
```

---

## Webhooks

### Alert Webhook

```json
{
  "event": "alert.created",
  "timestamp": "2026-04-14T12:00:00Z",
  "data": {
    "id": "alert-001",
    "type": "CONOPREP",
    "priority": "critical"
  }
}
```

### Track Webhook

```json
{
  "event": "track.created",
  "timestamp": "2026-04-14T12:00:00Z",
  "data": {
    "id": "track-001",
    "source": "OPIR"
  }
}
```

---

**Version:** 1.0.0
**Last Updated:** 2026-04-14