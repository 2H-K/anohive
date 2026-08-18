# Pulse Runbook

## Overview

Pulse is a real-time log aggregation and anomaly detection system. This runbook covers common operational procedures, troubleshooting, and maintenance tasks.

## Table of Contents

1. [Architecture](#architecture)
2. [Deployment](#deployment)
3. [Configuration](#configuration)
4. [Monitoring](#monitoring)
5. [Troubleshooting](#troubleshooting)
6. [Maintenance](#maintenance)
7. [Disaster Recovery](#disaster-recovery)
8. [Security](#security)

## Architecture

### Components

- **API Server**: HTTP server handling REST API and WebSocket connections
- **Collector**: Log file collector for tailing local log files
- **Parser**: Multi-format log parser (JSON, Docker, Kubernetes, Syslog, etc.)
- **Detector**: Anomaly detection engine with configurable thresholds
- **Storage**: SQLite database with WAL mode for concurrent access

### Ports

| Port | Protocol | Description |
|------|----------|-------------|
| 8080 | HTTP | API server and WebSocket endpoint |

### Data Flow

```
Log Sources → Collector → Parser → Detector → Storage
                                              ↓
                                          WebSocket → Clients
                                              ↓
                                          REST API → Dashboards
```

## Deployment

### Docker

```bash
# Build and run with Docker Compose
docker-compose up -d

# View logs
docker-compose logs -f pulse

# Scale collector (optional)
docker-compose --profile collector up -d
```

### Kubernetes

```bash
# Deploy to Kubernetes
kubectl apply -k deployments/kubernetes/

# Check deployment status
kubectl -n pulse-monitoring get pods
kubectl -n pulse-monitoring logs -f deployment/pulse

# Port forward for local access
kubectl -n pulse-monitoring port-forward svc/pulse 8080:80
```

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `PULSE_HOST` | `0.0.0.0` | Server bind address |
| `PULSE_PORT` | `8080` | Server port |
| `PULSE_DB_PATH` | `pulse.db` | SQLite database path |
| `PULSE_RETENTION_HOURS` | `168` | Log retention in hours (7 days) |
| `PULSE_MAX_LOGS` | `1000000` | Maximum number of logs to retain |
| `PULSE_API_KEY` | `pulse-dev-key-2024` | API key for authentication |
| `PULSE_LOG_LEVEL` | `info` | Log level (debug/info/warn/error) |
| `PULSE_LOG_FORMAT` | `text` | Log format (text/json) |
| `PULSE_ALLOWED_ORIGINS` | `*` | Allowed CORS origins |
| `PULSE_RATE_LIMIT` | `100` | Rate limit per minute per IP |
| `PULSE_WS_MAX_CONNECTIONS` | `100` | Maximum WebSocket connections |
| `PULSE_MAX_BODY_SIZE` | `10485760` | Maximum request body size (bytes) |
| `PULSE_CLEANUP_INTERVAL` | `3600` | Cleanup interval in seconds |
| `PULSE_CORS_ORIGIN` | `*` | CORS allowed origin |
| `PULSE_ALERT_ENABLED` | `false` | Enable webhook alerts |
| `PULSE_ALERT_WEBHOOK_URL` | `` | Webhook URL for alerts |
| `PULSE_ALERT_COOLDOWN` | `300` | Alert cooldown in seconds |

## Configuration

### Configuration File

Pulse supports JSON configuration files:

```json
{
  "server": {
    "host": "0.0.0.0",
    "port": 8080,
    "read_timeout_seconds": 15,
    "write_timeout_seconds": 15,
    "graceful_timeout_seconds": 30
  },
  "storage": {
    "db_path": "/var/lib/pulse/data/pulse.db",
    "retention": "7d",
    "max_logs": 1000000,
    "cleanup_interval_seconds": 3600
  },
  "detector": {
    "window_size": 60,
    "error_rate_threshold": 0.3,
    "rate_multiplier_threshold": 3.0,
    "burst_threshold": 100
  },
  "security": {
    "api_keys": ["your-secure-api-key"],
    "allowed_origins": ["https://your-domain.com"],
    "rate_limit_per_minute": 100,
    "max_ws_connections": 100
  },
  "log": {
    "level": "info",
    "format": "json"
  },
  "alert": {
    "enabled": false,
    "webhook_url": "https://hooks.slack.com/...",
    "cooldown_seconds": 300,
    "severities": ["critical", "high"]
  }
}
```

### Hot Reload

Configuration can be reloaded without restart:

```bash
curl -X POST -H "X-API-Key: your-api-key" http://localhost:8080/api/config/reload
```

## Monitoring

### Health Checks

| Endpoint | Purpose | Expected Response |
|----------|---------|-------------------|
| `GET /api/health` | General health | `{"status": "healthy"}` |
| `GET /api/health/live` | K8s liveness | `{"status": "alive"}` |
| `GET /api/health/ready` | K8s readiness | `{"status": "ready"}` |

### Resource Monitoring

```bash
# Get resource usage statistics
curl -H "X-API-Key: your-api-key" http://localhost:8080/api/admin/resources
```

Response:
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

### Key Metrics

| Metric | Warning Threshold | Critical Threshold |
|--------|------------------|-------------------|
| Memory Usage | > 300 MB | > 400 MB |
| Goroutines | > 5000 | > 10000 |
| Degradation Level | 1 (Light) | 3 (Severe) |
| Circuit Breaker | - | Open |

### Prometheus Metrics

Prometheus-compatible metrics are available at `GET /api/metrics`.

JSON format metrics at `GET /api/metrics/json`.

## Troubleshooting

### High Memory Usage

**Symptoms**: Memory usage exceeds 400MB, degradation level >= 2

**Resolution**:
1. Check log ingestion rate: `curl http://localhost:8080/api/stats`
2. Reduce retention period: Set `PULSE_RETENTION_HOURS` to a lower value
3. Enable log sampling (automatic under high load)
4. Force GC: Not directly controllable, but system triggers automatically

### Database Errors

**Symptoms**: Circuit breaker opens, write operations fail

**Resolution**:
1. Check database file permissions
2. Verify disk space: `df -h /var/lib/pulse/data`
3. Check for database corruption: `PRAGMA integrity_check`
4. Restart server if needed

### WebSocket Connection Issues

**Symptoms**: Clients disconnect frequently, connection refused

**Resolution**:
1. Check max connections: `PULSE_WS_MAX_CONNECTIONS`
2. Verify origin is in allowed origins
3. Check network/firewall settings
4. Review server logs for errors

### High Ingestion Latency

**Symptoms**: Logs appear with delay, API responses slow

**Resolution**:
1. Check resource usage: `curl -H "X-API-Key: ..." http://localhost:8080/api/admin/resources`
2. Increase `PULSE_MAX_LOGS` if hitting limit
3. Reduce batch size in client
4. Check disk I/O performance

## Maintenance

### Database Backup

```bash
# Using CLI tool
pulse backup -db /var/lib/pulse/data/pulse.db -output /backup/pulse-$(date +%Y%m%d).db

# Using SQLite directly
sqlite3 /var/lib/pulse/data/pulse.db "VACUUM INTO '/backup/pulse-backup.db'"
```

### Database Restore

```bash
# Using CLI tool
pulse restore -db /var/lib/pulse/data/pulse.db -input /backup/pulse-backup.db

# Manual restore
cp /backup/pulse-backup.db /var/lib/pulse/data/pulse.db
```

### Log Rotation

Pulse automatically cleans up logs based on retention policy. Manual cleanup:

```bash
# Force cleanup by reducing retention
curl -X PUT -H "X-API-Key: ..." -H "Content-Type: application/json" \
  -d '{"retention_hours": 24}' http://localhost:8080/api/config/thresholds

# Then restore original retention
curl -X POST -H "X-API-Key: ..." http://localhost:8080/api/config/reload
```

### Database Vacuum

To reclaim disk space after large deletions:

```bash
sqlite3 /var/lib/pulse/data/pulse.db "VACUUM"
```

## Disaster Recovery

### Backup Strategy

1. **Automated Backups**: Schedule daily backups using cron
2. **Offsite Storage**: Copy backups to S3 or similar
3. **Retention**: Keep 7 days of backups

Example cron job:
```bash
0 2 * * * /usr/local/bin/pulse backup -db /var/lib/pulse/data/pulse.db -output /backup/pulse-$(date +\%Y\%m\%d).db
```

### Recovery Procedure

1. Stop the pulse service
2. Restore from backup
3. Verify data integrity
4. Start the service
5. Verify health checks pass

```bash
# Stop service
kubectl rollout pause deployment/pulse -n pulse-monitoring
# or
docker-compose stop pulse

# Restore backup
pulse restore -db /var/lib/pulse/data/pulse.db -input /backup/pulse-latest.db

# Start service
kubectl rollout resume deployment/pulse -n pulse-monitoring
# or
docker-compose start pulse

# Verify
curl http://localhost:8080/api/health
```

## Security

### API Key Management

1. **Rotate API Keys**: Change keys periodically
2. **Use Strong Keys**: At least 32 characters, random
3. **Restrict Origins**: Set `PULSE_ALLOWED_ORIGINS` to specific domains
4. **Rate Limiting**: Adjust `PULSE_RATE_LIMIT` based on expected traffic

### Network Security

1. **TLS Termination**: Use Ingress or LoadBalancer with TLS
2. **Network Policies**: Restrict pod-to-pod communication
3. **Firewall**: Only expose necessary ports

### Security Best Practices

1. Run as non-root user (Dockerfile includes this)
2. Use Kubernetes security contexts
3. Enable audit logging
4. Regular security scans of container images
