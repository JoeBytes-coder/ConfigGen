package storage

import (
	"database/sql"
	"encoding/json"
	"strings"
	"time"

	"configgen/internal/domain"

	_ "github.com/mattn/go-sqlite3"
)

type SQLiteStore struct {
	db *sql.DB
}

func NewSQLiteStore(dsn string) (*SQLiteStore, error) {
	separator := "?"
	if strings.Contains(dsn, "?") {
		separator = "&"
	}
	db, err := sql.Open("sqlite3", dsn+separator+"_foreign_keys=on&_journal_mode=WAL")
	if err != nil {
		return nil, err
	}

	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS config_records (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	request TEXT NOT NULL,
	result TEXT NOT NULL,
	created_at DATETIME NOT NULL
)`); err != nil {
		_ = db.Close()
		return nil, err
	}

	return &SQLiteStore{db: db}, nil
}

func (s *SQLiteStore) Save(rec domain.ConfigRecord) (int64, error) {
	requestBytes, err := json.Marshal(rec.Request)
	if err != nil {
		return 0, err
	}
	resultBytes, err := json.Marshal(rec.Result)
	if err != nil {
		return 0, err
	}

	stmt, err := s.db.Prepare(`INSERT INTO config_records (request, result, created_at) VALUES (?, ?, ?)`)
	if err != nil {
		return 0, err
	}
	defer stmt.Close()

	res, err := stmt.Exec(string(requestBytes), string(resultBytes), rec.CreatedAt)
	if err != nil {
		return 0, err
	}

	return res.LastInsertId()
}

func (s *SQLiteStore) Find(id int64) (domain.ConfigRecord, error) {
	rows, err := s.db.Query(`SELECT id, request, result, created_at FROM config_records WHERE id = ?`, id)
	if err != nil {
		return domain.ConfigRecord{}, err
	}
	defer rows.Close()

	if !rows.Next() {
		return domain.ConfigRecord{}, sql.ErrNoRows
	}

	return scanRecord(rows)
}

func (s *SQLiteStore) List(offset, limit int) ([]domain.ConfigRecord, error) {
	rows, err := s.db.Query(`SELECT id, request, result, created_at FROM config_records ORDER BY id DESC LIMIT ? OFFSET ?`, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []domain.ConfigRecord
	for rows.Next() {
		rec, err := scanRecord(rows)
		if err != nil {
			return nil, err
		}
		records = append(records, rec)
	}
	if records == nil {
		records = []domain.ConfigRecord{}
	}
	return records, rows.Err()
}

func scanRecord(scanner interface {
	Scan(dest ...interface{}) error
}) (domain.ConfigRecord, error) {
	var rec domain.ConfigRecord
	var requestJSON, resultJSON, createdAt string

	if err := scanner.Scan(&rec.ID, &requestJSON, &resultJSON, &createdAt); err != nil {
		return domain.ConfigRecord{}, err
	}

	if err := json.Unmarshal([]byte(requestJSON), &rec.Request); err != nil {
		return domain.ConfigRecord{}, err
	}
	if err := json.Unmarshal([]byte(resultJSON), &rec.Result); err != nil {
		return domain.ConfigRecord{}, err
	}

	parsed, err := time.Parse(time.RFC3339Nano, createdAt)
	if err != nil {
		return domain.ConfigRecord{}, err
	}
	rec.CreatedAt = parsed

	return rec, nil
}

func (s *SQLiteStore) Close() error {
	return s.db.Close()
}
