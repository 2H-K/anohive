package parser

import (
	"testing"

	"github.com/2H-K/pulse/internal/models"
)

func TestParseJSON(t *testing.T) {
	p := New()

	tests := []struct {
		name     string
		input    string
		wantLevel models.LogLevel
		wantMsg  string
	}{
		{
			name:     "standard json log",
			input:    `{"time":"2024-01-15T10:30:00Z","level":"ERROR","message":"connection failed","service":"api"}`,
			wantLevel: models.LogLevelError,
			wantMsg:  "connection failed",
		},
		{
			name:     "info level json",
			input:    `{"timestamp":"2024-01-15T10:30:00Z","level":"INFO","msg":"request processed"}`,
			wantLevel: models.LogLevelInfo,
			wantMsg:  "request processed",
		},
		{
			name:     "warn level json",
			input:    `{"@t":"2024-01-15T10:30:00Z","@m":"high memory usage","level":"WARN"}`,
			wantLevel: models.LogLevelWarn,
			wantMsg:  "high memory usage",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entry := p.Parse(tt.input, "test")
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

func TestParseSyslog(t *testing.T) {
	p := New()

	input := "Jan 15 10:30:00 myhost sshd[1234]: Accepted publickey for user"
	entry := p.Parse(input, "")

	if entry == nil {
		t.Fatal("expected non-nil entry")
	}
	if entry.Host != "myhost" {
		t.Errorf("host = %v, want myhost", entry.Host)
	}
	if entry.Service != "sshd" {
		t.Errorf("service = %v, want sshd", entry.Service)
	}
}

func TestParseLeveled(t *testing.T) {
	p := New()

	tests := []struct {
		input    string
		wantLevel models.LogLevel
	}{
		{"2024-01-15 10:30:00 ERROR Something broke", models.LogLevelError},
		{"2024-01-15 10:30:00 WARN  Low disk space", models.LogLevelWarn},
		{"2024-01-15 10:30:00 INFO  Server started", models.LogLevelInfo},
		{"2024-01-15 10:30:00 DEBUG Query executed", models.LogLevelDebug},
	}

	for _, tt := range tests {
		entry := p.Parse(tt.input, "test")
		if entry == nil {
			t.Fatalf("expected non-nil entry for: %s", tt.input)
		}
		if entry.Level != tt.wantLevel {
			t.Errorf("level = %v, want %v for: %s", entry.Level, tt.wantLevel, tt.input)
		}
	}
}

func TestParseNginx(t *testing.T) {
	p := New()

	input := `192.168.1.1 - - [15/Jan/2024:10:30:00 +0000] "GET /api/health HTTP/1.1" 200 42`
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
}

func TestParseFallback(t *testing.T) {
	p := New()

	tests := []struct {
		input     string
		wantLevel models.LogLevel
	}{
		{"ERROR: database connection failed", models.LogLevelError},
		{"WARN: disk space low", models.LogLevelWarn},
		{"DEBUG: entering function", models.LogLevelDebug},
		{"FATAL: out of memory", models.LogLevelFatal},
		{"Something happened", models.LogLevelInfo},
	}

	for _, tt := range tests {
		entry := p.Parse(tt.input, "test")
		if entry == nil {
			t.Fatalf("expected non-nil entry for: %s", tt.input)
		}
		if entry.Level != tt.wantLevel {
			t.Errorf("level = %v, want %v for: %s", entry.Level, tt.wantLevel, tt.input)
		}
	}
}

func TestParseEmpty(t *testing.T) {
	p := New()

	if entry := p.Parse("", "test"); entry != nil {
		t.Error("expected nil for empty input")
	}
	if entry := p.Parse("   ", "test"); entry != nil {
		t.Error("expected nil for whitespace-only input")
	}
}

func TestNormalizeLogLevel(t *testing.T) {
	tests := []struct {
		input    string
		expected models.LogLevel
	}{
		{"debug", models.LogLevelDebug},
		{"DEBUG", models.LogLevelDebug},
		{"trace", models.LogLevelDebug},
		{"info", models.LogLevelInfo},
		{"INFO", models.LogLevelInfo},
		{"notice", models.LogLevelInfo},
		{"warn", models.LogLevelWarn},
		{"WARNING", models.LogLevelWarn},
		{"error", models.LogLevelError},
		{"ERROR", models.LogLevelError},
		{"fatal", models.LogLevelFatal},
		{"critical", models.LogLevelFatal},
		{"unknown", models.LogLevelUnknown},
		{"", models.LogLevelUnknown},
	}

	for _, tt := range tests {
		got := normalizeLogLevel(tt.input)
		if got != tt.expected {
			t.Errorf("normalizeLogLevel(%q) = %v, want %v", tt.input, got, tt.expected)
		}
	}
}

func TestParseWithSource(t *testing.T) {
	p := New()

	entry := p.Parse("test message", "my-source")
	if entry == nil {
		t.Fatal("expected non-nil entry")
	}
	if entry.Source != "my-source" {
		t.Errorf("source = %v, want my-source", entry.Source)
	}
}

func TestParseJSONWithFields(t *testing.T) {
	p := New()

	input := `{"level":"INFO","message":"request","method":"/api","status":"200","duration_ms":"45"}`
	entry := p.Parse(input, "test")

	if entry == nil {
		t.Fatal("expected non-nil entry")
	}
	if entry.Fields["method"] != "/api" {
		t.Errorf("fields[method] = %v, want /api", entry.Fields["method"])
	}
	if entry.Fields["status"] != "200" {
		t.Errorf("fields[status] = %v, want 200", entry.Fields["status"])
	}
}
