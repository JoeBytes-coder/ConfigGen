package storage

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "modernc.org/sqlite"

	"configgen/internal/domain"
)

type SQLiteStore struct {
	db *sql.DB
}

func buildDSN(path string) string {
	if path == ":memory:" {
		return "file::memory:?cache=shared&_pragma=busy_timeout(5000)"
	}
	p := path
	if !strings.HasPrefix(p, "file:") {
		p = "file:" + filepath.ToSlash(p)
	}
	sep := "?"
	if strings.Contains(p, "?") {
		sep = "&"
	}
	return p + sep + "_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)&_pragma=foreign_keys(1)"
}

func NewSQLiteStore(dsn string) (*SQLiteStore, error) {
	target := dsn
	if target == "" {
		target = "db/configgen.db"
	}
	if target != ":memory:" {
		dir := filepath.Dir(target)
		if dir != "." && dir != "" {
			if err := os.MkdirAll(dir, 0o755); err != nil {
				return nil, err
			}
		}
	}

	db, err := sql.Open("sqlite", buildDSN(target))
	if err != nil {
		return nil, err
	}
	// SQLite 只有一个写者，串行化连接可避免 "database is locked"。
	db.SetMaxOpenConns(1)
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, err
	}
	if _, err := db.Exec("CREATE TABLE IF NOT EXISTS render_records (" +
		"id INTEGER PRIMARY KEY AUTOINCREMENT," +
		"config_type TEXT NOT NULL," +
		"app_name TEXT NOT NULL," +
		"namespace TEXT NOT NULL DEFAULT ''," +
		"selection_json TEXT NOT NULL," +
		"plan_json TEXT NOT NULL," +
		"files_json TEXT NOT NULL," +
		"bundle TEXT NOT NULL," +
		"catalog_version TEXT NOT NULL," +
		"created_at TEXT NOT NULL)"); err != nil {
		db.Close()
		return nil, err
	}
	if _, err := db.Exec("CREATE INDEX IF NOT EXISTS idx_render_records_created ON render_records(created_at DESC)"); err != nil {
		db.Close()
		return nil, err
	}
	return &SQLiteStore{db: db}, nil
}

func (s *SQLiteStore) Save(rec domain.Record) (int64, error) {
	selection, err := json.Marshal(rec.Selection)
	if err != nil {
		return 0, err
	}
	plan, err := json.Marshal(rec.Plan)
	if err != nil {
		return 0, err
	}
	files, err := json.Marshal(rec.Files)
	if err != nil {
		return 0, err
	}
	if rec.CreatedAt.IsZero() {
		rec.CreatedAt = time.Now().UTC()
	}
	res, err := s.db.Exec(
		"INSERT INTO render_records (config_type, app_name, namespace, selection_json, plan_json, files_json, bundle, catalog_version, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)",
		string(rec.Type), rec.App.Name, rec.App.Namespace, string(selection), string(plan), string(files), rec.Bundle, rec.CatalogVersion,
		rec.CreatedAt.UTC().Format(time.RFC3339Nano),
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (s *SQLiteStore) Find(id int64) (domain.Record, error) {
	row := s.db.QueryRow(
		"SELECT id, config_type, selection_json, plan_json, files_json, bundle, catalog_version, created_at FROM render_records WHERE id = ?", id)
	return scanRecord(row)
}

func (s *SQLiteStore) List(offset, limit int) ([]domain.RecordSummary, error) {
	rows, err := s.db.Query(
		"SELECT id, config_type, app_name, namespace, plan_json, created_at FROM render_records ORDER BY id DESC LIMIT ? OFFSET ?", limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	summaries := make([]domain.RecordSummary, 0)
	for rows.Next() {
		var (
			item       domain.RecordSummary
			configType string
			planJSON   string
			createdAt  string
		)
		if err := rows.Scan(&item.ID, &configType, &item.AppName, &item.Namespace, &planJSON, &createdAt); err != nil {
			return nil, err
		}
		item.Type = domain.ConfigType(configType)
		parsed, err := time.Parse(time.RFC3339Nano, createdAt)
		if err != nil {
			return nil, err
		}
		item.CreatedAt = parsed
		var plan domain.Plan
		if err := json.Unmarshal([]byte(planJSON), &plan); err == nil {
			item.Kinds = make([]string, 0, len(plan.Nodes))
			for _, n := range plan.Nodes {
				item.Kinds = append(item.Kinds, n.Kind)
			}
		}
		summaries = append(summaries, item)
	}
	return summaries, rows.Err()
}

func (s *SQLiteStore) Count() (int, error) {
	var total int
	if err := s.db.QueryRow("SELECT COUNT(*) FROM render_records").Scan(&total); err != nil {
		return 0, err
	}
	return total, nil
}

func (s *SQLiteStore) Ping() error { return s.db.Ping() }

func (s *SQLiteStore) Close() error { return s.db.Close() }

func scanRecord(scanner interface{ Scan(dest ...any) error }) (domain.Record, error) {
	var (
		rec        domain.Record
		configType string
		selection  string
		plan       string
		files      string
		createdAt  string
	)
	if err := scanner.Scan(&rec.ID, &configType, &selection, &plan, &files, &rec.Bundle, &rec.CatalogVersion, &createdAt); err != nil {
		return domain.Record{}, err
	}
	rec.Type = domain.ConfigType(configType)
	if err := json.Unmarshal([]byte(selection), &rec.Selection); err != nil {
		return domain.Record{}, fmt.Errorf("decode selection: %w", err)
	}
	if err := json.Unmarshal([]byte(plan), &rec.Plan); err != nil {
		return domain.Record{}, fmt.Errorf("decode plan: %w", err)
	}
	if err := json.Unmarshal([]byte(files), &rec.Files); err != nil {
		return domain.Record{}, fmt.Errorf("decode files: %w", err)
	}
	parsed, err := time.Parse(time.RFC3339Nano, createdAt)
	if err != nil {
		return domain.Record{}, fmt.Errorf("decode created_at: %w", err)
	}
	rec.CreatedAt = parsed
	rec.App = rec.Plan.App
	return rec, nil
}
