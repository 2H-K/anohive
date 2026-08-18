package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/2H-K/anohive/internal/config"
	"github.com/2H-K/anohive/internal/detector"
	"github.com/2H-K/anohive/internal/models"
	"github.com/2H-K/anohive/internal/parser"
	"github.com/2H-K/anohive/internal/runtime"
	"github.com/2H-K/anohive/internal/storage"
)

// Server represents the HTTP API server
type Server struct {
	storage        *storage.SQLiteStore
	detector       *detector.Detector
	config         *config.Config
	configFilePath string
	upgrader       websocket.Upgrader
	clients        map[*websocket.Conn]bool
	broadcast      chan interface{}
	mu             sync.RWMutex
	startTime      time.Time
	metrics        *Metrics
	retention      time.Duration
	maxClients     int
	rateLimiter    *RateLimiter
	logger         *Logger
	alertManager   *AlertManager
	resourceMon    *runtime.ResourceMonitor
	logSampler     *runtime.LogSampler
	dbCircuit      *runtime.CircuitBreaker
}

// NewServer creates a new API server instance
func NewServer(store *storage.SQLiteStore, det *detector.Detector, cfg *config.Config, configFilePath ...string) *Server {
	cfgPath := ""
	if len(configFilePath) > 0 {
		cfgPath = configFilePath[0]
	}

	s := &Server{
		storage:        store,
		detector:       det,
		config:         cfg,
		configFilePath: cfgPath,
		clients:        make(map[*websocket.Conn]bool),
		broadcast:      make(chan interface{}, 1000),
		startTime:      time.Now(),
		metrics:        NewMetrics(),
		retention:      cfg.RetentionDuration(),
		maxClients:     cfg.Security.MaxWSConnections,
		rateLimiter:    NewRateLimiter(cfg.Security.RateLimitPerMinute),
		logger:         NewLogger(&cfg.Log),
		alertManager:   NewAlertManager(&cfg.Alert),
		resourceMon:    runtime.NewResourceMonitor(),
		logSampler:     runtime.NewLogSampler(1.0),
		dbCircuit:      runtime.NewCircuitBreaker(5, 3, 30*time.Second),
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin: func(r *http.Request) bool {
				origin := r.Header.Get("Origin")
				if origin == "" {
					return true
				}
				return cfg.IsOriginAllowed(origin)
			},
		},
	}

	go s.broadcastLoop()
	go s.retentionLoop()
	go s.resourceMon.Start()

	s.logger.Info("server initialized",
		"port", cfg.Server.Port,
		"retention", cfg.Storage.Retention,
		"rate_limit", cfg.Security.RateLimitPerMinute,
		"config_file", cfgPath,
	)

	return s
}

// SetRetention sets the log retention duration
func (s *Server) SetRetention(d time.Duration) {
	s.retention = d
}

