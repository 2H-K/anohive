package storage

import (
	"os"
	"testing"
	"time"

	"github.com/2H-K/pulse/internal/models"
)

func setupTestDB(t *testing.T) (*SQLiteStore, func()) {
	tmpFile, err := os.CreateTemp("", "pulse_test_*.db")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	tmpFile.Close()

	store, err := NewSQLiteStore(tmpFile.Name())
	if err != nil {
		os.Remove(tmpFile.Name())
		t.Fatalf("failed to create store: %v", err)
	}

	cleanup := func() {
		store.Close()
		os.Remove(tmpFile.Name())
	}

	return store, cleanup
}

func TestInsertAndQueryLog(t *testing.T) {
	store, cleanup := setupTestDB(t)
	defer cleanup()

	entry := &models.LogEntry{
		ID:        "log_test_1",
		Timestamp: time.Now(),
		Level:     models.LogLevelError,
		Message:   "test error message",
		Source:    "test",
		Host:      "localhost",
		Fields:    map[string]string{"key": "value"},
	}

	if err := store.InsertLog(entry); err != nil {
		t.Fatalf("insert log failed: %v", err)
	}

	logs, err := store.QueryLogs(models.LogQuery{Source: "test"})
	if err != nil {
		t.Fatalf("query logs failed: %v", err)
	}

	if len(logs) != 1 {
		t.Fatalf("expected 1 log, got %d", len(logs))
	}

	if logs[0].Message != "test error message" {
		t.Errorf("message = %v, want test error message", logs[0].Message)
	}
	if logs[0].Level != models.LogLevelError {
		t.Errorf("level = %v, want ERROR", logs[0].Level)
	}
}

func TestInsertBatch(t *testing.T) {
	store, cleanup := setupTestDB(t)
	defer cleanup()

	entries := []*models.LogEntry{
		{ID: "1", Timestamp: time.Now(), Level: models.LogLevelInfo, Source: "test"},
		{ID: "2", Timestamp: time.Now(), Level: models.LogLevelError, Source: "test"},
		{ID: "3", Timestamp: time.Now(), Level: models.LogLevelWarn, Source: "test"},
	}

	if err := store.InsertLogs(entries); err != nil {
		t.Fatalf("insert batch failed: %v", err)
	}

	total, err := store.CountLogs(models.LogQuery{Source: "test"})
	if err != nil {
		t.Fatalf("count logs failed: %v", err)
	}
	if total != 3 {
		t.Errorf("total = %d, want 3", total)
	}
}

func TestQueryWithLevelFilter(t *testing.T) {
	store, cleanup := setupTestDB(t)
	defer cleanup()

	entries := []*models.LogEntry{
		{ID: "1", Timestamp: time.Now(), Level: models.LogLevelInfo, Source: "test"},
		{ID: "2", Timestamp: time.Now(), Level: models.LogLevelError, Source: "test"},
		{ID: "3", Timestamp: time.Now(), Level: models.LogLevelError, Source: "test"},
	}
	store.InsertLogs(entries)

	errorLogs, _ := store.QueryLogs(models.LogQuery{Level: models.LogLevelError, Source: "test"})
	if len(errorLogs) != 2 {
		t.Errorf("expected 2 error logs, got %d", len(errorLogs))
	}
}

func TestQueryWithSearch(t *testing.T) {
	store, cleanup := setupTestDB(t)
	defer cleanup()

	entries := []*models.LogEntry{
		{ID: "1", Timestamp: time.Now(), Level: models.LogLevelInfo, Message: "database connection established", Source: "test"},
		{ID: "2", Timestamp: time.Now(), Level: models.LogLevelError, Message: "database timeout", Source: "test"},
		{ID: "3", Timestamp: time.Now(), Level: models.LogLevelInfo, Message: "request completed", Source: "test"},
	}
	store.InsertLogs(entries)

	results, _ := store.QueryLogs(models.LogQuery{Search: "database", Source: "test"})
	if len(results) != 2 {
		t.Errorf("expected 2 results for 'database' search, got %d", len(results))
	}
}

func TestInsertAndQueryAnomaly(t *testing.T) {
	store, cleanup := setupTestDB(t)
	defer cleanup()

	anomaly := &models.Anomaly{
		ID:          "anm_1",
		Timestamp:   time.Now(),
		Type:        models.AnomalySpike,
		Severity:    models.SeverityHigh,
		Description: "Error spike detected",
		Source:      "test",
		Metadata:    map[string]string{"rate": "0.5"},
	}

	if err := store.InsertAnomaly(anomaly); err != nil {
		t.Fatalf("insert anomaly failed: %v", err)
	}

	anomalies, err := store.QueryAnomalies(10, "")
	if err != nil {
		t.Fatalf("query anomalies failed: %v", err)
	}

	if len(anomalies) != 1 {
		t.Fatalf("expected 1 anomaly, got %d", len(anomalies))
	}

	if anomalies[0].Severity != models.SeverityHigh {
		t.Errorf("severity = %v, want HIGH", anomalies[0].Severity)
	}
}

