# AnoHive - Real-time Log Monitor

[中文文档](README_CN.md) | [English](README.md)

AnoHive is a high-performance, real-time log aggregation and anomaly detection system. It collects logs from multiple sources, parses various log formats, detects anomalies in real-time, and provides a web dashboard for monitoring.

## Features

- **Multi-format Log Parsing**: JSON, Docker, Kubernetes, Log4j, Apache, Nginx, Syslog, leveled, and generic text
- **Real-time Anomaly Detection**: Error spikes, log bursts, new error patterns, rate changes
- **REST API**: Full-featured API for log ingestion and querying
- **Raw Log Ingestion**: POST raw log lines for automatic format detection and parsing
- **WebSocket Streaming**: Real-time log streaming to connected clients
- **Web Dashboard**: Dark-themed monitoring UI with virtual scrolling, live filtering, and WebSocket support
- **CLI Tool**: Command-line interface for interacting with the server
- **SQLite Storage**: WAL mode with connection pooling for high concurrency
- **Prometheus Metrics**: /api/metrics endpoint for monitoring
- **Log Retention**: Automatic cleanup of old logs based on configurable retention period
- **High Performance**: Handles 15,000+ entries/second on modest hardware

## Architecture

```
┌─────────────┐     ┌──────────────┐     ┌─────────────┐
│ Log Sources │────▶│  Collector   │────▶│   Parser    │
└─────────────┘     └──────────────┘     └──────┬──────┘
                                                │
                   ┌──────────────┐     ┌──────▼──────┐
                   │  Detector    │◀────│  Log Entry  │
                   └──────┬───────┘     └─────────────┘
                          │
        ┌─────────────────┼─────────────────┐
        ▼                 ▼                  ▼
┌──────────────┐  ┌──────────────┐  ┌──────────────┐
│   SQLite     │  │   WebSocket   │  │  REST API    │
│   Storage    │  │   Broadcast   │  │              │
└──────────────┘  └──────────────┘  └──────────────┘
```

## Quick Start

### Build

```bash
make build
```

### Run Server

```bash
./build/pulse -port 8080
```

### Ingest Logs

```bash
# Single log entry
curl -X POST http://localhost:8080/api/logs/ingest \
  -H "Content-Type: application/json" \
  -d '{"source": "myapp", "entries": [{"level": "ERROR", "message": "Connection failed"}]}'

# Batch ingestion
curl -X POST http://localhost:8080/api/logs/ingest \
  -H "Content-Type: application/json" \
  -d '{"source": "myapp", "entries": [
    {"level": "INFO", "message": "Server started"},
    {"level": "ERROR", "message": "Database error"},
    {"level": "WARN", "message": "High memory usage"}
  ]}'
```

### Ingest Raw Logs (Auto-Parse)

```bash
curl -X POST http://localhost:8080/api/logs/raw \
  -H "Content-Type: application/json" \
  -d '{"source": "myapp", "lines": [
    "stdout | 2024-01-15T10:30:00.123456789Z Server started",
    "stderr | 2024-01-15T10:30:01.123456789Z ERROR: Connection failed",
    "2024-01-15 10:30:00.123 ERROR [main] com.example.Service - Failed"
  ]}'
```

### Query Logs

```bash
# Get recent logs
curl "http://localhost:8080/api/logs?limit=10"

# Filter by level
curl "http://localhost:8080/api/logs?level=ERROR"

# Search
curl "http://localhost:8080/api/logs?search=database"

# Combined filters
curl "http://localhost:8080/api/logs?source=myapp&level=ERROR&limit=20"
```

### CLI Usage

```bash
# Check server health
./build/anohive-cli health

# View recent logs
./build/anohive-cli logs --level ERROR --limit 10

# Ingest logs
./build/anohive-cli ingest --level ERROR "Something went wrong"

# View anomalies
./build/anohive-cli anomalies --severity CRITICAL

# Stream real-time logs
./build/anohive-cli stream
```

## API Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | /api/health | Health check |
| GET | /api/logs | Query logs (supports filters) |
| POST | /api/logs/ingest | Ingest structured log entries |
| POST | /api/logs/raw | Ingest raw log lines (auto-parse) |
| GET | /api/anomalies | List detected anomalies |
| GET | /api/stats | System statistics |
| GET | /api/sources | List active sources |
| PUT | /api/config/thresholds | Update detection thresholds |
| GET | /api/metrics | Prometheus-compatible metrics |
| GET | /api/metrics/json | JSON-formatted metrics |
| WS | /ws | WebSocket for real-time streaming |

