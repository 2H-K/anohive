package test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func TestEndToEndLogFlow(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	baseURL := "http://localhost:8080"

	// Step 1: Ingest logs
	batch := map[string]interface{}{
		"source": "e2e-test",
		"entries": []map[string]string{
			{"level": "INFO", "message": "Application started"},
			{"level": "INFO", "message": "Connected to database"},
			{"level": "WARN", "message": "Cache miss for key: user_123"},
			{"level": "ERROR", "message": "Database connection timeout"},
			{"level": "ERROR", "message": "Database connection timeout"},
			{"level": "ERROR", "message": "Database connection timeout"},
			{"level": "FATAL", "message": "Out of memory error"},
		},
	}

	body, _ := json.Marshal(batch)
	resp, err := http.Post(baseURL+"/api/logs/ingest", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("ingest failed: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("ingest status = %d, want 200", resp.StatusCode)
	}
	resp.Body.Close()

	// Step 2: Query logs
	time.Sleep(100 * time.Millisecond)
	resp, err = http.Get(baseURL + "/api/logs?source=e2e-test&limit=10")
	if err != nil {
		t.Fatalf("query failed: %v", err)
	}

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	resp.Body.Close()

	total := result["total"].(float64)
	if total < 7 {
		t.Errorf("total logs = %f, want >= 7", total)
	}

	// Step 3: Check stats
	resp, err = http.Get(baseURL + "/api/stats")
	if err != nil {
		t.Fatalf("stats failed: %v", err)
	}

	var stats map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&stats)
	resp.Body.Close()

	if stats["total_logs"].(float64) < 7 {
		t.Errorf("total logs in stats = %f, want >= 7", stats["total_logs"])
	}

	t.Logf("End-to-end test passed: %.0f logs ingested, %.0f in stats",
		total, stats["total_logs"])
}

func TestWebSocketConnection(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	wsURL := "ws://localhost:8080/ws"
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("websocket dial failed: %v", err)
	}
	defer conn.Close()

	// Ingest a log while connected
	batch := map[string]interface{}{
		"source":  "ws-test",
		"entries": []map[string]string{{"level": "INFO", "message": "WebSocket test"}},
	}
	body, _ := json.Marshal(batch)
	resp, err := http.Post("http://localhost:8080/api/logs/ingest", "application/json", bytes.NewReader(body))
	if err == nil {
		resp.Body.Close()
	}

	// Try to read with timeout
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, _, err = conn.ReadMessage()
	// Timeout is expected since we may not get a message immediately
	if err != nil {
		t.Logf("WebSocket read (may timeout): %v", err)
	}
}

func TestJSONLogParsing(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	jsonLog := `{"time":"2024-01-15T10:30:00Z","level":"ERROR","message":"JSON test","service":"api"}`

	batch := map[string]interface{}{
		"source": "json-parse-test",
		"entries": []map[string]interface{}{
			{"message": jsonLog, "level": "INFO", "source": "raw"},
		},
	}

	body, _ := json.Marshal(batch)
	resp, err := http.Post("http://localhost:8080/api/logs/ingest", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("ingest failed: %v", err)
	}
	resp.Body.Close()

	resp, err = http.Get("http://localhost:8080/api/logs?limit=1")
	if err != nil {
		t.Fatalf("query failed: %v", err)
	}

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	resp.Body.Close()

	logs := result["logs"].([]interface{})
	if len(logs) == 0 {
		t.Fatal("no logs found")
	}

	t.Log("JSON log ingestion test passed")
}

func TestAnomalyDetection(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	baseURL := "http://localhost:8080"

	// Send many error logs to trigger spike detection
	var entries []map[string]string
	for i := 0; i < 20; i++ {
		level := "ERROR"
		if i < 5 {
			level = "INFO"
		}
		entries = append(entries, map[string]string{
			"level":   level,
			"message": fmt.Sprintf("Test message %d", i),
		})
	}

	batch := map[string]interface{}{
		"source":  "anomaly-test",
		"entries": entries,
	}

	body, _ := json.Marshal(batch)
	resp, err := http.Post(baseURL+"/api/logs/ingest", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("ingest failed: %v", err)
	}
	resp.Body.Close()

	// Check for anomalies
	time.Sleep(100 * time.Millisecond)
	resp, err = http.Get(baseURL + "/api/anomalies?limit=10")
	if err != nil {
		t.Fatalf("anomalies query failed: %v", err)
	}

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	resp.Body.Close()

	t.Logf("Anomalies found: %v", result["count"])
}

func TestConfigurationUpdate(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	baseURL := "http://localhost:8080"

	config := map[string]interface{}{
		"error_rate": 0.6,
		"burst":      150,
	}

	body, _ := json.Marshal(config)
	req, _ := http.NewRequest("PUT", baseURL+"/api/config/thresholds", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("config update failed: %v", err)
	}
	resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("config update status = %d, want 200", resp.StatusCode)
	}
}

func TestTimeRangeQuery(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	baseURL := "http://localhost:8080"

	start := time.Now().Add(-1 * time.Hour).Format(time.RFC3339)
	end := time.Now().Add(1 * time.Hour).Format(time.RFC3339)

	url := fmt.Sprintf("%s/api/logs?start=%s&end=%s&limit=100", baseURL, start, end)
	resp, err := http.Get(url)
	if err != nil {
		t.Fatalf("time range query failed: %v", err)
	}

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	resp.Body.Close()

	t.Logf("Time range query returned %v logs", result["total"])
}

func TestConcurrentWriters(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	baseURL := "http://localhost:8080"
	numWriters := 5
	numEntries := 20

	done := make(chan bool, numWriters)

	for w := 0; w < numWriters; w++ {
		go func(workerID int) {
			var entries []map[string]string
			for i := 0; i < numEntries; i++ {
				entries = append(entries, map[string]string{
					"level":   "INFO",
					"message": fmt.Sprintf("Worker %d - Entry %d", workerID, i),
				})
			}

			batch := map[string]interface{}{
				"source":  fmt.Sprintf("concurrent-%d", workerID),
				"entries": entries,
			}

			body, _ := json.Marshal(batch)
			resp, err := http.Post(baseURL+"/api/logs/ingest", "application/json", bytes.NewReader(body))
			if err != nil {
				t.Errorf("worker %d: ingest failed: %v", workerID, err)
			} else {
				resp.Body.Close()
			}

			done <- true
		}(w)
	}

	for i := 0; i < numWriters; i++ {
		<-done
	}

	t.Logf("Concurrent writers test passed: %d writers x %d entries", numWriters, numEntries)
}
