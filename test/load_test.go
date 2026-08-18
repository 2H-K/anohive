package test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"testing"
	"time"
)

func TestHighThroughputIngestion(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping load test in short mode")
	}

	baseURL := "http://localhost:8080"
	totalEntries := 1000
	concurrency := 10
	entriesPerRequest := totalEntries / concurrency

	var wg sync.WaitGroup
	errors := make(chan error, concurrency)

	start := time.Now()

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			batch := map[string]interface{}{
				"source": fmt.Sprintf("load-test-%d", workerID),
				"entries": generateEntries(entriesPerRequest, workerID),
			}

			body, _ := json.Marshal(batch)
			resp, err := http.Post(
				baseURL+"/api/logs/ingest",
				"application/json",
				bytes.NewReader(body),
			)
			if err != nil {
				errors <- fmt.Errorf("worker %d: %v", workerID, err)
				return
			}
			resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				errors <- fmt.Errorf("worker %d: status %d", workerID, resp.StatusCode)
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	duration := time.Since(start)

	for err := range errors {
		t.Error(err)
	}

	elapsed := duration.Seconds()
	rate := float64(totalEntries) / elapsed
	t.Logf("Ingested %d entries in %.2f seconds (%.0f entries/sec)", totalEntries, elapsed, rate)

	if rate < 100 {
		t.Errorf("ingestion rate %.0f entries/sec is below threshold", rate)
	}
}

func TestConcurrentMixedOperations(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping load test in short mode")
	}

	baseURL := "http://localhost:8080"
	duration := 5 * time.Second

	var wg sync.WaitGroup
	stop := make(chan struct{})

	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-stop:
				return
			default:
				batch := map[string]interface{}{
					"source":  "mixed-test",
					"entries": generateEntries(10, 0),
				}
				body, _ := json.Marshal(batch)
				resp, err := http.Post(baseURL+"/api/logs/ingest", "application/json", bytes.NewReader(body))
				if err == nil {
					resp.Body.Close()
				}
			}
		}
	}()

	readers := 5
	for i := 0; i < readers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-stop:
					return
				default:
					resp, err := http.Get(baseURL + "/api/logs?limit=20")
					if err == nil {
						resp.Body.Close()
					}
					resp, err = http.Get(baseURL + "/api/stats")
					if err == nil {
						resp.Body.Close()
					}
				}
			}
		}()
	}

	time.Sleep(duration)
	close(stop)
	wg.Wait()
	t.Log("Concurrent mixed operations completed without deadlock")
}

func generateEntries(count, workerID int) []map[string]string {
	levels := []string{"INFO", "WARN", "ERROR", "DEBUG"}
	entries := make([]map[string]string, count)

	for i := 0; i < count; i++ {
		level := levels[i%len(levels)]
		entries[i] = map[string]string{
			"level":   level,
			"message": fmt.Sprintf("Worker %d - Log entry %d - %s message", workerID, i, level),
		}
	}

	return entries
}
