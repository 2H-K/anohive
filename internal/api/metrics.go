package api

import (
	"fmt"
	"net/http"
	"runtime"
	"sync/atomic"
	"time"
)

type Metrics struct {
	startTime       time.Time
	totalRequests   int64
	totalLogs       int64
	totalAnomalies  int64
	activeClients   int64
	bytesReceived   int64
	bytesSent       int64
}

func NewMetrics() *Metrics {
	return &Metrics{
		startTime: time.Now(),
	}
}

func (m *Metrics) IncrementRequests() {
	atomic.AddInt64(&m.totalRequests, 1)
}

func (m *Metrics) AddLogs(count int64) {
	atomic.AddInt64(&m.totalLogs, count)
}

func (m *Metrics) AddAnomalies(count int64) {
	atomic.AddInt64(&m.totalAnomalies, count)
}

func (m *Metrics) SetActiveClients(count int64) {
	atomic.StoreInt64(&m.activeClients, count)
}

func (m *Metrics) AddBytesReceived(bytes int64) {
	atomic.AddInt64(&m.bytesReceived, bytes)
}

func (m *Metrics) AddBytesSent(bytes int64) {
	atomic.AddInt64(&m.bytesSent, bytes)
}

func (m *Metrics) HandleMetrics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	uptime := time.Since(m.startTime).Seconds()

	w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")

	fmt.Fprintf(w, "# HELP anohive_uptime_seconds Total uptime in seconds\n")
	fmt.Fprintf(w, "# TYPE anohive_uptime_seconds counter\n")
	fmt.Fprintf(w, "anohive_uptime_seconds %.2f\n", uptime)

	fmt.Fprintf(w, "# HELP anohive_requests_total Total number of HTTP requests\n")
	fmt.Fprintf(w, "# TYPE anohive_requests_total counter\n")
	fmt.Fprintf(w, "anohive_requests_total %d\n", atomic.LoadInt64(&m.totalRequests))

	fmt.Fprintf(w, "# HELP anohive_logs_ingested_total Total number of logs ingested\n")
	fmt.Fprintf(w, "# TYPE anohive_logs_ingested_total counter\n")
	fmt.Fprintf(w, "anohive_logs_ingested_total %d\n", atomic.LoadInt64(&m.totalLogs))

	fmt.Fprintf(w, "# HELP anohive_anomalies_detected_total Total number of anomalies detected\n")
	fmt.Fprintf(w, "# TYPE anohive_anomalies_detected_total counter\n")
	fmt.Fprintf(w, "anohive_anomalies_detected_total %d\n", atomic.LoadInt64(&m.totalAnomalies))

	fmt.Fprintf(w, "# HELP anohive_active_websocket_clients Number of active WebSocket clients\n")
	fmt.Fprintf(w, "# TYPE anohive_active_websocket_clients gauge\n")
	fmt.Fprintf(w, "anohive_active_websocket_clients %d\n", atomic.LoadInt64(&m.activeClients))

	fmt.Fprintf(w, "# HELP anohive_bytes_received_total Total bytes received\n")
	fmt.Fprintf(w, "# TYPE anohive_bytes_received_total counter\n")
	fmt.Fprintf(w, "anohive_bytes_received_total %d\n", atomic.LoadInt64(&m.bytesReceived))

	fmt.Fprintf(w, "# HELP anohive_bytes_sent_total Total bytes sent\n")
	fmt.Fprintf(w, "# TYPE anohive_bytes_sent_total counter\n")
	fmt.Fprintf(w, "anohive_bytes_sent_total %d\n", atomic.LoadInt64(&m.bytesSent))

	fmt.Fprintf(w, "# HELP anohive_memory_alloc_bytes Current memory allocation in bytes\n")
	fmt.Fprintf(w, "# TYPE anohive_memory_alloc_bytes gauge\n")
	fmt.Fprintf(w, "anohive_memory_alloc_bytes %d\n", memStats.Alloc)

	fmt.Fprintf(w, "# HELP anohive_memory_sys_bytes Total memory obtained from OS\n")
	fmt.Fprintf(w, "# TYPE anohive_memory_sys_bytes gauge\n")
	fmt.Fprintf(w, "anohive_memory_sys_bytes %d\n", memStats.Sys)

	fmt.Fprintf(w, "# HELP anohive_gc_runs_total Total number of GC runs\n")
	fmt.Fprintf(w, "# TYPE anohive_gc_runs_total counter\n")
	fmt.Fprintf(w, "anohive_gc_runs_total %d\n", memStats.NumGC)

	fmt.Fprintf(w, "# HELP anohive_goroutines Current number of goroutines\n")
	fmt.Fprintf(w, "# TYPE anohive_goroutines gauge\n")
	fmt.Fprintf(w, "anohive_goroutines %d\n", runtime.NumGoroutine())
}

func (m *Metrics) HandleMetricsJSON(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{
		"uptime_seconds": %.2f,
		"requests_total": %d,
		"logs_ingested_total": %d,
		"anomalies_detected_total": %d,
		"active_websocket_clients": %d,
		"bytes_received_total": %d,
		"bytes_sent_total": %d,
		"memory_alloc_bytes": %d,
		"memory_sys_bytes": %d,
		"gc_runs_total": %d,
		"goroutines": %d
	}`,
		time.Since(m.startTime).Seconds(),
		atomic.LoadInt64(&m.totalRequests),
		atomic.LoadInt64(&m.totalLogs),
		atomic.LoadInt64(&m.totalAnomalies),
		atomic.LoadInt64(&m.activeClients),
		atomic.LoadInt64(&m.bytesReceived),
		atomic.LoadInt64(&m.bytesSent),
		memStats.Alloc,
		memStats.Sys,
		memStats.NumGC,
		runtime.NumGoroutine(),
	)
}
