package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

type CLI struct {
	serverURL string
	client    *http.Client
}

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	serverURL := os.Getenv("PULSE_URL")
	if serverURL == "" {
		serverURL = "http://localhost:8080"
	}

	cli := &CLI{
		serverURL: strings.TrimRight(serverURL, "/"),
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}

	command := os.Args[1]
	args := os.Args[2:]

	switch command {
	case "status", "health":
		cli.cmdStatus(args)
	case "logs":
		cli.cmdLogs(args)
	case "ingest":
		cli.cmdIngest(args)
	case "anomalies":
		cli.cmdAnomalies(args)
	case "stats":
		cli.cmdStats(args)
	case "sources":
		cli.cmdSources(args)
	case "stream":
		cli.cmdStream(args)
	case "config":
		cli.cmdConfig(args)
	case "backup":
		cli.cmdBackup(args)
	case "restore":
		cli.cmdRestore(args)
	case "help", "-h", "--help":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", command)
		printUsage()
		os.Exit(1)
	}
}

func (c *CLI) cmdStatus(args []string) {
	var serverURL string
	fs := flag.NewFlagSet("status", flag.ContinueOnError)
	fs.StringVar(&serverURL, "server", "", "")
	fs.Parse(args)

	url := c.serverURL
	if serverURL != "" {
		url = serverURL
	}

	resp, err := c.client.Get(url + "/api/health")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	fmt.Printf("AnoHive Server Status\n")
	fmt.Printf("==================\n")
	fmt.Printf("Status:    %s\n", result["status"])
	fmt.Printf("Uptime:    %s\n", result["uptime"])
	fmt.Printf("Timestamp: %s\n", result["timestamp"])
}

func (c *CLI) cmdLogs(args []string) {
	var (
		source string
		level  string
		search string
		limit  int
		offset int
	)

	fs := flag.NewFlagSet("logs", flag.ContinueOnError)
	fs.StringVar(&source, "source", "", "Filter by source")
	fs.StringVar(&level, "level", "", "Filter by level (DEBUG/INFO/WARN/ERROR/FATAL)")
	fs.StringVar(&search, "search", "", "Search in log messages")
	fs.IntVar(&limit, "limit", 50, "Number of results")
	fs.IntVar(&offset, "offset", 0, "Offset for pagination")
	fs.Parse(args)

	params := fmt.Sprintf("?limit=%d&offset=%d", limit, offset)
	if source != "" {
		params += "&source=" + source
	}
	if level != "" {
		params += "&level=" + level
	}
	if search != "" {
		params += "&search=" + search
	}

	resp, err := c.client.Get(c.serverURL + "/api/logs" + params)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		fmt.Fprintf(os.Stderr, "Error decoding response: %v\n", err)
		os.Exit(1)
	}

	logs, _ := result["logs"].([]interface{})
	total, _ := result["total"].(float64)

	fmt.Printf("Log Entries (showing %d of %.0f)\n", len(logs), total)
	fmt.Println(strings.Repeat("=", 80))

	for _, l := range logs {
		log, _ := l.(map[string]interface{})
		ts, _ := log["timestamp"].(string)
		level, _ := log["level"].(string)
		msg, _ := log["message"].(string)
		src, _ := log["source"].(string)

		levelColor := levelColorCode(level)
		reset := "\033[0m"
		fmt.Printf("%s [%s%s%s] %s: %s\n", ts[:min(len(ts), 19)], levelColor, level, reset, src, msg)
	}
}

func (c *CLI) cmdIngest(args []string) {
	var (
		source string
		level  string
		file   string
	)

	fs := flag.NewFlagSet("ingest", flag.ContinueOnError)
	fs.StringVar(&source, "source", "cli", "Log source name")
	fs.StringVar(&level, "level", "INFO", "Log level")
	fs.StringVar(&file, "file", "", "Read log from file (- for stdin)")
	fs.Parse(args)

	var entries []map[string]interface{}

	if file != "" {
		var reader io.Reader
		if file == "-" {
			reader = os.Stdin
		} else {
			f, err := os.Open(file)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
			defer f.Close()
			reader = f
		}

		buf := new(bytes.Buffer)
		buf.ReadFrom(reader)
		for _, line := range strings.Split(buf.String(), "\n") {
			if strings.TrimSpace(line) == "" {
				continue
			}
			entries = append(entries, map[string]interface{}{
				"message": line,
				"level":   level,
				"source":  source,
			})
		}
	} else {
		fs.Args()
		if len(fs.Args()) == 0 {
			fmt.Fprintln(os.Stderr, "Usage: anohive ingest [options] <message>")
			fmt.Fprintln(os.Stderr, "   or: anohive ingest [options] -file <path>")
			os.Exit(1)
		}
		entries = append(entries, map[string]interface{}{
			"message": strings.Join(fs.Args(), " "),
			"level":   level,
			"source":  source,
		})
	}

	batch := map[string]interface{}{
		"source":  source,
		"entries": entries,
	}

	body, _ := json.Marshal(batch)
	resp, err := c.client.Post(c.serverURL+"/api/logs/ingest", "application/json", bytes.NewReader(body))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	fmt.Printf("Ingested %v entries\n", result["ingested"])
}