## Configuration

### Server Flags

| Flag | Default | Description |
|------|---------|-------------|
| -host | 0.0.0.0 | Server host |
| -port | 8080 | Server port |
| -db | anohive.db | SQLite database path |
| -static | | Static files directory |
| -buffer | 10000 | Collector buffer size |

### Detection Thresholds

```bash
# Update via API
curl -X PUT http://localhost:8080/api/config/thresholds \
  -H "Content-Type: application/json" \
  -d '{"error_rate": 0.5, "rate_multiplier": 4.0, "burst": 200}'

# Update via CLI
./build/anohive-cli config error_rate 0.5
./build/anohive-cli config burst 200
```

## Supported Log Formats

1. **JSON**: Structured logs with level/message fields
   ```
   {"time":"2024-01-15T10:30:00Z","level":"ERROR","message":"connection failed"}
   ```

2. **Docker**: Container stdout/stderr with timestamps
   ```
   stdout | 2024-01-15T10:30:00.123456789Z Server started
   stderr | 2024-01-15T10:30:01.123456789Z ERROR: Connection failed
   ```

3. **Kubernetes**: Pod/container prefixed logs
   ```
   E 2024-01-15T10:30:00Z my-pod my-container Error: pod crashed
   ```

4. **RFC 5424 Syslog**: Standard syslog with structured data
   ```
   <134>1 2024-01-15T10:30:00.000Z myhost appname 1234 ID47 - Message
   ```

5. **Log4j/Logback**: Java logging framework format
   ```
   2024-01-15 10:30:00.123 ERROR [main] com.example.Service - Failed
   ```

6. **Syslog**: Traditional BSD syslog format
   ```
   Jan 15 10:30:00 myhost sshd[1234]: Accepted publickey for user
   ```

7. **Leveled**: Timestamp + level + message
   ```
   2024-01-15 10:30:00 ERROR Something broke
   ```

8. **Apache Combined**: Full Apache access log format
   ```
   192.168.1.1 - frank [15/Jan/2024:10:30:00 +0000] "GET /index.html HTTP/1.1" 200 1234 "http://example.com" "Mozilla/5.0"
   ```

9. **Nginx**: Combined access log format
   ```
   192.168.1.1 - - [15/Jan/2024:10:30:00 +0000] "GET /api/health HTTP/1.1" 200 42
   ```

10. **Generic**: Auto-detects ERROR/WARN/DEBUG/FATAL keywords

## Anomaly Detection

The system detects four types of anomalies:

1. **Error Spike** (ERROR_SPIKE): When error rate exceeds threshold (default 30%)
2. **Log Burst** (LOG_BURST): When many logs arrive in short time window
3. **New Error Pattern** (NEW_ERROR_TYPE): When a new error message appears
4. **Rate Change** (RATE_CHANGE): When log volume increases by multiplier (default 3x)

## Testing

```bash
# Run unit tests
make test

# Run with coverage
go test -cover ./internal/...

# Run load tests (requires running server)
go test -v ./test/
```

## Project Structure

```
anohive/anohive/
├── cmd/
│   ├── server/        # Server entry point
│   └── cli/           # CLI entry point
├── internal/
│   ├── api/           # HTTP handlers + WebSocket
│   ├── collector/     # Log collection from sources
│   ├── config/        # Configuration management
│   ├── detector/      # Anomaly detection engine
│   ├── models/        # Data models
│   ├── parser/        # Multi-format log parser
│   └── storage/       # SQLite storage layer
├── web/               # React frontend
│   └── src/
│       ├── components/ # React components
│       ├── services/   # API client
│       └── styles/     # CSS styles
├── test/              # Integration and load tests
├── Makefile           # Build automation
└── README.md
```

## Performance

Tested on local machine:
- **Ingestion**: 15,000+ entries/second (batch with SQLite WAL mode)
- **Raw ingestion**: 1,800+ lines/second (with format parsing)
- **Concurrent operations**: 50 concurrent writers without errors
- **Parser**: 500K-1.5M parses/second (depending on format)
- **Detector**: 1.7M-3M entries/second
- **Memory**: ~3.5MB for 23K log entries

## License

MIT
