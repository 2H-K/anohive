#!/bin/bash
# Pulse Demo Script
# This script demonstrates the Pulse log monitoring system

set -e

SERVER_URL="${PULSE_URL:-http://localhost:8080}"
CLI="./build/pulse-cli"

echo "=========================================="
echo "  Pulse - Real-time Log Monitor Demo"
echo "=========================================="
echo ""

# Check server health
echo "1. Checking server health..."
curl -s "$SERVER_URL/api/health" | python3 -m json.tool
echo ""

# Ingest sample logs
echo "2. Ingesting sample logs..."
curl -s -X POST "$SERVER_URL/api/logs/ingest" \
  -H "Content-Type: application/json" \
  -d '{
    "source": "web-server",
    "entries": [
      {"level": "INFO", "message": "Server started on port 8080"},
      {"level": "INFO", "message": "Connected to database"},
      {"level": "INFO", "message": "Cache initialized"},
      {"level": "WARN", "message": "High memory usage: 75%"},
      {"level": "ERROR", "message": "Database connection timeout"},
      {"level": "ERROR", "message": "Database connection timeout"},
      {"level": "ERROR", "message": "Database connection timeout"},
      {"level": "ERROR", "message": "Failed to process request"},
      {"level": "FATAL", "message": "Out of memory error"},
      {"level": "INFO", "message": "Request processed successfully"}
    ]
  }' | python3 -m json.tool
echo ""

# Ingest JSON format logs
echo "3. Ingesting JSON format logs..."
curl -s -X POST "$SERVER_URL/api/logs/ingest" \
  -H "Content-Type: application/json" \
  -d '{
    "source": "api-gateway",
    "entries": [
      {"level": "INFO", "message": "GET /api/users 200 OK"},
      {"level": "INFO", "message": "POST /api/orders 201 Created"},
      {"level": "WARN", "message": "Rate limit approaching for client 123"},
      {"level": "ERROR", "message": "GET /api/products 500 Internal Server Error"},
      {"level": "ERROR", "message": "POST /api/payment 502 Bad Gateway"}
    ]
  }' | python3 -m json.tool
echo ""

# Query logs
echo "4. Querying ERROR logs..."
curl -s "$SERVER_URL/api/logs?level=ERROR&limit=5" | python3 -m json.tool 2>/dev/null | head -30
echo ""

# Check stats
echo "5. System statistics..."
curl -s "$SERVER_URL/api/stats" | python3 -m json.tool
echo ""

# Check anomalies
echo "6. Detected anomalies..."
curl -s "$SERVER_URL/api/anomalies?limit=10" | python3 -m json.tool 2>/dev/null | head -40
echo ""

# Check sources
echo "7. Active sources..."
curl -s "$SERVER_URL/api/sources" | python3 -m json.tool
echo ""

echo "=========================================="
echo "  Demo complete!"
echo "=========================================="