func (c *CLI) cmdAnomalies(args []string) {
	var (
		limit    int
		severity string
	)

	fs := flag.NewFlagSet("anomalies", flag.ContinueOnError)
	fs.IntVar(&limit, "limit", 50, "Number of results")
	fs.StringVar(&severity, "severity", "", "Filter by severity")
	fs.Parse(args)

	params := fmt.Sprintf("?limit=%d", limit)
	if severity != "" {
		params += "&severity=" + severity
	}

	resp, err := c.client.Get(c.serverURL + "/api/anomalies" + params)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	anomalies, _ := result["anomalies"].([]interface{})
	fmt.Printf("Anomalies (total: %v)\n", result["count"])
	fmt.Println(strings.Repeat("=", 80))

	for _, a := range anomalies {
		anom, _ := a.(map[string]interface{})
		ts, _ := anom["timestamp"].(string)
		sev, _ := anom["severity"].(string)
		typ, _ := anom["type"].(string)
		desc, _ := anom["description"].(string)

		sevColor := severityColorCode(sev)
		reset := "\033[0m"
		fmt.Printf("%s [%s%s%s] %s: %s\n", ts[:min(len(ts), 19)], sevColor, sev, reset, typ, desc)
	}
}

func (c *CLI) cmdStats(args []string) {
	resp, err := c.client.Get(c.serverURL + "/api/stats")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	var stats map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&stats)

	fmt.Printf("AnoHive Statistics\n")
	fmt.Println(strings.Repeat("=", 40))
	fmt.Printf("Uptime:          %s\n", stats["uptime"])
	fmt.Printf("Total Sources:   %.0f\n", stats["total_sources"])
	fmt.Printf("Total Logs:      %.0f\n", stats["total_logs"])
	fmt.Printf("Total Anomalies: %.0f\n", stats["total_anomalies"])

	if sources, ok := stats["sources_stats"].([]interface{}); ok && len(sources) > 0 {
		fmt.Println("\nSource Details:")
		fmt.Println(strings.Repeat("-", 40))
		for _, s := range sources {
			src, _ := s.(map[string]interface{})
			fmt.Printf("  %-20s logs: %.0f, errors: %.0f\n",
				src["source"], src["total_logs"], src["error_count"])
		}
	}
}

func (c *CLI) cmdSources(args []string) {
	resp, err := c.client.Get(c.serverURL + "/api/sources")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	var sources []map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&sources)

	fmt.Printf("Active Sources (%d)\n", len(sources))
	fmt.Println(strings.Repeat("=", 60))
	fmt.Printf("%-20s %10s %10s %10s\n", "Source", "Logs", "Errors", "Warns")
	fmt.Println(strings.Repeat("-", 60))

	for _, s := range sources {
		fmt.Printf("%-20s %10.0f %10.0f %10.0f\n",
			s["source"], s["total_logs"], s["error_count"], s["warn_count"])
	}
}

func (c *CLI) cmdStream(args []string) {
	fmt.Println("Streaming real-time logs (Ctrl+C to stop)...")
	fmt.Println(strings.Repeat("=", 80))

	// For now, just poll the logs endpoint
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	seen := make(map[string]bool)

	for range ticker.C {
		resp, err := c.client.Get(c.serverURL + "/api/logs?limit=10")
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			continue
		}

		var result map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&result)

		logs, _ := result["logs"].([]interface{})
		for _, l := range logs {
			log, _ := l.(map[string]interface{})
			id, _ := log["id"].(string)
			if seen[id] {
				continue
			}
			seen[id] = true

			ts, _ := log["timestamp"].(string)
			level, _ := log["level"].(string)
			msg, _ := log["message"].(string)

			levelColor := levelColorCode(level)
			reset := "\033[0m"
			fmt.Printf("%s [%s%s%s] %s\n", ts[:min(len(ts), 19)], levelColor, level, reset, msg)
		}

		if len(seen) > 1000 {
			seen = make(map[string]bool)
		}
	}
}