// RegisterRoutes registers all HTTP routes
func (s *Server) RegisterRoutes(mux *http.ServeMux) {
	// API v1 routes (public read, protected write)
	mux.HandleFunc("/api/v1/logs", s.corsMiddleware(s.rateLimitMiddleware(s.handleLogs)))
	mux.HandleFunc("/api/v1/logs/ingest", s.corsMiddleware(s.authMiddleware(s.rateLimitMiddleware(s.handleIngest))))
	mux.HandleFunc("/api/v1/logs/raw", s.corsMiddleware(s.authMiddleware(s.rateLimitMiddleware(s.handleRawIngest))))
	mux.HandleFunc("/api/v1/logs/export", s.corsMiddleware(s.rateLimitMiddleware(s.handleExport)))
	mux.HandleFunc("/api/v1/anomalies", s.corsMiddleware(s.rateLimitMiddleware(s.handleAnomalies)))
	mux.HandleFunc("/api/v1/stats", s.corsMiddleware(s.rateLimitMiddleware(s.handleStats)))
	mux.HandleFunc("/api/v1/stats/trends", s.corsMiddleware(s.rateLimitMiddleware(s.handleTrends)))
	mux.HandleFunc("/api/v1/stats/levels", s.corsMiddleware(s.rateLimitMiddleware(s.handleLevelDistribution)))
	mux.HandleFunc("/api/v1/stats/anomalies/timeline", s.corsMiddleware(s.rateLimitMiddleware(s.handleAnomalyTimeline)))
	mux.HandleFunc("/api/v1/sources", s.corsMiddleware(s.rateLimitMiddleware(s.handleSources)))
	mux.HandleFunc("/api/v1/config/thresholds", s.corsMiddleware(s.authMiddleware(s.rateLimitMiddleware(s.handleThresholds))))
	mux.HandleFunc("/api/v1/metrics", s.corsMiddleware(s.rateLimitMiddleware(s.metrics.HandleMetrics)))
	mux.HandleFunc("/api/v1/metrics/json", s.corsMiddleware(s.rateLimitMiddleware(s.metrics.HandleMetricsJSON)))

	// WebSocket endpoint
	mux.HandleFunc("/ws", s.corsMiddleware(s.rateLimitMiddleware(s.handleWebSocket)))

	// Health check endpoints (Kubernetes probes)
	mux.HandleFunc("/api/health", s.handleHealth)
	mux.HandleFunc("/api/health/live", s.handleLiveness)
	mux.HandleFunc("/api/health/ready", s.handleReadiness)

	// Legacy API routes (backward compatibility)
	mux.HandleFunc("/api/logs", s.corsMiddleware(s.rateLimitMiddleware(s.handleLogs)))
	mux.HandleFunc("/api/logs/ingest", s.corsMiddleware(s.authMiddleware(s.rateLimitMiddleware(s.handleIngest))))
	mux.HandleFunc("/api/logs/raw", s.corsMiddleware(s.authMiddleware(s.rateLimitMiddleware(s.handleRawIngest))))
	mux.HandleFunc("/api/logs/export", s.corsMiddleware(s.rateLimitMiddleware(s.handleExport)))
	mux.HandleFunc("/api/anomalies", s.corsMiddleware(s.rateLimitMiddleware(s.handleAnomalies)))
	mux.HandleFunc("/api/stats", s.corsMiddleware(s.rateLimitMiddleware(s.handleStats)))
	mux.HandleFunc("/api/stats/trends", s.corsMiddleware(s.rateLimitMiddleware(s.handleTrends)))
	mux.HandleFunc("/api/stats/levels", s.corsMiddleware(s.rateLimitMiddleware(s.handleLevelDistribution)))
	mux.HandleFunc("/api/stats/anomalies/timeline", s.corsMiddleware(s.rateLimitMiddleware(s.handleAnomalyTimeline)))
	mux.HandleFunc("/api/sources", s.corsMiddleware(s.rateLimitMiddleware(s.handleSources)))
	mux.HandleFunc("/api/config/thresholds", s.corsMiddleware(s.authMiddleware(s.rateLimitMiddleware(s.handleThresholds))))
	mux.HandleFunc("/api/metrics", s.corsMiddleware(s.rateLimitMiddleware(s.metrics.HandleMetrics)))
	mux.HandleFunc("/api/metrics/json", s.corsMiddleware(s.rateLimitMiddleware(s.metrics.HandleMetricsJSON)))

	// Config reload endpoint
	mux.HandleFunc("/api/config/reload", s.corsMiddleware(s.authMiddleware(s.rateLimitMiddleware(s.handleConfigReload))))

	// Resource stats endpoint
	mux.HandleFunc("/api/v1/admin/resources", s.corsMiddleware(s.authMiddleware(s.rateLimitMiddleware(s.handleResourceStats))))
	mux.HandleFunc("/api/admin/resources", s.corsMiddleware(s.authMiddleware(s.rateLimitMiddleware(s.handleResourceStats))))
}

// BroadcastLog broadcasts a log entry to all connected WebSocket clients
func (s *Server) BroadcastLog(entry *models.LogEntry) {
	select {
	case s.broadcast <- entry:
	default:
		s.logger.Warn("broadcast channel full, dropping log entry")
	}
}

