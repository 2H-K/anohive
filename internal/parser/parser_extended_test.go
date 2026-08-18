package parser

import (
	"testing"

	"github.com/2H-K/pulse/internal/models"
)

func TestParseDocker(t *testing.T) {
	p := New()

	tests := []struct {
		name      string
		input     string
		wantLevel models.LogLevel
		wantMsg   string
	}{
		{
			name:      "docker stdout",
			input:    "stdout | 2024-01-15T10:30:00.123456789Z Server started on port 8080",
			wantLevel: models.LogLevelInfo,
			wantMsg:   "Server started on port 8080",
		},
		{
			name:      "docker stderr error",
			input:    "stderr | 2024-01-15T10:30:00.123456789Z ERROR: Connection refused",
			wantLevel: models.LogLevelError,
			wantMsg:   "ERROR: Connection refused",
		},
		{
			name:      "docker stderr warning",
			input:    "stderr | 2024-01-15T10:30:00.123456789Z WARN: High memory usage",
			wantLevel: models.LogLevelWarn,
			wantMsg:   "WARN: High memory usage",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entry := p.Parse(tt.input, "docker-test")
			if entry == nil {
				t.Fatal("expected non-nil entry")
			}
			if entry.Level != tt.wantLevel {
				t.Errorf("level = %v, want %v", entry.Level, tt.wantLevel)
			}
			if entry.Message != tt.wantMsg {
				t.Errorf("message = %v, want %v", entry.Message, tt.wantMsg)
			}
		})
	}
}

func TestParseKubernetes(t *testing.T) {
	p := New()

	tests := []struct {
		name      string
		input     string
		wantLevel models.LogLevel
	}{
		{
			name:      "k8s error",
			input:    `E 2024-01-15T10:30:00Z my-pod my-container Error: pod crashed`,
			wantLevel: models.LogLevelError,
		},
		{
			name:      "k8s warn",
			input:    `W 2024-01-15T10:30:00Z my-pod my-container High CPU usage`,
			wantLevel: models.LogLevelWarn,
		},
		{
			name:      "k8s info",
			input:    `I 2024-01-15T10:30:00Z my-pod my-container Container started`,
			wantLevel: models.LogLevelInfo,
		},
		{
			name:      "k8s debug",
			input:    `D 2024-01-15T10:30:00Z my-pod my-container Debugging request flow`,
			wantLevel: models.LogLevelDebug,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entry := p.Parse(tt.input, "k8s-test")
			if entry == nil {
				t.Fatal("expected non-nil entry")
			}
			if entry.Level != tt.wantLevel {
				t.Errorf("level = %v, want %v", entry.Level, tt.wantLevel)
			}
		})
	}
}

func TestParseRFC5424(t *testing.T) {
	p := New()

	input := `<134>1 2024-01-15T10:30:00.000Z myhost appname 1234 ID47 - User login successful`
	entry := p.Parse(input, "")

	if entry == nil {
		t.Fatal("expected non-nil entry")
	}
	if entry.Host != "myhost" {
		t.Errorf("host = %v, want myhost", entry.Host)
	}
	if entry.Service != "appname" {
		t.Errorf("service = %v, want appname", entry.Service)
	}
	if entry.Message != "User login successful" {
		t.Errorf("message = %v, want 'User login successful'", entry.Message)
	}
}

func TestParseLog4j(t *testing.T) {
	p := New()

	tests := []struct {
		name      string
		input     string
		wantLevel models.LogLevel
		wantMsg   string
	}{
		{
			name:      "log4j error",
			input:    "2024-01-15 10:30:00.123 ERROR [main] com.example.Service - Failed to connect to database",
			wantLevel: models.LogLevelError,
			wantMsg:   "Failed to connect to database",
		},
		{
			name:      "log4j info",
			input:    "2024-01-15 10:30:00,456 INFO [http-worker-1] com.example.Controller - Request processed",
			wantLevel: models.LogLevelInfo,
			wantMsg:   "Request processed",
		},
		{
			name:      "log4j warn",
			input:    "2024-01-15 10:30:00 WARN [scheduler] com.example.Cache - Cache miss rate high",
			wantLevel: models.LogLevelWarn,
			wantMsg:   "Cache miss rate high",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entry := p.Parse(tt.input, "log4j-test")
			if entry == nil {
				t.Fatal("expected non-nil entry")
			}
			if entry.Level != tt.wantLevel {
				t.Errorf("level = %v, want %v", entry.Level, tt.wantLevel)
			}
			if entry.Message != tt.wantMsg {
				t.Errorf("message = %v, want %v", entry.Message, tt.wantMsg)
			}
		})
	}
}

func TestParseApache(t *testing.T) {
	p := New()

	input := `192.168.1.1 - frank [15/Jan/2024:10:30:00 +0000] "GET /index.html HTTP/1.1" 200 1234 "http://example.com" "Mozilla/5.0"`
	entry := p.Parse(input, "")

	if entry == nil {
		t.Fatal("expected non-nil entry")
	}
	if entry.Host != "192.168.1.1" {
		t.Errorf("host = %v, want 192.168.1.1", entry.Host)
	}
	if entry.Level != models.LogLevelInfo {
		t.Errorf("level = %v, want INFO", entry.Level)
	}
	if entry.Fields["referer"] != "http://example.com" {
		t.Errorf("referer = %v, want http://example.com", entry.Fields["referer"])
	}
	if entry.Fields["user_agent"] != "Mozilla/5.0" {
		t.Errorf("user_agent = %v, want Mozilla/5.0", entry.Fields["user_agent"])
	}
}

func TestDetectLevelFromText(t *testing.T) {
	tests := []struct {
		input    string
		expected models.LogLevel
	}{
		{"ERROR: database failed", models.LogLevelError},
		{"WARN: disk space low", models.LogLevelWarn},
		{"INFO: operation complete", models.LogLevelInfo},
		{"DEBUG: entering function", models.LogLevelDebug},
		{"FATAL: out of memory", models.LogLevelFatal},
		{"CRITICAL: system failure", models.LogLevelFatal},
		{"Something happened", models.LogLevelInfo},
	}

	for _, tt := range tests {
		got := detectLevelFromText(tt.input)
		if got != tt.expected {
			t.Errorf("detectLevelFromText(%q) = %v, want %v", tt.input, got, tt.expected)
		}
	}
}

func TestParseFormatPriority(t *testing.T) {
	p := New()

	jsonLog := `{"level":"ERROR","message":"json log"}`
	entry := p.Parse(jsonLog, "test")
	if entry == nil || entry.Level != models.LogLevelError {
		t.Error("JSON format should be detected first for JSON input")
	}

	dockerLog := "stdout | 2024-01-15T10:30:00.123456789Z test message"
	entry = p.Parse(dockerLog, "test")
	if entry == nil || entry.Fields["stream"] != "stdout" {
		t.Error("Docker format should be detected for Docker-style logs")
	}

	k8sLog := `E 2024-01-15T10:30:00Z pod container Error occurred`
	entry = p.Parse(k8sLog, "test")
	if entry == nil || entry.Level != models.LogLevelError {
		t.Error("Kubernetes format should be detected for K8s-style logs")
	}
}

func TestParseEdgeCases(t *testing.T) {
	p := New()

	if entry := p.Parse("   ", "test"); entry != nil {
		t.Error("expected nil for whitespace-only input")
	}

	if entry := p.Parse("\n\n", "test"); entry != nil {
		t.Error("expected nil for newline-only input")
	}

	if entry := p.Parse("", "test"); entry != nil {
		t.Error("expected nil for empty input")
	}
}
