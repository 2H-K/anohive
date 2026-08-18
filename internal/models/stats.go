package models

import "time"

type SourceStats struct {
	Source       string            `json:"source"`
	TotalLogs    int64             `json:"total_logs"`
	ErrorCount   int64             `json:"error_count"`
	WarnCount    int64             `json:"warn_count"`
	LogsPerMin   float64           `json:"logs_per_min"`
	TopErrors    []ErrorCount      `json:"top_errors"`
	LastLogTime  time.Time         `json:"last_log_time"`
	LevelDistribution map[string]int64 `json:"level_distribution"`
}

type ErrorCount struct {
	Message string `json:"message"`
	Count   int64  `json:"count"`
}

type SystemStats struct {
	Uptime       string         `json:"uptime"`
	TotalSources int            `json:"total_sources"`
	TotalLogs    int64          `json:"total_logs"`
	TotalAnomalies int64        `json:"total_anomalies"`
	SourcesStats []SourceStats  `json:"sources_stats"`
}