// BroadcastAnomaly broadcasts an anomaly to all connected WebSocket clients
func (s *Server) BroadcastAnomaly(anomaly *models.Anomaly) {
	select {
	case s.broadcast <- anomaly:
	default:
		s.logger.Warn("broadcast channel full, dropping anomaly")
	}

	// Send alert if configured
	if s.alertManager.IsEnabled() {
		s.alertManager.SendAlert(anomaly)
	}
}

// corsMiddleware adds CORS headers to responses
func (s *Server) corsMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !s.config.ShouldCORS() {
			next(w, r)
			return
		}

		origin := s.config.Security.CORSAllowedOrigin
		if origin == "*" {
			w.Header().Set("Access-Control-Allow-Origin", "*")
		} else {
			reqOrigin := r.Header.Get("Origin")
			if reqOrigin != "" && s.config.IsOriginAllowed(reqOrigin) {
				w.Header().Set("Access-Control-Allow-Origin", reqOrigin)
			}
		}

		w.Header().Set("Access-Control-Allow-Methods", s.config.Security.CORSAllowedMethods)
		w.Header().Set("Access-Control-Allow-Headers", s.config.Security.CORSAllowedHeaders)
		w.Header().Set("Access-Control-Max-Age", "86400")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		next(w, r)
	}
}

// authMiddleware validates API key for protected endpoints
func (s *Server) authMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		apiKey := r.Header.Get("X-API-Key")
		if apiKey == "" {
			apiKey = r.URL.Query().Get("api_key")
		}

		if !s.config.IsValidAPIKey(apiKey) {
			s.logger.Warn("unauthorized request", "path", r.URL.Path, "ip", getClientIP(r))
			http.Error(w, "Unauthorized: invalid or missing API key", http.StatusUnauthorized)
			return
		}

		next(w, r)
	}
}

// rateLimitMiddleware enforces rate limiting per client IP
func (s *Server) rateLimitMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		clientIP := getClientIP(r)
		if !s.rateLimiter.Allow(clientIP) {
			s.logger.Warn("rate limit exceeded", "ip", clientIP, "path", r.URL.Path)
			http.Error(w, "Too many requests", http.StatusTooManyRequests)
			return
		}
		next(w, r)
	}
}

// handleLogs handles GET /api/logs - query logs with filters
func (s *Server) handleLogs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	query := models.LogQuery{
		Source: r.URL.Query().Get("source"),
		Level:  models.LogLevel(r.URL.Query().Get("level")),
		Search: r.URL.Query().Get("search"),
	}

	if start := r.URL.Query().Get("start"); start != "" {
		if t, err := time.Parse(time.RFC3339, start); err == nil {
			query.StartTime = t
		}
	}
	if end := r.URL.Query().Get("end"); end != "" {
		if t, err := time.Parse(time.RFC3339, end); err == nil {
			query.EndTime = t
		}
	}
	if limit := r.URL.Query().Get("limit"); limit != "" {
		if l, err := strconv.Atoi(limit); err == nil {
			query.Limit = l
		}
	}
	if offset := r.URL.Query().Get("offset"); offset != "" {
		if o, err := strconv.Atoi(offset); err == nil {
			query.Offset = o
		}
	}

	logs, err := s.storage.QueryLogs(query)
	if err != nil {
		s.logger.Error("query logs failed", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	total, _ := s.storage.CountLogs(query)

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"logs":  logs,
		"total": total,
		"limit": query.Limit,
		"offset": query.Offset,
	})
}

