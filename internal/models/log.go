package models

import "time"

type LogLevel string

const (
	LogLevelDebug    LogLevel = "DEBUG"
	LogLevelInfo     LogLevel = "INFO"
	LogLevelWarn     LogLevel = "WARN"
	LogLevelError    LogLevel = "ERROR"
	LogLevelFatal    LogLevel = "FATAL"
	LogLevelUnknown  LogLevel = "UNKNOWN"
)

type LogEntry struct {
	ID        string            `json:"id"`
	Timestamp time.Time         `json:"timestamp"`
	Level     LogLevel          `json:"level"`
	Message   string            `json:"message"`
	Source    string            `json:"source"`
	Host      string            `json:"host"`
	Service   string            `json:"service"`
	Fields    map[string]string `json:"fields,omitempty"`
	Raw       string            `json:"raw,omitempty"`
}

type Anomaly struct {
	ID          string            `json:"id"`
	Timestamp   time.Time         `json:"timestamp"`
	Type        AnomalyType       `json:"type"`
	Severity    AnomalySeverity   `json:"severity"`
	Description string            `json:"description"`
	Source      string            `json:"source"`
	LogEntryID  string            `json:"log_entry_id,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

type AnomalyType string

const (
	AnomalySpike        AnomalyType = "ERROR_SPIKE"
	AnomalyPattern      AnomalyType = "PATTERN_DETECTED"
	AnomalyRateChange   AnomalyType = "RATE_CHANGE"
	AnomalyNewError     AnomalyType = "NEW_ERROR_TYPE"
	AnomalyBurst        AnomalyType = "LOG_BURST"
)

type AnomalySeverity string

const (
	SeverityLow     AnomalySeverity = "LOW"
	SeverityMedium  AnomalySeverity = "MEDIUM"
	SeverityHigh    AnomalySeverity = "HIGH"
	SeverityCritical AnomalySeverity = "CRITICAL"
)

type LogQuery struct {
	Source    string    `json:"source,omitempty"`
	Level     LogLevel  `json:"level,omitempty"`
	Search    string    `json:"search,omitempty"`
	StartTime time.Time `json:"start_time,omitempty"`
	EndTime   time.Time `json:"end_time,omitempty"`
	Limit     int       `json:"limit,omitempty"`
	Offset    int       `json:"offset,omitempty"`
}

type LogBatch struct {
	Entries []LogEntry `json:"entries"`
	Source  string     `json:"source"`
}