func (c *CLI) cmdConfig(args []string) {
	if len(args) < 2 {
		fmt.Println("Usage: anohive config <key> <value>")
		fmt.Println("  error_rate <float>   - Error rate threshold (0.0-1.0)")
		fmt.Println("  rate_multiplier <float> - Rate change multiplier")
		fmt.Println("  burst <int>          - Burst detection threshold")
		return
	}

	key := args[0]
	value := args[1]

	var config map[string]interface{}
	switch key {
	case "error_rate":
		var v float64
		fmt.Sscanf(value, "%f", &v)
		config = map[string]interface{}{"error_rate": v}
	case "rate_multiplier":
		var v float64
		fmt.Sscanf(value, "%f", &v)
		config = map[string]interface{}{"rate_multiplier": v}
	case "burst":
		var v int
		fmt.Sscanf(value, "%d", &v)
		config = map[string]interface{}{"burst": v}
	default:
		fmt.Fprintf(os.Stderr, "Unknown config key: %s\n", key)
		os.Exit(1)
	}

	body, _ := json.Marshal(config)
	req, _ := http.NewRequest("PUT", c.serverURL+"/api/config/thresholds", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	fmt.Println("Configuration updated")
}

func (c *CLI) cmdBackup(args []string) {
	var (
		dbPath     string
		outputPath string
	)

	fs := flag.NewFlagSet("backup", flag.ContinueOnError)
	fs.StringVar(&dbPath, "db", "", "Database path (default: from server)")
	fs.StringVar(&outputPath, "output", "", "Output backup file path (required)")
	fs.Parse(args)

	if outputPath == "" {
		fmt.Fprintln(os.Stderr, "Usage: anohive backup -output <path> [-db <path>]")
		fmt.Fprintln(os.Stderr, "  -output string   Output backup file path (required)")
		fmt.Fprintln(os.Stderr, "  -db string       Database path (if not using server)")
		os.Exit(1)
	}

	// If db path is provided, backup directly
	if dbPath != "" {
		err := backupDatabase(dbPath, outputPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Backup failed: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Database backed up to: %s\n", outputPath)
		return
	}

	// Otherwise, use server API (if available)
	resp, err := c.client.Get(c.serverURL + "/api/health")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error connecting to server: %v\n", err)
		fmt.Fprintln(os.Stderr, "Use -db flag to backup local database")
		os.Exit(1)
	}
	resp.Body.Close()

	fmt.Println("Server-based backup not yet implemented")
	fmt.Println("Use: anohive backup -db <path> -output <backup_path>")
}

func (c *CLI) cmdRestore(args []string) {
	var (
		dbPath     string
		backupPath string
	)

	fs := flag.NewFlagSet("restore", flag.ContinueOnError)
	fs.StringVar(&dbPath, "db", "", "Database path (default: from server)")
	fs.StringVar(&backupPath, "input", "", "Backup file path (required)")
	fs.Parse(args)

	if backupPath == "" {
		fmt.Fprintln(os.Stderr, "Usage: anohive restore -input <path> [-db <path>]")
		fmt.Fprintln(os.Stderr, "  -input string    Backup file path (required)")
		fmt.Fprintln(os.Stderr, "  -db string       Database path (if not using server)")
		os.Exit(1)
	}

	// If db path is provided, restore directly
	if dbPath != "" {
		err := restoreDatabase(backupPath, dbPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Restore failed: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Database restored from: %s\n", backupPath)
		return
	}

	fmt.Println("Server-based restore not yet implemented")
	fmt.Println("Use: anohive restore -db <path> -input <backup_path>")
}

func backupDatabase(dbPath, backupPath string) error {
	// Open source database
	srcDB, err := sql.Open("sqlite3", dbPath+"?_journal=WAL&_busy_timeout=10000")
	if err != nil {
		return fmt.Errorf("open source database: %w", err)
	}
	defer srcDB.Close()

	// Use SQLite backup API
	_, err = srcDB.Exec(fmt.Sprintf("VACUUM INTO '%s'", backupPath))
	if err != nil {
		return fmt.Errorf("backup database: %w", err)
	}

	return nil
}

func restoreDatabase(backupPath, dbPath string) error {
	// Copy backup file to database path
	input, err := os.ReadFile(backupPath)
	if err != nil {
		return fmt.Errorf("read backup file: %w", err)
	}

	err = os.WriteFile(dbPath, input, 0644)
	if err != nil {
		return fmt.Errorf("write database file: %w", err)
	}

	return nil
}

func printUsage() {
	fmt.Print(`AnoHive CLI - Real-time Log Monitor

Usage: anohive <command> [options]

Commands:
   health              Check server status
   logs                Query log entries
   ingest              Ingest log entries
   anomalies           List detected anomalies
   stats               Show system statistics
   sources             Show active sources
   stream              Stream real-time logs
   config              Update configuration
   backup              Backup database
   restore             Restore database from backup

Environment Variables:
   PULSE_URL           Server URL (default: http://localhost:8080)

Examples:
   anohive health
   anohive logs --level ERROR --limit 10
   anohive ingest --level ERROR "Something went wrong"
   anohive anomalies --severity CRITICAL
   anohive stream
   anohive backup -db /var/lib/anohive/data/anohive.db -output /tmp/anohive-backup.db
   anohive restore -db /var/lib/anohive/data/anohive.db -input /tmp/anohive-backup.db`)
}

func levelColorCode(level string) string {
	switch level {
	case "DEBUG", "TRACE":
		return "\033[36m"
	case "INFO":
		return "\033[32m"
	case "WARN", "WARNING":
		return "\033[33m"
	case "ERROR":
		return "\033[31m"
	case "FATAL", "CRITICAL":
		return "\033[35m"
	default:
		return "\033[0m"
	}
}

func severityColorCode(sev string) string {
	switch sev {
	case "LOW":
		return "\033[36m"
	case "MEDIUM":
		return "\033[33m"
	case "HIGH":
		return "\033[31m"
	case "CRITICAL":
		return "\033[35m"
	default:
		return "\033[0m"
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
