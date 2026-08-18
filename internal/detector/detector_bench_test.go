package detector

import (
	"fmt"
	"testing"
	"time"

	"github.com/2H-K/anohive/internal/models"
)

func BenchmarkProcessSingleEntry(b *testing.B) {
	d := NewDetector(60)
	now := time.Now()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		d.Process(&models.LogEntry{
			ID:        fmt.Sprintf("log_%d", i),
			Timestamp: now.Add(time.Duration(i) * time.Millisecond),
			Level:     models.LogLevelInfo,
			Message:   "benchmark log entry",
			Source:    "bench",
		})
	}
}

func BenchmarkProcessWithErrorSpike(b *testing.B) {
	d := NewDetector(60)
	d.SetThresholds(0.3, 3.0, 100)
	now := time.Now()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		level := models.LogLevelInfo
		if i%3 == 0 {
			level = models.LogLevelError
		}
		d.Process(&models.LogEntry{
			ID:        fmt.Sprintf("log_%d", i),
			Timestamp: now.Add(time.Duration(i) * time.Millisecond),
			Level:     level,
			Message:   fmt.Sprintf("benchmark entry %d", i),
			Source:    "bench",
		})
	}
}

func BenchmarkProcessMultipleSources(b *testing.B) {
	d := NewDetector(60)
	now := time.Now()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		d.Process(&models.LogEntry{
			ID:        fmt.Sprintf("log_%d", i),
			Timestamp: now.Add(time.Duration(i) * time.Millisecond),
			Level:     models.LogLevelInfo,
			Message:   "benchmark entry",
			Source:    fmt.Sprintf("source_%d", i%10),
		})
	}
}

func BenchmarkProcessWithCallback(b *testing.B) {
	d := NewDetector(60)
	d.SetAnomalyCallback(func(a *models.Anomaly) {})
	now := time.Now()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		d.Process(&models.LogEntry{
			ID:        fmt.Sprintf("log_%d", i),
			Timestamp: now.Add(time.Duration(i) * time.Millisecond),
			Level:     models.LogLevelError,
			Message:   fmt.Sprintf("error %d", i),
			Source:    "bench",
		})
	}
}
