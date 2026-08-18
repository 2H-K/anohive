package parser

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/2H-K/anohive/internal/models"
)

type LogParser struct {
	patterns []ParsePattern
}

type ParsePattern struct {
	Name    string
	Regex   *regexp.Regexp
	Extract func(matches []string, raw string) *models.LogEntry
}

func New() *LogParser {
	p := &LogParser{}
	p.initPatterns()
	return p
}

func (p *LogParser) initPatterns() {
	p.patterns = []ParsePattern{
		{
			Name:  "json",
			Regex: regexp.MustCompile(`^\s*\{`),
			Extract: func(matches []string, raw string) *models.LogEntry {
				return p.parseJSON(raw)
			},
		},
		{
			Name:  "docker",
			Regex: regexp.MustCompile(`^(stdout|stderr)\s+\|\s+(\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}\.\d+Z)\s+(.+)`),
			Extract: func(matches []string, raw string) *models.LogEntry {
				return p.parseDocker(matches, raw)
			},
		},
		{
			Name:  "kubernetes",
			Regex: regexp.MustCompile(`^(\w)\s+(\S+)\s+(\S+)\s+(\S+)\s+(.+)$`),
			Extract: func(matches []string, raw string) *models.LogEntry {
				return p.parseKubernetes(matches, raw)
			},
		},
		{
			Name:  "rfc5424",
			Regex: regexp.MustCompile(`^<(\d+)>(\d+)\s+(\S+)\s+(\S+)\s+(\S+)\s+(\S+)\s+(\S+)\s+-\s+(.+)`),
			Extract: func(matches []string, raw string) *models.LogEntry {
				return p.parseRFC5424(matches, raw)
			},
		},
		{
			Name:  "syslog",
			Regex: regexp.MustCompile(`^(\w{3}\s+\d{1,2}\s+\d{2}:\d{2}:\d{2})\s+(\S+)\s+(\S+):\s*(.+)`),
			Extract: func(matches []string, raw string) *models.LogEntry {
				return p.parseSyslog(matches, raw)
			},
		},
		{
			Name:  "log4j",
			Regex: regexp.MustCompile(`^(\d{4}-\d{2}-\d{2}\s+\d{2}:\d{2}:\d{2}[,\.]?\d*)\s+(DEBUG|INFO|WARN(?:ING)?|ERROR|FATAL|TRACE)\s+\[([^\]]+)\]\s+(\S+)\s+-\s+(.+)$`),
			Extract: func(matches []string, raw string) *models.LogEntry {
				return p.parseLog4j(matches, raw)
			},
		},
		{
			Name:  "leveled",
			Regex: regexp.MustCompile(`(?i)^(\d{4}[-/]\d{2}[-/]\d{2}[T ]\d{2}:\d{2}:\d{2}[\.,]?\d*\S*)\s+(DEBUG|INFO|WARN(?:ING)?|ERROR|FATAL|TRACE)\s+(.+)`),
			Extract: func(matches []string, raw string) *models.LogEntry {
				return p.parseLeveled(matches, raw)
			},
		},
		{
			Name:  "apache",
			Regex: regexp.MustCompile(`^(\S+)\s+(\S+)\s+(\S+)\s+\[(.+?)\]\s+"(.+?)"\s+(\d{3})\s+(\d+)\s+"(.*?)"\s+"(.*?)"`),
			Extract: func(matches []string, raw string) *models.LogEntry {
				return p.parseApache(matches, raw)
			},
		},
		{
			Name:  "nginx",
			Regex: regexp.MustCompile(`^(\S+)\s+-\s+(\S+)\s+\[(.+?)\]\s+"(.+?)"\s+(\d{3})\s+(\d+)`),
			Extract: func(matches []string, raw string) *models.LogEntry {
				return p.parseNginx(matches, raw)
			},
		},
	}
}

func (p *LogParser) Parse(line string, source string) *models.LogEntry {
	line = strings.TrimSpace(line)
	if line == "" {
		return nil
	}

	for _, pattern := range p.patterns {
		if pattern.Regex.MatchString(line) {
			matches := pattern.Regex.FindStringSubmatch(line)
			entry := pattern.Extract(matches, line)
			if entry != nil {
				if entry.Source == "" {
					entry.Source = source
				}
				return entry
			}
		}
	}

	return p.parseFallback(line, source)
}

