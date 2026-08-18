package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/2H-K/anohive/internal/api"
	"github.com/2H-K/anohive/internal/collector"
	"github.com/2H-K/anohive/internal/config"
	"github.com/2H-K/anohive/internal/detector"
	"github.com/2H-K/anohive/internal/models"
	"github.com/2H-K/anohive/internal/storage"
)

type AppConfig struct {
	Host       string
	Port       int
	DBPath     string
	StaticDir  string
	ConfigPath string
	LogFiles   []string
	BufferSize int
}

func main() {
	appCfg := parseFlags()

	// Load configuration
	cfg, err := config.LoadConfig(appCfg.ConfigPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Override config with command-line flags
	if appCfg.Host != "" {
		cfg.Server.Host = appCfg.Host
	}
	if appCfg.Port != 0 {
		cfg.Server.Port = appCfg.Port
	}
	if appCfg.DBPath != "" {
		cfg.Storage.DBPath = appCfg.DBPath
	}
	if appCfg.StaticDir != "" {
		cfg.Server.StaticDir = appCfg.StaticDir
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		log.Fatalf("Invalid configuration: %v", err)
	}

	logger := log.New(os.Stdout, "", log.LstdFlags)
	logger.Println("Starting AnoHive server...")
	logger.Printf("Host: %s, Port: %d, DB: %s", cfg.Server.Host, cfg.Server.Port, cfg.Storage.DBPath)
	logger.Printf("Log Level: %s, Format: %s", cfg.Log.Level, cfg.Log.Format)

	// Initialize storage
	store, err := storage.NewSQLiteStore(cfg.Storage.DBPath)
	if err != nil {
		log.Fatalf("Failed to initialize storage: %v", err)
	}
	defer store.Close()

	// Initialize detector
	det := detector.NewDetector(cfg.Detector.WindowSize)
	det.SetThresholds(
		cfg.Detector.ErrorRate,
		cfg.Detector.RateMultiplier,
		cfg.Detector.BurstCount,
	)

	// Initialize collector
	coll := collector.NewCollector(appCfg.BufferSize)

	// Initialize API server with config
	apiServer := api.NewServer(store, det, cfg, appCfg.ConfigPath)

	// Set up anomaly callback
	det.SetAnomalyCallback(func(a *models.Anomaly) {
		if err := store.InsertAnomaly(a); err != nil {
			logger.Printf("Failed to store anomaly: %v", err)
		}
		apiServer.BroadcastAnomaly(a)
	})

	// Start processing collector output
	go processCollectorOutput(coll, det, apiServer, store)

	// Add log file sources
	for _, f := range appCfg.LogFiles {
		if err := coll.AddFileSource(f, f); err != nil {
			logger.Printf("Warning: could not add log file %s: %v", f, err)
		} else {
			logger.Printf("Monitoring log file: %s", f)
		}
	}

	// Register routes
	mux := http.NewServeMux()
	apiServer.RegisterRoutes(mux)

	// Serve static files or API info
	if cfg.Server.StaticDir != "" {
		if _, err := os.Stat(cfg.Server.StaticDir); err == nil {
			fs := http.FileServer(http.Dir(cfg.Server.StaticDir))
			mux.Handle("/", fs)
			logger.Printf("Serving static files from: %s", cfg.Server.StaticDir)
		}
	} else {
		mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/" {
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(map[string]interface{}{
					"name":    "AnoHive - Real-time Log Monitor",
					"version": "1.0.0",
					"endpoints": []string{
						"GET  /api/health",
						"GET  /api/health/live",
						"GET  /api/health/ready",
						"GET  /api/v1/logs",
						"POST /api/v1/logs/ingest",
						"GET  /api/v1/anomalies",
						"GET  /api/v1/stats",
						"GET  /api/v1/sources",
						"PUT  /api/v1/config/thresholds",
						"WS   /ws",
					},
				})
				return
			}
			http.NotFound(w, r)
		})
	}

	// Create HTTP server with timeouts from config
	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	server := &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  time.Duration(cfg.Server.ReadTimeout) * time.Second,
		WriteTimeout: time.Duration(cfg.Server.WriteTimeout) * time.Second,
		IdleTimeout:  time.Duration(cfg.Server.IdleTimeout) * time.Second,
	}

	// Start server in goroutine
	go func() {
		logger.Printf("AnoHive server listening on http://%s", addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	// Wait for shutdown signal
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig

	logger.Println("Shutting down AnoHive server...")

	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cfg.Server.GracefulTimeout)*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		logger.Printf("Server shutdown error: %v", err)
	}

	coll.Close()
	apiServer.Shutdown()
	logger.Println("AnoHive server stopped")
}

