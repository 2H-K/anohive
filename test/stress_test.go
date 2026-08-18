package test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestStressIngestion(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping stress test in short mode")
	}

	baseURL := "http://localhost:8080"
	totalEntries := 10000
	concurrency := 50
	entriesPerRequest := totalEntries / concurrency

	var wg sync.WaitGroup
	var errorCount int64
	var successCount int64

	start := time.Now()

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			batch := map[string]interface{}{
				"source":  fmt.Sprintf("stress-%d", workerID),
				"entries": generateStressEntries(entriesPerRequest, workerID),
			}

			body, _ := json.Marshal(batch)
			resp, err := http.Post(
				baseURL+"/api/logs/ingest",
				"application/json",
				bytes.NewReader(body),
			)
			if err != nil {
				atomic.AddInt64(&errorCount, 1)
				return
			}
			resp.Body.Close()

			if resp.StatusCode == http.StatusOK {
				atomic.AddInt64(&successCount, 1)
			} else {
				atomic.AddInt64(&errorCount, 1)
			}
		}(i)
	}

	wg.Wait()

	duration := time.Since(start)
	rate := float64(totalEntries) / duration.Seconds()

	t.Logf("Stress test: %d entries in %.2f seconds (%.0f entries/sec)", totalEntries, duration.Seconds(), rate)
	t.Logf("Success: %d, Errors: %d", successCount, errorCount)

	if errorCount > 0 {
		t.Errorf("Had %d errors during stress test", errorCount)
	}
}

func TestStressRawIngestion(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping stress test in short mode")
	}

	baseURL := "http://localhost:8080"
	numRequests := 100
	linesPerRequest := 50

	diverseLogs := []string{
		`stdout | 2024-01-15T10:30:00.123456789Z Server started`,
		`stderr | 2024-01-15T10:30:01.123456789Z ERROR: Connection failed`,
		`{"time":"2024-01-15T10:30:00Z","level":"INFO","message":"JSON log"}`,
		`2024-01-15 10:30:00 ERROR Something broke`,
		`192.168.1.1 - - [15/Jan/2024:10:30:00 +0000] "GET / HTTP/1.1" 200 42`,
		`2024-01-15 10:30:00.123 ERROR [main] com.example.Service - Failed to connect`,
	}

	var wg sync.WaitGroup
	start := time.Now()

	for i := 0; i < numRequests; i++ {
		wg.Add(1)
		go func(reqID int) {
			defer wg.Done()

			lines := make([]string, linesPerRequest)
			for j := range lines {
				lines[j] = diverseLogs[(reqID+j)%len(diverseLogs)]
			}

			batch := map[string]interface{}{
				"source": fmt.Sprintf("raw-stress-%d", reqID),
				"lines":  lines,
			}

			body, _ := json.Marshal(batch)
			resp, err := http.Post(
				baseURL+"/api/logs/raw",
				"application/json",
				bytes.NewReader(body),
			)
			if err == nil {
				resp.Body.Close()
			}
		}(i)
	}

	wg.Wait()

	duration := time.Since(start)
	totalLines := numRequests * linesPerRequest
	rate := float64(totalLines) / duration.Seconds()

	t.Logf("Raw ingestion stress: %d lines in %.2f seconds (%.0f lines/sec)", totalLines, duration.Seconds(), rate)
}

func TestStressReadAfterWrite(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping stress test in short mode")
	}

	baseURL := "http://localhost:8080"

	// Write a lot of data
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			batch := map[string]interface{}{
				"source":  fmt.Sprintf("read-write-%d", id),
				"entries": generateStressEntries(100, id),
			}
			body, _ := json.Marshal(batch)
			resp, err := http.Post(baseURL+"/api/logs/ingest", "application/json", bytes.NewReader(body))
			if err == nil {
				resp.Body.Close()
			}
		}(i)
	}

	// Concurrently read stats
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 10; j++ {
				resp, err := http.Get(baseURL + "/api/stats")
				if err == nil {
					resp.Body.Close()
				}
				resp, err = http.Get(baseURL + "/api/logs?limit=50")
				if err == nil {
					resp.Body.Close()
				}
			}
		}()
	}

	wg.Wait()
	t.Log("Read-after-write stress test completed")
}

func TestStressMetrics(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping stress test in short mode")
	}

	baseURL := "http://localhost:8080"

	// Ingest some data
	for i := 0; i < 10; i++ {
		batch := map[string]interface{}{
			"source":  fmt.Sprintf("metrics-%d", i),
			"entries": generateStressEntries(10, i),
		}
		body, _ := json.Marshal(batch)
		resp, err := http.Post(baseURL+"/api/logs/ingest", "application/json", bytes.NewReader(body))
		if err == nil {
			resp.Body.Close()
		}
	}

	// Check metrics
	resp, err := http.Get(baseURL + "/api/metrics/json")
	if err != nil {
		t.Fatalf("metrics failed: %v", err)
	}

	var metrics map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&metrics)
	resp.Body.Close()

	t.Logf("Metrics: %+v", metrics)
}

func generateStressEntries(count, workerID int) []map[string]string {
	levels := []string{"INFO", "WARN", "ERROR", "DEBUG"}
	entries := make([]map[string]string, count)

	for i := 0; i < count; i++ {
		level := levels[i%len(levels)]
		entries[i] = map[string]string{
			"level":   level,
			"message": fmt.Sprintf("Worker %d - Stress entry %d - %s", workerID, i, level),
		}
	}

	return entries
}
