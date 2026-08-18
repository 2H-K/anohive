package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/2H-K/pulse/internal/config"
	"github.com/2H-K/pulse/internal/detector"
	"github.com/2H-K/pulse/internal/models"
	"github.com/2H-K/pulse/internal/storage"
)

func setupTestServer(t *testing.T) (*Server, *storage.SQLiteStore, func()) {
	tmpFile := t.TempDir() + "/test.db"
	store, err := storage.NewSQLiteStore(tmpFile)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}

	cfg := config.DefaultConfig()
	cfg.Storage.DBPath = tmpFile

	det := detector.NewDetector(60)
	server := NewServer(store, det, cfg, "")

	cleanup := func() {
		store.Close()
	}

	return server, store, cleanup
}

func TestHandleHealth(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest("GET", "/api/health", nil)
	w := httptest.NewRecorder()

	server.handleHealth(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
	}

	var result map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &result)

	if result["status"] != "healthy" {
		t.Errorf("status = %v, want healthy", result["status"])
	}
}

func TestHandleIngest(t *testing.T) {
	server, store, cleanup := setupTestServer(t)
	defer cleanup()

	batch := models.LogBatch{
		Source: "test",
		Entries: []models.LogEntry{
			{Message: "test log entry", Level: models.LogLevelInfo},
		},
	}

	body, _ := json.Marshal(batch)
	req := httptest.NewRequest("POST", "/api/logs/ingest", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.handleIngest(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200, body: %s", w.Code, w.Body.String())
	}

	var result map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &result)

	if result["ingested"].(float64) != 1 {
		t.Errorf("ingested = %v, want 1", result["ingested"])
	}

	logs, _ := store.QueryLogs(models.LogQuery{Source: "test"})
	if len(logs) != 1 {
		t.Errorf("stored logs = %d, want 1", len(logs))
	}
}

func TestHandleLogs(t *testing.T) {
	server, store, cleanup := setupTestServer(t)
	defer cleanup()

	now := time.Now()
	entries := []*models.LogEntry{
		{ID: "1", Timestamp: now, Level: models.LogLevelInfo, Message: "test", Source: "test"},
		{ID: "2", Timestamp: now, Level: models.LogLevelError, Message: "error", Source: "test"},
	}
	store.InsertLogs(entries)

	req := httptest.NewRequest("GET", "/api/logs?source=test&limit=10", nil)
	w := httptest.NewRecorder()

	server.handleLogs(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
	}

	var result map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &result)

	if result["total"].(float64) != 2 {
		t.Errorf("total = %v, want 2", result["total"])
	}
}

func TestHandleAnomalies(t *testing.T) {
	server, store, cleanup := setupTestServer(t)
	defer cleanup()

	anomaly := &models.Anomaly{
		ID:        "anm_1",
		Timestamp: time.Now(),
		Type:      models.AnomalySpike,
		Severity:  models.SeverityHigh,
		Source:    "test",
	}
	store.InsertAnomaly(anomaly)

	req := httptest.NewRequest("GET", "/api/anomalies?limit=10", nil)
	w := httptest.NewRecorder()

	server.handleAnomalies(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
	}

	var result map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &result)

	if result["count"].(float64) != 1 {
		t.Errorf("count = %v, want 1", result["count"])
	}
}

func TestHandleStats(t *testing.T) {
	server, store, cleanup := setupTestServer(t)
	defer cleanup()

	entries := []*models.LogEntry{
		{ID: "1", Timestamp: time.Now(), Level: models.LogLevelInfo, Source: "test"},
	}
	store.InsertLogs(entries)

	req := httptest.NewRequest("GET", "/api/stats", nil)
	w := httptest.NewRecorder()

	server.handleStats(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
	}

	var result map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &result)

	if result["total_logs"].(float64) != 1 {
		t.Errorf("total_logs = %v, want 1", result["total_logs"])
	}
}

func TestHandleThresholds(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	config := map[string]interface{}{
		"error_rate": 0.5,
		"burst":      200,
	}

	body, _ := json.Marshal(config)
	req := httptest.NewRequest("PUT", "/api/config/thresholds", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.handleThresholds(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
	}
}

func TestAuthMiddleware(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	handler := server.authMiddleware(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}

	req2 := httptest.NewRequest("GET", "/test", nil)
	req2.Header.Set("X-API-Key", "pulse-dev-key-2024")
	w2 := httptest.NewRecorder()

	handler(w2, req2)

	if w2.Code != http.StatusOK {
		t.Errorf("status with valid key = %d, want %d", w2.Code, http.StatusOK)
	}
}

func TestMethodNotAllowed(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest("DELETE", "/api/logs", nil)
	w := httptest.NewRecorder()

	server.handleLogs(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want 405", w.Code)
	}
}

func TestInvalidIngestBody(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest("POST", "/api/logs/ingest", bytes.NewReader([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.handleIngest(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
}