// handleIngest handles POST /api/logs/ingest - ingest structured log entries
func (s *Server) handleIngest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	s.metrics.IncrementRequests()

	r.Body = http.MaxBytesReader(w, r.Body, s.config.Security.MaxBodySize)
	defer r.Body.Close()

	var batch models.LogBatch
	if err := json.NewDecoder(r.Body).Decode(&batch); err != nil {
		var single models.LogEntry
		r.Body = http.MaxBytesReader(w, r.Body, s.config.Security.MaxBodySize)
		if err := json.NewDecoder(r.Body).Decode(&single); err != nil {
			s.logger.Warn("invalid JSON in ingest", "error", err)
			http.Error(w, "Invalid JSON payload", http.StatusBadRequest)
			return
		}
		batch.Entries = []models.LogEntry{single}
	}

	if batch.Source == "" {
		batch.Source = "api"
	}

	now := time.Now()
	for i := range batch.Entries {
		if batch.Entries[i].ID == "" {
			batch.Entries[i].ID = generateLogID()
		}
		if batch.Entries[i].Timestamp.IsZero() {
			batch.Entries[i].Timestamp = now
		}
		if batch.Entries[i].Source == "" {
			batch.Entries[i].Source = batch.Source
		}
		if len(batch.Entries[i].Message) > 10000 {
			batch.Entries[i].Message = batch.Entries[i].Message[:10000] + "...[truncated]"
		}
	}

	// Check resource pressure for sampling
	if s.resourceMon.ShouldSample() && !s.shouldKeepLog(batch.Entries) {
		respondJSON(w, http.StatusOK, map[string]interface{}{
			"ingested": 0,
			"status":   "sampled_out",
			"message":  "logs sampled due to high load",
		})
		return
	}

	entries := make([]*models.LogEntry, len(batch.Entries))
	for i := range batch.Entries {
		entries[i] = &batch.Entries[i]
		s.detector.Process(&batch.Entries[i])
		s.BroadcastLog(&batch.Entries[i])
	}

	// Use circuit breaker for database operations
	if err := s.dbCircuit.Execute(func() error {
		return s.storage.InsertLogs(entries)
	}); err != nil {
		s.logger.Error("insert logs failed", "error", err)
		if err == runtime.ErrCircuitOpen {
			respondJSON(w, http.StatusServiceUnavailable, map[string]interface{}{
				"status": "degraded",
				"message": "service temporarily unavailable, please retry",
			})
			return
		}
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	s.metrics.AddLogs(int64(len(entries)))

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"ingested": len(batch.Entries),
		"status":   "ok",
	})
}

// shouldKeepLog determines if a log should be kept based on sampling
func (s *Server) shouldKeepLog(entries []models.LogEntry) bool {
	// Always keep error and fatal logs
	for _, entry := range entries {
		if entry.Level == models.LogLevelError ||
			entry.Level == models.LogLevelFatal {
			return true
		}
	}
	// Apply sampling for non-critical logs
	return s.logSampler.ShouldLog()
}