func (p *LogParser) parseJSON(line string) *models.LogEntry {
	var data map[string]interface{}
	if err := json.Unmarshal([]byte(line), &data); err != nil {
		return nil
	}

	entry := &models.LogEntry{
		Timestamp: time.Now(),
		Level:     models.LogLevelInfo,
		Source:    "",
		Fields:    make(map[string]string),
		Raw:       line,
	}

	if v, ok := data["time"].(string); ok {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			entry.Timestamp = t
		} else if t, err := time.Parse("2006-01-02 15:04:05", v); err == nil {
			entry.Timestamp = t
		}
	}
	if v, ok := data["timestamp"].(string); ok {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			entry.Timestamp = t
		}
	}
	if v, ok := data["@t"].(string); ok {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			entry.Timestamp = t
		}
	}

	levelFields := []string{"level", "severity", "log_level", "lvl"}
	for _, f := range levelFields {
		if v, ok := data[f].(string); ok {
			entry.Level = normalizeLogLevel(v)
			break
		}
	}

	msgFields := []string{"message", "msg", "text", "log", "content", "@m"}
	for _, f := range msgFields {
		if v, ok := data[f].(string); ok {
			entry.Message = v
			break
		}
	}

	if v, ok := data["source"].(string); ok {
		entry.Source = v
	}
	if v, ok := data["host"].(string); ok {
		entry.Host = v
	}
	if v, ok := data["service"].(string); ok {
		entry.Service = v
	}
	if v, ok := data["logger"].(string); ok {
		entry.Service = v
	}

	for k, v := range data {
		switch k {
		case "time", "timestamp", "@t", "level", "severity", "log_level", "lvl",
			"message", "msg", "text", "log", "content", "@m", "source", "host", "service", "logger":
			continue
		}
		entry.Fields[k] = fmt.Sprintf("%v", v)
	}

	return entry
}

func (p *LogParser) parseDocker(matches []string, raw string) *models.LogEntry {
	if len(matches) < 4 {
		return nil
	}

	stream := matches[1]
	ts, err := time.Parse(time.RFC3339Nano, matches[2])
	if err != nil {
		ts = time.Now()
	}

	message := matches[3]
	level := detectLevelFromText(message)

	return &models.LogEntry{
		Timestamp: ts,
		Level:     level,
		Service:   "docker",
		Message:   message,
		Raw:       raw,
		Fields: map[string]string{
			"stream": stream,
		},
	}
}

func (p *LogParser) parseKubernetes(matches []string, raw string) *models.LogEntry {
	if len(matches) < 6 {
		return nil
	}

	levelChar := matches[1]
	ts, err := time.Parse(time.RFC3339, matches[2])
	if err != nil {
		ts = time.Now()
	}

	podName := matches[3]
	containerName := matches[4]
	message := matches[5]

	level := models.LogLevelInfo
	switch strings.ToUpper(levelChar) {
	case "E", "F":
		level = models.LogLevelError
	case "W":
		level = models.LogLevelWarn
	case "I":
		level = models.LogLevelInfo
	case "D":
		level = models.LogLevelDebug
	}

	if level == models.LogLevelInfo {
		level = detectLevelFromText(message)
	}

	return &models.LogEntry{
		Timestamp: ts,
		Level:     level,
		Service:   containerName,
		Host:      podName,
		Message:   message,
		Raw:       raw,
		Fields: map[string]string{
			"pod":       podName,
			"container": containerName,
		},
	}
}

func (p *LogParser) parseRFC5424(matches []string, raw string) *models.LogEntry {
	if len(matches) < 9 {
		return nil
	}

	priority, _ := strconv.Atoi(matches[1])
	severity := priority % 8

	ts, err := time.Parse(time.RFC3339, matches[3])
	if err != nil {
		ts = time.Now()
	}

	hostname := matches[4]
	appName := matches[5]
	message := matches[8]

	level := models.LogLevelInfo
	switch {
	case severity <= 2:
		level = models.LogLevelError
	case severity == 3:
		level = models.LogLevelError
	case severity == 4:
		level = models.LogLevelWarn
	case severity == 5, severity == 6:
		level = models.LogLevelInfo
	case severity == 7:
		level = models.LogLevelDebug
	}

	return &models.LogEntry{
		Timestamp: ts,
		Level:     level,
		Host:      hostname,
		Service:   appName,
		Message:   message,
		Raw:       raw,
		Fields: map[string]string{
			"priority":  matches[1],
			"version":   matches[2],
			"proc_id":   matches[6],
			"msg_id":    matches[7],
		},
	}
}

func (p *LogParser) parseSyslog(matches []string, raw string) *models.LogEntry {
	if len(matches) < 5 {
		return nil
	}

	ts, _ := time.Parse("Jan _2 15:04:05", matches[1])
	if ts.Year() == 0 {
		ts = time.Date(time.Now().Year(), ts.Month(), ts.Day(),
			ts.Hour(), ts.Minute(), ts.Second(), 0, time.UTC)
	}

	return &models.LogEntry{
		Timestamp: ts,
		Level:     models.LogLevelInfo,
		Host:      matches[2],
		Service:   strings.Split(matches[3], "[")[0],
		Message:   matches[4],
		Raw:       raw,
		Fields:    make(map[string]string),
	}
}

