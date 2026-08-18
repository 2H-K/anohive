package storage

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/2H-K/pulse/internal/models"

	_ "github.com/mattn/go-sqlite3"
)

type SQLiteStore struct {
	db *sql.DB
}

func NewSQLiteStore(dbPath string) (*SQLiteStore, error) {
	db, err := sql.Open("sqlite3", dbPath+"?_journal=WAL&_busy_timeout=10000&_synchronous=NORMAL&_cache_size=-8000&_foreign_keys=ON")
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	db.SetMaxOpenConns(4)
	db.SetMaxIdleConns(2)
	db.SetConnMaxLifetime(time.Hour)

	store := &SQLiteStore{db: db}
	if err := store.migrate(); err != nil {
		return nil, fmt.Errorf("migrate: %w", err)
	}

	return store, nil
}

func (s *SQLiteStore) migrate() error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS logs (
			id TEXT PRIMARY KEY,
			timestamp DATETIME NOT NULL,
			level TEXT NOT NULL,
			message TEXT NOT NULL,
			source TEXT,
			host TEXT,
			service TEXT,
			fields TEXT,
			raw TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE INDEX IF NOT EXISTS idx_logs_timestamp ON logs(timestamp)`,
		`CREATE INDEX IF NOT EXISTS idx_logs_source ON logs(source)`,
		`CREATE INDEX IF NOT EXISTS idx_logs_level ON logs(level)`,
		`CREATE INDEX IF NOT EXISTS idx_logs_source_timestamp ON logs(source, timestamp)`,
		`CREATE TABLE IF NOT EXISTS anomalies (
			id TEXT PRIMARY KEY,
			timestamp DATETIME NOT NULL,
			type TEXT NOT NULL,
			severity TEXT NOT NULL,
			description TEXT NOT NULL,
			source TEXT,
			log_entry_id TEXT,
			metadata TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE INDEX IF NOT EXISTS idx_anomalies_timestamp ON anomalies(timestamp)`,
		`CREATE INDEX IF NOT EXISTS idx_anomalies_severity ON anomalies(severity)`,
		`CREATE INDEX IF NOT EXISTS idx_anomalies_source ON anomalies(source)`,
		`PRAGMA optimize`,
	}

	for _, q := range queries {
		if _, err := s.db.Exec(q); err != nil {
			return fmt.Errorf("migration failed: %s: %w", q, err)
		}
	}

	return nil
}

func (s *SQLiteStore) InsertLog(entry *models.LogEntry) error {
	fieldsJSON, _ := json.Marshal(entry.Fields)

	_, err := s.db.Exec(
		`INSERT OR REPLACE INTO logs (id, timestamp, level, message, source, host, service, fields, raw)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		entry.ID, entry.Timestamp, entry.Level, entry.Message,
		entry.Source, entry.Host, entry.Service, string(fieldsJSON), entry.Raw,
	)

	return err
}

func (s *SQLiteStore) InsertLogs(entries []*models.LogEntry) error {
	if len(entries) == 0 {
		return nil
	}

	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(
		`INSERT OR REPLACE INTO logs (id, timestamp, level, message, source, host, service, fields, raw)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
	)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, entry := range entries {
		fieldsJSON, _ := json.Marshal(entry.Fields)
		if _, err := stmt.Exec(
			entry.ID, entry.Timestamp, entry.Level, entry.Message,
			entry.Source, entry.Host, entry.Service, string(fieldsJSON), entry.Raw,
		); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (s *SQLiteStore) QueryLogs(query models.LogQuery) ([]*models.LogEntry, error) {
	where := []string{"1=1"}
	args := []interface{}{}

	if query.Source != "" {
		where = append(where, "source = ?")
		args = append(args, query.Source)
	}
	if query.Level != "" {
		where = append(where, "level = ?")
		args = append(args, query.Level)
	}
	if query.Search != "" {
		escaped := strings.ReplaceAll(query.Search, "%", "\\%")
		escaped = strings.ReplaceAll(escaped, "_", "\\_")
		where = append(where, "message LIKE ? ESCAPE '\\'")
		args = append(args, "%"+escaped+"%")
	}
	if !query.StartTime.IsZero() {
		where = append(where, "timestamp >= ?")
		args = append(args, query.StartTime)
	}
	if !query.EndTime.IsZero() {
		where = append(where, "timestamp <= ?")
		args = append(args, query.EndTime)
	}

	limit := 100
	if query.Limit > 0 && query.Limit <= 1000 {
		limit = query.Limit
	}
	offset := query.Offset

	q := fmt.Sprintf(
		"SELECT id, timestamp, level, message, source, host, service, fields, raw FROM logs WHERE %s ORDER BY timestamp DESC LIMIT ? OFFSET ?",
		strings.Join(where, " AND "),
	)
	args = append(args, limit, offset)

	rows, err := s.db.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return s.scanLogs(rows)
}

func (s *SQLiteStore) CountLogs(query models.LogQuery) (int64, error) {
	where := []string{"1=1"}
	args := []interface{}{}

	if query.Source != "" {
		where = append(where, "source = ?")
		args = append(args, query.Source)
	}
	if query.Level != "" {
		where = append(where, "level = ?")
		args = append(args, query.Level)
	}
	if query.Search != "" {
		escaped := strings.ReplaceAll(query.Search, "%", "\\%")
		escaped = strings.ReplaceAll(escaped, "_", "\\_")
		where = append(where, "message LIKE ? ESCAPE '\\'")
		args = append(args, "%"+escaped+"%")
	}
	if !query.StartTime.IsZero() {
		where = append(where, "timestamp >= ?")
		args = append(args, query.StartTime)
	}
	if !query.EndTime.IsZero() {
		where = append(where, "timestamp <= ?")
		args = append(args, query.EndTime)
	}

	q := fmt.Sprintf("SELECT COUNT(*) FROM logs WHERE %s", strings.Join(where, " AND "))
	var count int64
	err := s.db.QueryRow(q, args...).Scan(&count)
	return count, err
}

func (s *SQLiteStore) InsertAnomaly(anomaly *models.Anomaly) error {
	metadataJSON, _ := json.Marshal(anomaly.Metadata)

	_, err := s.db.Exec(
		`INSERT OR REPLACE INTO anomalies (id, timestamp, type, severity, description, source, log_entry_id, metadata)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		anomaly.ID, anomaly.Timestamp, anomaly.Type, anomaly.Severity,
		anomaly.Description, anomaly.Source, anomaly.LogEntryID, string(metadataJSON),
	)

	return err
}