// handleRawIngest handles POST /api/logs/raw - ingest raw log lines
func (s *Server) handleRawIngest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	s.metrics.IncrementRequests()

	r.Body = http.MaxBytesReader(w, r.Body, s.config.Security.MaxBodySize)
	defer r.Body.Close()

	var req struct {
		Source string   `json:"source"`
		Lines  []string `json:"lines"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.logger.Warn("invalid JSON in raw ingest", "error", err)
		http.Error(w, "Invalid JSON payload", http.StatusBadRequest)
		return
	}

	if req.Source == "" {
		req.Source = "raw"
	}

	p := parser.New()
	entries := make([]*models.LogEntry, 0, len(req.Lines))

	for _, line := range req.Lines {
		entry := p.Parse(line, req.Source)
		if entry == nil {
			continue
		}
		if entry.ID == "" {
			entry.ID = generateLogID()
		}
		entries = append(entries, entry)
		s.detector.Process(entry)
		s.BroadcastLog(entry)
	}

	if len(entries) > 0 {
		if err := s.storage.InsertLogs(entries); err != nil {
			s.logger.Error("insert raw logs failed", "error", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
		s.metrics.AddLogs(int64(len(entries)))
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"ingested": len(entries),
		"status":   "ok",
	})
}

// handleAnomalies handles GET /api/anomalies - query detected anomalies
func (s *Server) handleAnomalies(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	limit := 100
	if l := r.URL.Query().Get("limit"); l != "" {
		if v, err := strconv.Atoi(l); err == nil {
			limit = v
		}
	}

	severity := r.URL.Query().Get("severity")
	anomalies, err := s.storage.QueryAnomalies(limit, severity)
	if err != nil {
		s.logger.Error("query anomalies failed", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"anomalies": anomalies,
		"count":     len(anomalies),
	})
}

// handleStats handles GET /api/stats - get system statistics
func (s *Server) handleStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	totalLogs, _ := s.storage.GetTotalLogCount()
	totalAnomalies, _ := s.storage.CountAnomalies()
	sourcesStats, _ := s.storage.GetSourceStats()

	respondJSON(w, http.StatusOK, models.SystemStats{
		Uptime:         time.Since(s.startTime).Round(time.Second).String(),
		TotalSources:   len(sourcesStats),
		TotalLogs:      totalLogs,
		TotalAnomalies: totalAnomalies,
		SourcesStats:   sourcesStats,
	})
}

// handleTrends handles GET /api/v1/stats/trends - get log trends over time
func (s *Server) handleTrends(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	hours := 24
	if h := r.URL.Query().Get("hours"); h != "" {
		if v, err := strconv.Atoi(h); err == nil && v > 0 {
			hours = v
		}
	}
	source := r.URL.Query().Get("source")

	trends, err := s.storage.GetLogTrends(hours, source)
	if err != nil {
		s.logger.Error("get trends failed", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"trends": trends,
		"hours":  hours,
	})
}

// handleLevelDistribution handles GET /api/v1/stats/levels - get level distribution
func (s *Server) handleLevelDistribution(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	source := r.URL.Query().Get("source")
	dist, err := s.storage.GetLevelDistribution(source)
	if err != nil {
		s.logger.Error("get level distribution failed", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"distribution": dist,
	})
}

// handleAnomalyTimeline handles GET /api/v1/stats/anomalies/timeline - get anomaly timeline
func (s *Server) handleAnomalyTimeline(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	hours := 24
	if h := r.URL.Query().Get("hours"); h != "" {
		if v, err := strconv.Atoi(h); err == nil && v > 0 {
			hours = v
		}
	}
	source := r.URL.Query().Get("source")

	timeline, err := s.storage.GetAnomalyTimeline(hours, source)
	if err != nil {
		s.logger.Error("get anomaly timeline failed", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"timeline": timeline,
		"hours":    hours,
	})
}

// handleExport handles GET /api/v1/logs/export - export logs as JSON download
func (s *Server) handleExport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	query := models.LogQuery{
		Source: r.URL.Query().Get("source"),
		Level:  models.LogLevel(r.URL.Query().Get("level")),
		Search: r.URL.Query().Get("search"),
		Limit:  10000,
	}

	if start := r.URL.Query().Get("start"); start != "" {
		if t, err := time.Parse(time.RFC3339, start); err == nil {
			query.StartTime = t
		}
	}
	if end := r.URL.Query().Get("end"); end != "" {
		if t, err := time.Parse(time.RFC3339, end); err == nil {
			query.EndTime = t
		}
	}

	logs, err := s.storage.QueryLogs(query)
	if err != nil {
		s.logger.Error("export logs failed", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	filename := fmt.Sprintf("anohive-logs-%s.json", time.Now().Format("20060102-150405"))
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
	json.NewEncoder(w).Encode(map[string]interface{}{
		"exported_at": time.Now().UTC(),
		"count":       len(logs),
		"logs":        logs,
	})
}

// handleSources handles GET /api/sources - get source statistics
func (s *Server) handleSources(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	stats, _ := s.storage.GetSourceStats()
	respondJSON(w, http.StatusOK, stats)
}

// handleHealth handles GET /api/health - health check endpoint
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"status":    "healthy",
		"uptime":    time.Since(s.startTime).Round(time.Second).String(),
		"timestamp": time.Now().UTC(),
	})
}

// handleLiveness handles GET /api/health/live - Kubernetes liveness probe
func (s *Server) handleLiveness(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, http.StatusOK, map[string]string{
		"status": "alive",
	})
}

// handleReadiness handles GET /api/health/ready - Kubernetes readiness probe
func (s *Server) handleReadiness(w http.ResponseWriter, r *http.Request) {
	// Check database connectivity
	if err := s.storage.Ping(); err != nil {
		respondJSON(w, http.StatusServiceUnavailable, map[string]string{
			"status": "not ready",
			"error":  "database connection failed",
		})
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{
		"status": "ready",
	})
}

// handleConfigReload handles POST /api/config/reload - reload configuration
func (s *Server) handleConfigReload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if s.configFilePath == "" {
		respondJSON(w, http.StatusOK, map[string]interface{}{
			"status":  "skipped",
			"message": "no config file configured, using environment variables",
		})
		return
	}

	newCfg, err := config.LoadConfig(s.configFilePath)
	if err != nil {
		s.logger.Error("config reload failed", "error", err)
		respondJSON(w, http.StatusInternalServerError, map[string]interface{}{
			"status": "error",
			"error":  err.Error(),
		})
		return
	}

	// Apply new configuration
	s.reloadConfig(newCfg)

	s.logger.Info("configuration reloaded", "file", s.configFilePath)
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"status":  "reloaded",
		"message": "configuration reloaded successfully",
	})
}

// reloadConfig applies new configuration to the running server
func (s *Server) reloadConfig(newCfg *config.Config) {
	s.mu.Lock()
	defer s.mu.Unlock()

	oldCfg := s.config
	s.config = newCfg

	// Update rate limiter
	s.rateLimiter = NewRateLimiter(newCfg.Security.RateLimitPerMinute)

	// Update detector thresholds
	s.detector.SetThresholds(
		newCfg.Detector.ErrorRate,
		newCfg.Detector.RateMultiplier,
		newCfg.Detector.BurstCount,
	)

	// Update alert manager
	s.alertManager = NewAlertManager(&newCfg.Alert)

	// Log changes
	if oldCfg.Security.RateLimitPerMinute != newCfg.Security.RateLimitPerMinute {
		s.logger.Info("rate limit updated",
			"old", oldCfg.Security.RateLimitPerMinute,
			"new", newCfg.Security.RateLimitPerMinute,
		)
	}
	if oldCfg.Detector.ErrorRate != newCfg.Detector.ErrorRate {
		s.logger.Info("detector thresholds updated",
			"error_rate", newCfg.Detector.ErrorRate,
			"rate_multiplier", newCfg.Detector.RateMultiplier,
			"burst", newCfg.Detector.BurstCount,
		)
	}
}

// handleResourceStats handles GET /api/admin/resources - get resource usage statistics
func (s *Server) handleResourceStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"resources":   s.resourceMon.Stats(),
		"sampler":     s.logSampler.Stats(),
		"circuit":     s.dbCircuit.State(),
	})
}

// handleThresholds handles PUT/POST /api/config/thresholds - update detection thresholds
func (s *Server) handleThresholds(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut && r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 1024*1024) // 1MB
	defer r.Body.Close()

	var cfg struct {
		ErrorRate      *float64 `json:"error_rate,omitempty"`
		RateMultiplier *float64 `json:"rate_multiplier,omitempty"`
		Burst          *int     `json:"burst,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
		s.logger.Warn("invalid JSON in thresholds", "error", err)
		http.Error(w, "Invalid JSON payload", http.StatusBadRequest)
		return
	}

	s.detector.SetThresholds(
		derefF64(cfg.ErrorRate, 0.3),
		derefF64(cfg.RateMultiplier, 3.0),
		derefInt(cfg.Burst, 100),
	)

	s.logger.Info("thresholds updated",
		"error_rate", derefF64(cfg.ErrorRate, 0.3),
		"rate_multiplier", derefF64(cfg.RateMultiplier, 3.0),
		"burst", derefInt(cfg.Burst, 100),
	)

	respondJSON(w, http.StatusOK, map[string]string{"status": "updated"})
}