func TestGetSourceStats(t *testing.T) {
	store, cleanup := setupTestDB(t)
	defer cleanup()

	now := time.Now()
	entries := []*models.LogEntry{
		{ID: "1", Timestamp: now, Level: models.LogLevelInfo, Source: "api"},
		{ID: "2", Timestamp: now, Level: models.LogLevelError, Source: "api"},
		{ID: "3", Timestamp: now, Level: models.LogLevelWarn, Source: "api"},
		{ID: "4", Timestamp: now, Level: models.LogLevelInfo, Source: "db"},
	}
	store.InsertLogs(entries)

	stats, err := store.GetSourceStats()
	if err != nil {
		t.Fatalf("get source stats failed: %v", err)
	}

	if len(stats) != 2 {
		t.Fatalf("expected 2 source stats, got %d", len(stats))
	}
}

func TestGetLevelDistribution(t *testing.T) {
	store, cleanup := setupTestDB(t)
	defer cleanup()

	now := time.Now()
	entries := []*models.LogEntry{
		{ID: "1", Timestamp: now, Level: models.LogLevelInfo, Source: "test"},
		{ID: "2", Timestamp: now, Level: models.LogLevelError, Source: "test"},
		{ID: "3", Timestamp: now, Level: models.LogLevelError, Source: "test"},
	}
	store.InsertLogs(entries)

	dist, err := store.GetLevelDistribution("test")
	if err != nil {
		t.Fatalf("get level distribution failed: %v", err)
	}

	if dist["INFO"] != 1 {
		t.Errorf("INFO count = %d, want 1", dist["INFO"])
	}
	if dist["ERROR"] != 2 {
		t.Errorf("ERROR count = %d, want 2", dist["ERROR"])
	}
}

func TestCountLogs(t *testing.T) {
	store, cleanup := setupTestDB(t)
	defer cleanup()

	entries := []*models.LogEntry{
		{ID: "1", Timestamp: time.Now(), Level: models.LogLevelInfo, Source: "test"},
		{ID: "2", Timestamp: time.Now(), Level: models.LogLevelInfo, Source: "test"},
	}
	store.InsertLogs(entries)

	count, _ := store.CountLogs(models.LogQuery{Source: "test"})
	if count != 2 {
		t.Errorf("count = %d, want 2", count)
	}
}

func TestPagination(t *testing.T) {
	store, cleanup := setupTestDB(t)
	defer cleanup()

	now := time.Now()
	var entries []*models.LogEntry
	for i := 0; i < 10; i++ {
		entries = append(entries, &models.LogEntry{
			ID:        "log_" + string(rune('a'+i)),
			Timestamp: now.Add(time.Duration(i) * time.Second),
			Level:     models.LogLevelInfo,
			Source:    "test",
		})
	}
	store.InsertLogs(entries)

	page1, _ := store.QueryLogs(models.LogQuery{Source: "test", Limit: 5, Offset: 0})
	if len(page1) != 5 {
		t.Errorf("page1 size = %d, want 5", len(page1))
	}

	page2, _ := store.QueryLogs(models.LogQuery{Source: "test", Limit: 5, Offset: 5})
	if len(page2) != 5 {
		t.Errorf("page2 size = %d, want 5", len(page2))
	}
}

func TestTimeRangeQuery(t *testing.T) {
	store, cleanup := setupTestDB(t)
	defer cleanup()

	base := time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)
	entries := []*models.LogEntry{
		{ID: "1", Timestamp: base, Level: models.LogLevelInfo, Source: "test"},
		{ID: "2", Timestamp: base.Add(1 * time.Hour), Level: models.LogLevelInfo, Source: "test"},
		{ID: "3", Timestamp: base.Add(2 * time.Hour), Level: models.LogLevelInfo, Source: "test"},
	}
	store.InsertLogs(entries)

	query := models.LogQuery{
		Source:    "test",
		StartTime: base.Add(30 * time.Minute),
		EndTime:   base.Add(90 * time.Minute),
	}

	results, _ := store.QueryLogs(query)
	if len(results) != 1 {
		t.Errorf("expected 1 result in time range, got %d", len(results))
	}
}
