# Pulse API Documentation

## Base URL

```
http://localhost:8080
```

## Authentication

Protected endpoints require an API key passed via:
- Header: `X-API-Key: your-api-key`
- Query parameter: `?api_key=your-api-key`

## Endpoints

### Health

#### GET /api/health

Returns server health status.

**Response:**
```json
{
  "status": "healthy",
  "uptime": "2h30m0s",
  "timestamp": "2026-08-18T03:00:00Z"
}
```

#### GET /api/health/live

Kubernetes liveness probe.

**Response:**
```json
{
  "status": "alive"
}
```

#### GET /api/health/ready

Kubernetes readiness probe.

**Response:**
```json
{
  "status": "ready"
}
```

### Logs

#### GET /api/v1/logs

Query log entries with filters.

**Query Parameters:**

| Parameter | Type | Description |
|-----------|------|-------------|
| source | string | Filter by source name |
| level | string | Filter by level (DEBUG/INFO/WARN/ERROR/FATAL) |
| search | string | Search in log messages |
| start | string | Start time (RFC3339) |
| end | string | End time (RFC3339) |
| limit | int | Number of results (default: 50) |
| offset | int | Offset for pagination (default: 0) |

**Response:**
```json
{
  "logs": [
    {
      "id": "log_123",
      "timestamp": "2026-08-18T03:00:00Z",
      "level": "ERROR",
      "message": "Connection failed",
      "source": "api-gateway",
      "host": "server-01",
      "service": "auth",
      "fields": {"user_id": "123"},
      "raw": "original log line"
    }
  ],
  "total": 100,
  "limit": 50,
  "offset": 0
}
```

#### POST /api/v1/logs/ingest

Ingest structured log entries. **Requires authentication.**

**Request Body (Batch):**
```json
{
  "source": "my-service",
  "entries": [
    {
      "level": "INFO",
      "message": "Request processed",
      "timestamp": "2026-08-18T03:00:00Z",
      "host": "server-01",
      "service": "api",
      "fields": {"duration_ms": 150}
    }
  ]
}
```

**Request Body (Single):**
```json
{
  "level": "ERROR",
  "message": "Database connection failed",
  "source": "my-service"
}
```

**Response:**
```json
{
  "ingested": 1,
  "status": "ok"
}
```

#### POST /api/v1/logs/raw

Ingest raw log lines for automatic parsing. **Requires authentication.**

**Request Body:**
```json
{
  "source": "nginx",
  "lines": [
    "192.168.1.1 - - [18/Aug/2026:03:00:00 +0000] \"GET /api/users HTTP/1.1\" 200 1234",
    "192.168.1.1 - - [18/Aug/2026:03:00:01 +0000] \"POST /api/login HTTP/1.1\" 401 567"
  ]
}
```

**Response:**
```json
{
  "ingested": 2,
  "status": "ok"
}
```

### Anomalies

#### GET /api/v1/anomalies

Query detected anomalies.

**Query Parameters:**

| Parameter | Type | Description |
|-----------|------|-------------|
| limit | int | Number of results (default: 100) |
| severity | string | Filter by severity (LOW/MEDIUM/HIGH/CRITICAL) |

**Response:**
```json
{
  "anomalies": [
    {
      "id": "anm_123",
      "timestamp": "2026-08-18T03:00:00Z",
      "type": "ERROR_SPIKE",
      "severity": "HIGH",
      "description": "Error rate exceeded threshold",
      "source": "api-gateway",
      "metadata": {"error_rate": "0.45"}
    }
  ],
  "count": 1
}
```

### Statistics

#### GET /api/v1/stats

Get system statistics.

**Response:**
```json
{
  "uptime": "2h30m0s",
  "total_sources": 5,
  "total_logs": 10000,
  "total_anomalies": 15,
  "sources_stats": [
    {
      "source": "api-gateway",
      "total_logs": 5000,
      "error_count": 50,
      "warn_count": 100,
      "logs_per_min": 100,
      "top_errors": ["Connection failed", "Timeout"],
      "last_log_time": "2026-08-18T03:00:00Z",
      "level_distribution": {"INFO": 4500, "ERROR": 50, "WARN": 100}
    }
  ]
}
```

#### GET /api/v1/sources

Get source statistics.

**Response:**
```json
[
  {
    "source": "api-gateway",
    "total_logs": 5000,
    "error_count": 50,
    "warn_count": 100
  }
]
```

### Configuration

#### PUT /api/v1/config/thresholds

Update anomaly detection thresholds. **Requires authentication.**

**Request Body:**
```json
{
  "error_rate": 0.5,
  "rate_multiplier": 5.0,
  "burst": 200
}
```

**Response:**
```json
{
  "status": "updated"
}
```

#### POST /api/v1/config/reload

Reload configuration from file. **Requires authentication.**

**Response:**
```json
{
  "status": "reloaded",
  "message": "configuration reloaded successfully"
}
```

### Admin

#### GET /api/v1/admin/resources

Get resource usage statistics. **Requires authentication.**

**Response:**
```json
{
  "resources": {
    "memory_mb": 45,
    "memory_sys_mb": 120,
    "goroutines": 15,
    "gc_cycles": 10,
    "degradation": 0
  },
  "sampler": {
    "rate": 1.0,
    "total": 1000,
    "sampled": 1000,
    "dropped": 0
  },
  "circuit": "closed"
}
```

### Metrics

#### GET /api/metrics

Prometheus-format metrics.

**Response:**
```
# HELP pulse_logs_total Total number of logs ingested
# TYPE pulse_logs_total counter
pulse_logs_total 10000

# HELP pulse_anomalies_total Total number of anomalies detected
# TYPE pulse_anomalies_total counter
pulse_anomalies_total 15

# HELP pulse_requests_total Total API requests
# TYPE pulse_requests_total counter
pulse_requests_total 5000

# HELP pulse_websocket_clients Current WebSocket clients
# TYPE pulse_websocket_clients gauge
pulse_websocket_clients 5
```

#### GET /api/metrics/json

JSON format metrics.

**Response:**
```json
{
  "logs_total": 10000,
  "anomalies_total": 15,
  "requests_total": 5000,
  "active_clients": 5,
  "storage_size_bytes": 1048576
}
```

### WebSocket

#### WS /ws

Real-time log streaming via WebSocket.

**Connection:**
```
ws://localhost:8080/ws
```

**Messages:**
```json
{
  "id": "log_123",
  "timestamp": "2026-08-18T03:00:00Z",
  "level": "INFO",
  "message": "New log entry",
  "source": "my-service"
}
```

## Error Responses

All endpoints return standard HTTP status codes:

| Code | Description |
|------|-------------|
| 200 | Success |
| 400 | Bad Request |
| 401 | Unauthorized |
| 404 | Not Found |
| 405 | Method Not Allowed |
| 429 | Too Many Requests |
| 500 | Internal Server Error |
| 503 | Service Unavailable |

**Error Response Format:**
```json
{
  "error": "Invalid JSON payload"
}
```

## Rate Limiting

Rate limiting is applied per client IP address. Default: 100 requests per minute.

**Rate Limit Headers:**
```
X-RateLimit-Limit: 100
X-RateLimit-Remaining: 95
X-RateLimit-Reset: 1724000000
```

When rate limit is exceeded:
```json
{
  "error": "Too many requests"
}
```