// handleWebSocket handles WebSocket connections
func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	currentClients := len(s.clients)
	s.mu.RUnlock()

	if currentClients >= s.maxClients {
		s.logger.Warn("max WebSocket clients reached", "current", currentClients)
		http.Error(w, "Too many connections", http.StatusServiceUnavailable)
		return
	}

	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		s.logger.Error("WebSocket upgrade failed", "error", err)
		return
	}

	s.mu.Lock()
	s.clients[conn] = true
	clientCount := int64(len(s.clients))
	s.mu.Unlock()

	s.metrics.SetActiveClients(clientCount)
	s.logger.Info("WebSocket client connected", "total", clientCount)

	defer func() {
		s.mu.Lock()
		delete(s.clients, conn)
		clientCount := int64(len(s.clients))
		s.mu.Unlock()
		s.metrics.SetActiveClients(clientCount)
		conn.Close()
		s.logger.Info("WebSocket client disconnected", "total", clientCount)
	}()

	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			break
		}
	}
}

// broadcastLoop broadcasts messages to all connected clients
func (s *Server) broadcastLoop() {
	for msg := range s.broadcast {
		s.mu.RLock()
		clients := make([]*websocket.Conn, 0, len(s.clients))
		for c := range s.clients {
			clients = append(clients, c)
		}
		s.mu.RUnlock()

		data, err := json.Marshal(msg)
		if err != nil {
			continue
		}

		for _, conn := range clients {
			if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
				s.mu.Lock()
				delete(s.clients, conn)
				s.mu.Unlock()
				conn.Close()
			}
		}
	}
}

