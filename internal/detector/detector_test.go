package detector

import (
	"testing"
	"time"

	"github.com/2H-K/pulse/internal/models"
)

func TestNewDetector(t *testing.T) {
	d := NewDetector(60)
	if d.windowSize != 60 {
		t.Errorf("windowSize = %d, want 60", d.windowSize)
	}
	if d.errorThreshold != 0.3 {
		t.Errorf("errorThreshold = %f, want 0.3", d.errorThreshold)
	}

	d2 := NewDetector(0)
	if d2.windowSize != 60 {
		t.Errorf("default windowSize = %d, want 60", d2.windowSize)
	}
}

func TestProcessSingleEntry(t *testing.T) {
	d := NewDetector(60)

	entry := &models.LogEntry{
		ID:        "log_1",
		Timestamp: time.Now(),
		Level:     models.LogLevelInfo,
		Message:   "test message",
		Source:    "test",
	}

	anomalies := d.Process(entry)
	if len(anomalies) != 0 {
		t.Errorf("expected 0 anomalies for single info entry, got %d", len(anomalies))
	}
}

func TestErrorSpikeDetection(t *testing.T) {
	d := NewDetector(60)
	d.SetThresholds(0.3, 3.0, 100)

	now := time.Now()

	for i := 0; i < 7; i++ {
		d.Process(&models.LogEntry{
			ID:        "log_" + string(rune('a'+i)),
			Timestamp: now.Add(time.Duration(i) * time.Second),
			Level:     models.LogLevelError,
			Message:   "error " + string(rune('a'+i)),
			Source:    "test-source",
		})
	}

	for i := 7; i < 10; i++ {
		d.Process(&models.LogEntry{
			ID:        "log_" + string(rune('a'+i)),
			Timestamp: now.Add(time.Duration(i) * time.Second),
			Level:     models.LogLevelInfo,
			Message:   "info " + string(rune('a'+i)),
			Source:    "test-source",
		})
	}

	states := d.GetSourceStates()
	state, ok := states["test-source"]
	if !ok {
		t.Fatal("expected source state for test-source")
	}

	if state.TotalLogs != 10 {
		t.Errorf("total logs = %d, want 10", state.TotalLogs)
	}
	if state.ErrorLogs != 7 {
		t.Errorf("error logs = %d, want 7", state.ErrorLogs)
	}
}

func TestAnomalyCallback(t *testing.T) {
	d := NewDetector(60)
	d.SetThresholds(0.5, 3.0, 100)

	var received []*models.Anomaly
	d.SetAnomalyCallback(func(a *models.Anomaly) {
		received = append(received, a)
	})

	now := time.Now()
	for i := 0; i < 6; i++ {
		d.Process(&models.LogEntry{
			ID:        "log_err_" + string(rune('a'+i)),
			Timestamp: now.Add(time.Duration(i) * time.Second),
			Level:     models.LogLevelError,
			Message:   "critical failure " + string(rune('a'+i)),
			Source:    "callback-test",
		})
	}

	for i := 6; i < 10; i++ {
		d.Process(&models.LogEntry{
			ID:        "log_ok_" + string(rune('a'+i)),
			Timestamp: now.Add(time.Duration(i) * time.Second),
			Level:     models.LogLevelInfo,
			Message:   "ok " + string(rune('a'+i)),
			Source:    "callback-test",
		})
	}

	if len(received) == 0 {
		t.Error("expected at least one anomaly callback")
	}
}

func TestSetThresholds(t *testing.T) {
	d := NewDetector(60)
	d.SetThresholds(0.5, 5.0, 200)

	if d.errorThreshold != 0.5 {
		t.Errorf("errorThreshold = %f, want 0.5", d.errorThreshold)
	}
	if d.rateThreshold != 5.0 {
		t.Errorf("rateThreshold = %f, want 5.0", d.rateThreshold)
	}
	if d.burstThreshold != 200 {
		t.Errorf("burstThreshold = %d, want 200", d.burstThreshold)
	}
}

func TestNewErrorPattern(t *testing.T) {
	d := NewDetector(60)

	now := time.Now()
	d.Process(&models.LogEntry{
		ID:        "log_1",
		Timestamp: now,
		Level:     models.LogLevelError,
		Message:   "database timeout",
		Source:    "db-test",
	})

	states := d.GetSourceStates()
	state := states["db-test"]
	if state.ErrorPatterns["database"] != 1 {
		t.Errorf("error pattern count = %d, want 1", state.ErrorPatterns["database"])
	}
}

func TestMultipleSources(t *testing.T) {
	d := NewDetector(60)

	now := time.Now()
	d.Process(&models.LogEntry{ID: "1", Timestamp: now, Level: models.LogLevelInfo, Source: "src-a"})
	d.Process(&models.LogEntry{ID: "2", Timestamp: now, Level: models.LogLevelError, Source: "src-b"})
	d.Process(&models.LogEntry{ID: "3", Timestamp: now, Level: models.LogLevelInfo, Source: "src-a"})

	states := d.GetSourceStates()
	if len(states) != 2 {
		t.Errorf("expected 2 source states, got %d", len(states))
	}
	if states["src-a"].TotalLogs != 2 {
		t.Errorf("src-a total logs = %d, want 2", states["src-a"].TotalLogs)
	}
	if states["src-b"].TotalLogs != 1 {
		t.Errorf("src-b total logs = %d, want 1", states["src-b"].TotalLogs)
	}
}

func TestNormalizeMessage(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"connection refused", "connection"},
		{"timeout after 30s", "timeout"},
		{"error: something failed", "error:"},
		{"simple message", "simple"},
	}

	for _, tt := range tests {
		got := normalizeMessage(tt.input)
		if got != tt.expected {
			t.Errorf("normalizeMessage(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

func TestStdDev(t *testing.T) {
	tests := []struct {
		values   []float64
		expected float64
	}{
		{[]float64{1, 1, 1, 1}, 0},
		{[]float64{2, 4, 4, 4, 5, 5, 7, 9}, 2.0},
		{[]float64{}, 0},
		{[]float64{5}, 0},
	}

	for _, tt := range tests {
		got := stdDev(tt.values)
		diff := got - tt.expected
		if diff < -0.1 || diff > 0.1 {
			t.Errorf("stdDev(%v) = %f, want %f", tt.values, got, tt.expected)
		}
	}
}