func (p *LogParser) parseLog4j(matches []string, raw string) *models.LogEntry {
	if len(matches) < 6 {
		return nil
	}

	ts, err := time.Parse("2006-01-02 15:04:05.000", matches[1])
	if err != nil {
		ts, err = time.Parse("2006-01-02 15:04:05,000", matches[1])
	}
	if err != nil {
		ts, err = time.Parse("2006-01-02 15:04:05", matches[1])
	}
	if err != nil {
		ts = time.Now()
	}

	level := normalizeLogLevel(matches[2])
	thread := matches[3]
	logger := matches[4]
	message := matches[5]

	return &models.LogEntry{
		Timestamp: ts,
		Level:     level,
		Service:   logger,
		Message:   message,
		Raw:       raw,
		Fields: map[string]string{
			"thread": thread,
			"logger": logger,
		},
	}
}

func (p *LogParser) parseLeveled(matches []string, raw string) *models.LogEntry {
	if len(matches) < 4 {
		return nil
	}

	ts, err := time.Parse("2006-01-02T15:04:05", matches[1])
	if err != nil {
		ts, err = time.Parse("2006-01-02 15:04:05", matches[1])
	}
	if err != nil {
		ts = time.Now()
	}

	level := normalizeLogLevel(matches[2])
	return &models.LogEntry{
		Timestamp: ts,
		Level:     level,
		Message:   matches[3],
		Raw:       raw,
		Fields:    make(map[string]string),
	}
}

func (p *LogParser) parseApache(matches []string, raw string) *models.LogEntry {
	if len(matches) < 10 {
		return nil
	}

	level := models.LogLevelInfo
	if len(matches[6]) > 0 {
		switch matches[6][0] {
		case '4', '5':
			level = models.LogLevelError
		case '3':
			level = models.LogLevelWarn
		}
	}

	return &models.LogEntry{
		Timestamp: time.Now(),
		Level:     level,
		Host:      matches[1],
		Message:   fmt.Sprintf("%s (status: %s, size: %s)", matches[5], matches[6], matches[7]),
		Raw:       raw,
		Fields: map[string]string{
			"time":       matches[4],
			"user":       matches[3],
			"status":     matches[6],
			"size":       matches[7],
			"referer":    matches[8],
			"user_agent": matches[9],
		},
	}
}

func (p *LogParser) parseNginx(matches []string, raw string) *models.LogEntry {
	if len(matches) < 6 {
		return nil
	}

	level := models.LogLevelInfo
	if len(matches[5]) > 0 {
		switch matches[5][0] {
		case '4', '5':
			level = models.LogLevelError
		case '3':
			level = models.LogLevelWarn
		}
	}

	return &models.LogEntry{
		Timestamp: time.Now(),
		Level:     level,
		Host:      matches[1],
		Message:   fmt.Sprintf("%s (status: %s, size: %s)", matches[4], matches[5], matches[6]),
		Raw:       raw,
		Fields: map[string]string{
			"time":   matches[3],
			"user":   matches[2],
			"status": matches[5],
			"size":   matches[6],
		},
	}
}

func (p *LogParser) parseFallback(line string, source string) *models.LogEntry {
	level := detectLevelFromText(line)

	return &models.LogEntry{
		Timestamp: time.Now(),
		Level:     level,
		Message:   line,
		Source:    source,
		Raw:       line,
		Fields:    make(map[string]string),
	}
}

func detectLevelFromText(text string) models.LogLevel {
	upper := strings.ToUpper(text)

	switch {
	case strings.Contains(upper, "FATAL"):
		return models.LogLevelFatal
	case strings.Contains(upper, "CRITICAL"):
		return models.LogLevelFatal
	case strings.Contains(upper, "ERROR") || strings.Contains(upper, "ERR "):
		return models.LogLevelError
	case strings.Contains(upper, "WARN"):
		return models.LogLevelWarn
	case strings.Contains(upper, "DEBUG") || strings.Contains(upper, "TRACE"):
		return models.LogLevelDebug
	case strings.Contains(upper, "INFO"):
		return models.LogLevelInfo
	default:
		return models.LogLevelInfo
	}
}

func normalizeLogLevel(s string) models.LogLevel {
	switch strings.ToUpper(s) {
	case "DEBUG", "TRACE", "TRC":
		return models.LogLevelDebug
	case "INFO", "INF", "NOTICE", "NOT":
		return models.LogLevelInfo
	case "WARN", "WARNING", "WRN":
		return models.LogLevelWarn
	case "ERROR", "ERR", "ERRO":
		return models.LogLevelError
	case "FATAL", "CRITICAL", "CRIT", "EMERG", "EMERGENCY":
		return models.LogLevelFatal
	default:
		return models.LogLevelUnknown
	}
}