// retentionLoop periodically cleans up old logs
func (s *Server) retentionLoop() {
	interval := time.Duration(s.config.Storage.CleanupInterval) * time.Second
	if interval < time.Minute {
		interval = time.Hour
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for range ticker.C {
		s.runRetentionCleanup()
	}
}

// runRetentionCleanup performs log cleanup
func (s *Server) runRetentionCleanup() {
	deleted, err := s.storage.DeleteOldLogs(s.retention)
	if err != nil {
		s.logger.Error("retention cleanup error", "error", err)
	} else if deleted > 0 {
		s.logger.Info("retention cleanup completed", "logs_deleted", deleted)
	}

	deleted, err = s.storage.DeleteOldAnomalies(s.retention)
	if err != nil {
		s.logger.Error("anomaly retention cleanup error", "error", err)
	} else if deleted > 0 {
		s.logger.Info("anomaly retention cleanup completed", "anomalies_deleted", deleted)
	}

	// Enforce max logs limit
	if s.config.Storage.MaxLogs > 0 {
		count, _ := s.storage.GetTotalLogCount()
		if count > int64(s.config.Storage.MaxLogs) {
			excess := count - int64(s.config.Storage.MaxLogs)
			s.logger.Info("enforcing max logs limit", "excess", excess)
		}
	}
}

// Shutdown gracefully shuts down the server
func (s *Server) Shutdown() {
	s.logger.Info("shutting down server...")

	// Stop resource monitor
	s.resourceMon.Stop()

	// Close WebSocket clients
	s.mu.Lock()
	for conn := range s.clients {
		conn.Close()
	}
	s.clients = make(map[*websocket.Conn]bool)
	s.mu.Unlock()

	time.Sleep(100 * time.Millisecond)

	close(s.broadcast)

	s.logger.Info("server shutdown complete")
}

// Helper functions

func derefF64(p *float64, def float64) float64 {
	if p != nil {
		return *p
	}
	return def
}

func derefInt(p *int, def int) int {
	if p != nil {
		return *p
	}
	return def
}

func getClientIP(r *http.Request) string {
	xff := r.Header.Get("X-Forwarded-For")
	if xff != "" {
		parts := strings.Split(xff, ",")
		return strings.TrimSpace(parts[0])
	}
	xri := r.Header.Get("X-Real-Ip")
	if xri != "" {
		return xri
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

func generateLogID() string {
	return fmt.Sprintf("log_%d", time.Now().UnixNano())
}

func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// Logger provides structured logging
type Logger struct {
	level  string
	format string
	output string
	stdout *log.Logger
	stderr *log.Logger
}

// NewLogger creates a new logger instance
func NewLogger(cfg *config.LogConfig) *Logger {
	return &Logger{
		level:  cfg.Level,
		format: cfg.Format,
		output: cfg.Output,
		stdout: log.New(os.Stdout, "", 0),
		stderr: log.New(os.Stderr, "", 0),
	}
}

func (l *Logger) log(level string, msg string, keysAndValues ...interface{}) {
	if l.shouldLog(level) {
		if l.format == "json" {
			l.logJSON(level, msg, keysAndValues...)
		} else {
			l.logText(level, msg, keysAndValues...)
		}
	}
}

func (l *Logger) shouldLog(level string) bool {
	levels := map[string]int{
		"debug": 0,
		"info":  1,
		"warn":  2,
		"error": 3,
	}
	return levels[level] >= levels[l.level]
}

func (l *Logger) logJSON(level string, msg string, keysAndValues ...interface{}) {
	entry := map[string]interface{}{
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"level":     level,
		"message":   msg,
	}
	for i := 0; i < len(keysAndValues)-1; i += 2 {
		if key, ok := keysAndValues[i].(string); ok {
			entry[key] = keysAndValues[i+1]
		}
	}
	data, _ := json.Marshal(entry)
	l.stdout.Println(string(data))
}

func (l *Logger) logText(level string, msg string, keysAndValues ...interface{}) {
	var sb strings.Builder
	sb.WriteString(time.Now().UTC().Format(time.RFC3339))
	sb.WriteString(" [")
	sb.WriteString(strings.ToUpper(level))
	sb.WriteString("] ")
	sb.WriteString(msg)
	for i := 0; i < len(keysAndValues)-1; i += 2 {
		sb.WriteString(" ")
		sb.WriteString(fmt.Sprintf("%v=%v", keysAndValues[i], keysAndValues[i+1]))
	}
	l.stdout.Println(sb.String())
}

func (l *Logger) Debug(msg string, keysAndValues ...interface{}) { l.log("debug", msg, keysAndValues...) }
func (l *Logger) Info(msg string, keysAndValues ...interface{})  { l.log("info", msg, keysAndValues...) }
func (l *Logger) Warn(msg string, keysAndValues ...interface{})  { l.log("warn", msg, keysAndValues...) }
func (l *Logger) Error(msg string, keysAndValues ...interface{}) { l.log("error", msg, keysAndValues...) }

// AlertManager handles webhook alerts
type AlertManager struct {
	config     *config.AlertConfig
	lastAlert  map[string]time.Time
	mu         sync.Mutex
	httpClient *http.Client
}

// NewAlertManager creates a new alert manager
func NewAlertManager(cfg *config.AlertConfig) *AlertManager {
	return &AlertManager{
		config:    cfg,
		lastAlert: make(map[string]time.Time),
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// IsEnabled returns whether alerting is enabled
func (a *AlertManager) IsEnabled() bool {
	return a.config.Enabled && a.config.WebhookURL != ""
}

// SendAlert sends an alert via webhook
func (a *AlertManager) SendAlert(anomaly *models.Anomaly) {
	if !a.IsEnabled() {
		return
	}

	// Check if severity is configured for alerting
	shouldAlert := false
	for _, sev := range a.config.Severities {
		if sev == string(anomaly.Severity) {
			shouldAlert = true
			break
		}
	}
	if !shouldAlert {
		return
	}

	// Check cooldown
	typeKey := string(anomaly.Type)
	a.mu.Lock()
	last, ok := a.lastAlert[typeKey]
	cooldown := time.Duration(a.config.CooldownSeconds) * time.Second
	if ok && time.Since(last) < cooldown {
		a.mu.Unlock()
		return
	}
	a.lastAlert[typeKey] = time.Now()
	a.mu.Unlock()

	// Build alert payload
	payload := map[string]interface{}{
		"source":     "anohive",
		"type":       string(anomaly.Type),
		"severity":   string(anomaly.Severity),
		"message":    anomaly.Description,
		"timestamp":  anomaly.Timestamp,
		"source_log": anomaly.Source,
		"metadata":   anomaly.Metadata,
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return
	}

	// Send webhook asynchronously
	go func() {
		resp, err := a.httpClient.Post(a.config.WebhookURL, "application/json", strings.NewReader(string(data)))
		if err != nil {
			log.Printf("Failed to send alert: %v", err)
			return
		}
		resp.Body.Close()
	}()
}

var _ = os.Stdout // Keep os import