func (s *SQLiteStore) InsertAnomalies(anomalies []*models.Anomaly) error {
	if len(anomalies) == 0 {
		return nil
	}

	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(
		`INSERT OR REPLACE INTO anomalies (id, timestamp, type, severity, description, source, log_entry_id, metadata)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
	)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, a := range anomalies {
		metadataJSON, _ := json.Marshal(a.Metadata)
		if _, err := stmt.Exec(
			a.ID, a.Timestamp, a.Type, a.Severity,
			a.Description, a.Source, a.LogEntryID, string(metadataJSON),
		); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (s *SQLiteStore) QueryAnomalies(limit int, severity string) ([]*models.Anomaly, error) {
	where := "1=1"
	args := []interface{}{}

	if severity != "" {
		where = "severity = ?"
		args = append(args, severity)
	}

	if limit <= 0 || limit > 500 {
		limit = 100
	}

	q := fmt.Sprintf(
		"SELECT id, timestamp, type, severity, description, source, log_entry_id, metadata FROM anomalies WHERE %s ORDER BY timestamp DESC LIMIT ?",
		where,
	)
	args = append(args, limit)

	rows, err := s.db.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return s.scanAnomalies(rows)
}

func (s *SQLiteStore) CountAnomalies() (int64, error) {
	var count int64
	err := s.db.QueryRow("SELECT COUNT(*) FROM anomalies").Scan(&count)
	return count, err
}

func (s *SQLiteStore) GetSourceStats() ([]models.SourceStats, error) {
	query := `
		SELECT source,
		       COUNT(*) as total,
		       SUM(CASE WHEN level IN ('ERROR', 'FATAL') THEN 1 ELSE 0 END) as errors,
		       SUM(CASE WHEN level = 'WARN' THEN 1 ELSE 0 END) as warns,
		       MIN(timestamp) as first_log,
		       MAX(timestamp) as last_log
		FROM logs
		GROUP BY source
		ORDER BY total DESC
		LIMIT 100
	`

	rows, err := s.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stats []models.SourceStats
	for rows.Next() {
		var stat models.SourceStats
		var firstLogStr, lastLogStr sql.NullString
		if err := rows.Scan(&stat.Source, &stat.TotalLogs, &stat.ErrorCount, &stat.WarnCount, &firstLogStr, &lastLogStr); err != nil {
			return nil, err
		}
		var firstLog, lastLog time.Time
		if firstLogStr.Valid {
			firstLog, _ = time.Parse("2006-01-02 15:04:05", firstLogStr.String)
		}
		if lastLogStr.Valid {
			lastLog, _ = time.Parse("2006-01-02 15:04:05", lastLogStr.String)
		}
		if !lastLog.IsZero() && !firstLog.IsZero() {
			duration := lastLog.Sub(firstLog).Minutes()
			if duration > 0 {
				stat.LogsPerMin = float64(stat.TotalLogs) / duration
			}
			stat.LastLogTime = lastLog
		}
		stats = append(stats, stat)
	}

	return stats, rows.Err()
}

func (s *SQLiteStore) GetLevelDistribution(source string) (map[string]int64, error) {
	where := "1=1"
	args := []interface{}{}
	if source != "" {
		where = "source = ?"
		args = append(args, source)
	}

	q := fmt.Sprintf("SELECT level, COUNT(*) FROM logs WHERE %s GROUP BY level", where)
	rows, err := s.db.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	dist := make(map[string]int64)
	for rows.Next() {
		var level string
		var count int64
		if err := rows.Scan(&level, &count); err != nil {
			return nil, err
		}
		dist[level] = count
	}

	return dist, rows.Err()
}

func (s *SQLiteStore) GetRecentLogs(source string, minutes int) ([]*models.LogEntry, error) {
	cutoff := time.Now().Add(-time.Duration(minutes) * time.Minute)
	query := models.LogQuery{
		StartTime: cutoff,
		Limit:     1000,
	}
	if source != "" {
		query.Source = source
	}
	return s.QueryLogs(query)
}

func (s *SQLiteStore) GetLogsInWindow(source string, start, end time.Time) ([]*models.LogEntry, error) {
	query := models.LogQuery{
		Source:    source,
		StartTime: start,
		EndTime:   end,
		Limit:     10000,
	}
	return s.QueryLogs(query)
}

func (s *SQLiteStore) GetTotalLogCount() (int64, error) {
	var count int64
	err := s.db.QueryRow("SELECT COUNT(*) FROM logs").Scan(&count)
	return count, err
}

func (s *SQLiteStore) DeleteOldLogs(retention time.Duration) (int64, error) {
	cutoff := time.Now().Add(-retention)
	result, err := s.db.Exec("DELETE FROM logs WHERE timestamp < ?", cutoff)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

func (s *SQLiteStore) DeleteOldAnomalies(retention time.Duration) (int64, error) {
	cutoff := time.Now().Add(-retention)
	result, err := s.db.Exec("DELETE FROM anomalies WHERE timestamp < ?", cutoff)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

func (s *SQLiteStore) Vacuum() error {
	_, err := s.db.Exec("VACUUM")
	return err
}

func (s *SQLiteStore) scanLogs(rows *sql.Rows) ([]*models.LogEntry, error) {
	var entries []*models.LogEntry
	for rows.Next() {
		var e models.LogEntry
		var fieldsJSON string
		if err := rows.Scan(&e.ID, &e.Timestamp, &e.Level, &e.Message,
			&e.Source, &e.Host, &e.Service, &fieldsJSON, &e.Raw); err != nil {
			return nil, err
		}
		if fieldsJSON != "" {
			if err := json.Unmarshal([]byte(fieldsJSON), &e.Fields); err != nil {
				log.Printf("Failed to unmarshal fields JSON: %v", err)
				e.Fields = make(map[string]string)
			}
		}
		if e.Fields == nil {
			e.Fields = make(map[string]string)
		}
		entries = append(entries, &e)
	}
	return entries, rows.Err()
}

func (s *SQLiteStore) scanAnomalies(rows *sql.Rows) ([]*models.Anomaly, error) {
	var anomalies []*models.Anomaly
	for rows.Next() {
		var a models.Anomaly
		var metadataJSON string
		if err := rows.Scan(&a.ID, &a.Timestamp, &a.Type, &a.Severity,
			&a.Description, &a.Source, &a.LogEntryID, &metadataJSON); err != nil {
			return nil, err
		}
		if metadataJSON != "" {
			if err := json.Unmarshal([]byte(metadataJSON), &a.Metadata); err != nil {
				log.Printf("Failed to unmarshal metadata JSON: %v", err)
				a.Metadata = make(map[string]string)
			}
		}
		if a.Metadata == nil {
			a.Metadata = make(map[string]string)
		}
		anomalies = append(anomalies, &a)
	}
	return anomalies, rows.Err()
}

func (s *SQLiteStore) Close() error {
	return s.db.Close()
}

// Ping checks if the database connection is alive
func (s *SQLiteStore) Ping() error {
	return s.db.Ping()
}

// Backup creates a backup of the database to the specified path
func (s *SQLiteStore) Backup(backupPath string) error {
	_, err := s.db.Exec(fmt.Sprintf("VACUUM INTO '%s'", backupPath))
	return err
}

// Stats returns database statistics
func (s *SQLiteStore) Stats() (map[string]interface{}, error) {
	var pageCount, pageSize, freePages int
	if err := s.db.QueryRow("PRAGMA page_count").Scan(&pageCount); err != nil {
		return nil, err
	}
	if err := s.db.QueryRow("PRAGMA page_size").Scan(&pageSize); err != nil {
		return nil, err
	}
	if err := s.db.QueryRow("PRAGMA freelist_count").Scan(&freePages); err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"page_count":  pageCount,
		"page_size":   pageSize,
		"free_pages":  freePages,
		"size_bytes":  int64(pageCount) * int64(pageSize),
		"free_bytes":  int64(freePages) * int64(pageSize),
	}, nil
}