func processCollectorOutput(coll *collector.Collector, det *detector.Detector, apiServer *api.Server, store *storage.SQLiteStore) {
	batch := make([]*models.LogEntry, 0, 100)
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	processBatch := func() {
		if len(batch) == 0 {
			return
		}

		if err := store.InsertLogs(batch); err != nil {
			log.Printf("Failed to store batch: %v", err)
		}

		for _, entry := range batch {
			det.Process(entry)
			apiServer.BroadcastLog(entry)
		}

		batch = batch[:0]
	}

	for {
		select {
		case entry, ok := <-coll.Output():
			if !ok {
				processBatch()
				return
			}
			batch = append(batch, entry)
			if len(batch) >= 100 {
				processBatch()
			}
		case <-ticker.C:
			processBatch()
		}
	}
}

func parseFlags() AppConfig {
	host := flag.String("host", "", "Server host (overrides PULSE_HOST)")
	port := flag.Int("port", 0, "Server port (overrides PULSE_PORT)")
	dbPath := flag.String("db", "", "SQLite database path (overrides PULSE_DB_PATH)")
	staticDir := flag.String("static", "", "Static files directory")
	configPath := flag.String("config", "", "Configuration file path")
	bufferSize := flag.Int("buffer", 10000, "Collector buffer size")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "AnoHive - Real-time Log Monitor\n\n")
		fmt.Fprintf(os.Stderr, "Usage: %s [options] [log_files...]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nEnvironment Variables:\n")
		fmt.Fprintf(os.Stderr, "  PULSE_HOST              Server host\n")
		fmt.Fprintf(os.Stderr, "  PULSE_PORT              Server port\n")
		fmt.Fprintf(os.Stderr, "  PULSE_DB_PATH           Database path\n")
		fmt.Fprintf(os.Stderr, "  PULSE_RETENTION_HOURS   Log retention in hours\n")
		fmt.Fprintf(os.Stderr, "  PULSE_MAX_LOGS          Maximum number of logs\n")
		fmt.Fprintf(os.Stderr, "  PULSE_API_KEY           API key for authentication\n")
		fmt.Fprintf(os.Stderr, "  PULSE_LOG_LEVEL         Log level (debug/info/warn/error)\n")
		fmt.Fprintf(os.Stderr, "  PULSE_LOG_FORMAT        Log format (text/json)\n")
		fmt.Fprintf(os.Stderr, "  PULSE_ALLOWED_ORIGINS   Allowed CORS origins (comma-separated, * for all)\n")
		fmt.Fprintf(os.Stderr, "  PULSE_RATE_LIMIT        Rate limit per minute per IP\n")
		fmt.Fprintf(os.Stderr, "  PULSE_WS_MAX_CONNECTIONS  Max WebSocket connections\n")
		fmt.Fprintf(os.Stderr, "  PULSE_ALERT_ENABLED     Enable webhook alerts (true/false)\n")
		fmt.Fprintf(os.Stderr, "  PULSE_ALERT_WEBHOOK_URL Webhook URL for alerts\n")
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  %s -port 9090 /var/log/syslog /var/log/auth.log\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -db /tmp/anohive.db -static ./web/build\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  PULSE_LOG_FORMAT=json PULSE_LOG_LEVEL=debug %s\n", os.Args[0])
	}

	flag.Parse()

	return AppConfig{
		Host:       *host,
		Port:       *port,
		DBPath:     *dbPath,
		StaticDir:  *staticDir,
		ConfigPath: *configPath,
		LogFiles:   flag.Args(),
		BufferSize: *bufferSize,
	}
}
