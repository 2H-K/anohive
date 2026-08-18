package collector

import (
	"strings"
	"testing"
	"time"

	"github.com/2H-K/pulse/internal/models"
	"github.com/2H-K/pulse/internal/parser"
)

func TestNewCollector(t *testing.T) {
	c := NewCollector(100)
	if c == nil {
		t.Fatal("expected non-nil collector")
	}
	if cap(c.output) != 100 {
		t.Errorf("buffer size = %d, want 100", cap(c.output))
	}
}

func TestAddSource(t *testing.T) {
	c := NewCollector(100)

	reader := strings.NewReader("line1\nline2\nline3\n")
	c.AddSource("test", reader)

	sources := c.ActiveSources()
	if len(sources) != 1 {
		t.Fatalf("expected 1 source, got %d", len(sources))
	}
	if sources[0] != "test" {
		t.Errorf("source = %s, want test", sources[0])
	}
}

func TestReadLoop(t *testing.T) {
	c := NewCollector(100)

	lines := "INFO message 1\nERROR message 2\nWARN message 3\n"
	c.AddSource("test-source", strings.NewReader(lines))

	var received []*models.LogEntry
	done := make(chan struct{})

	go func() {
		for entry := range c.Output() {
			received = append(received, entry)
			if len(received) >= 3 {
				close(done)
				return
			}
		}
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for entries")
	}

	c.Close()

	if len(received) < 3 {
		t.Errorf("expected 3 entries, got %d", len(received))
	}
}

func TestParseAndClassify(t *testing.T) {
	p := parser.New()

	entry := ParseAndClassify("ERROR: something broke", "test", p)
	if entry == nil {
		t.Fatal("expected non-nil entry")
	}
	if entry.Level != models.LogLevelError {
		t.Errorf("level = %v, want ERROR", entry.Level)
	}
	if entry.Source != "test" {
		t.Errorf("source = %v, want test", entry.Source)
	}
}

func TestRemoveSource(t *testing.T) {
	c := NewCollector(100)

	reader := strings.NewReader("line1\nline2\n")
	c.AddSource("test", reader)

	if len(c.ActiveSources()) != 1 {
		t.Fatal("expected 1 source before removal")
	}

	c.RemoveSource("test")
	time.Sleep(100 * time.Millisecond)

	if len(c.ActiveSources()) != 0 {
		t.Errorf("expected 0 sources after removal, got %d", len(c.ActiveSources()))
	}
}

func TestDuplicateSource(t *testing.T) {
	c := NewCollector(100)

	reader1 := strings.NewReader("line1\n")
	reader2 := strings.NewReader("line2\n")

	c.AddSource("test", reader1)
	c.AddSource("test", reader2)

	if len(c.ActiveSources()) != 1 {
		t.Errorf("expected 1 source (duplicate ignored), got %d", len(c.ActiveSources()))
	}
}

func TestExtractKeyValuePairs(t *testing.T) {
	line := `method=GET status=200 duration=45`
	pairs := ExtractKeyValuePairs(line)

	if pairs["method"] != "GET" {
		t.Errorf("method = %v, want GET", pairs["method"])
	}
	if pairs["status"] != "200" {
		t.Errorf("status = %v, want 200", pairs["status"])
	}
	if pairs["duration"] != "45" {
		t.Errorf("duration = %v, want 45", pairs["duration"])
	}
}

func TestExtractKeyValuePairsQuoted(t *testing.T) {
	line := `name="John" age=30`
	pairs := ExtractKeyValuePairs(line)

	if pairs["name"] != "John" {
		t.Errorf("name = %v, want John", pairs["name"])
	}
	if pairs["age"] != "30" {
		t.Errorf("age = %v, want 30", pairs["age"])
	}
}

func TestHighThroughput(t *testing.T) {
	c := NewCollector(10000)

	var lines string
	for i := 0; i < 100; i++ {
		lines += "INFO test message\n"
	}
	c.AddSource("perf-test", strings.NewReader(lines))

	count := 0
	timeout := time.After(2 * time.Second)

loop:
	for {
		select {
		case _, ok := <-c.Output():
			if !ok {
				break loop
			}
			count++
			if count >= 100 {
				break loop
			}
		case <-timeout:
			break loop
		}
	}

	c.Close()

	if count < 100 {
		t.Errorf("expected 100 entries, got %d", count)
	}
}
