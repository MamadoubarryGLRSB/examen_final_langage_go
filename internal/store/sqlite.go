package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/MamadoubarryGLRSB/urlwatch/internal/domain"
	_ "modernc.org/sqlite"
)

// SQLiteStore persiste les lots dans un fichier .db
type SQLiteStore struct {
	db *sql.DB
}

func NewSQLite(path string) (*SQLiteStore, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)

	s := &SQLiteStore{db: db}
	if err := s.migrate(); err != nil {
		db.Close()
		return nil, err
	}
	return s, nil
}

func (s *SQLiteStore) migrate() error {
	_, err := s.db.Exec(`
		CREATE TABLE IF NOT EXISTS batches (
			id TEXT PRIMARY KEY,
			created_at TEXT NOT NULL,
			status TEXT NOT NULL DEFAULT '',
			summary_total INTEGER NOT NULL DEFAULT 0,
			summary_up INTEGER NOT NULL DEFAULT 0,
			summary_down INTEGER NOT NULL DEFAULT 0,
			summary_duration_ms INTEGER NOT NULL DEFAULT 0,
			results_json TEXT NOT NULL DEFAULT '[]'
		)
	`)
	return err
}

func (s *SQLiteStore) Save(_ context.Context, b domain.Batch) error {
	results, err := json.Marshal(b.Results)
	if err != nil {
		return err
	}

	_, err = s.db.Exec(`
		INSERT INTO batches (id, created_at, status, summary_total, summary_up, summary_down, summary_duration_ms, results_json)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			status = excluded.status,
			summary_total = excluded.summary_total,
			summary_up = excluded.summary_up,
			summary_down = excluded.summary_down,
			summary_duration_ms = excluded.summary_duration_ms,
			results_json = excluded.results_json
	`, b.ID, b.CreatedAt.UTC().Format(time.RFC3339), b.Status,
		b.Summary.Total, b.Summary.Up, b.Summary.Down, b.Summary.DurationMS, string(results))
	return err
}

func (s *SQLiteStore) Get(_ context.Context, id string) (domain.Batch, error) {
	row := s.db.QueryRow(`
		SELECT id, created_at, status, summary_total, summary_up, summary_down, summary_duration_ms, results_json
		FROM batches WHERE id = ?
	`, id)

	var b domain.Batch
	var createdAt string
	var resultsJSON string
	err := row.Scan(&b.ID, &createdAt, &b.Status,
		&b.Summary.Total, &b.Summary.Up, &b.Summary.Down, &b.Summary.DurationMS, &resultsJSON)
	if err == sql.ErrNoRows {
		return domain.Batch{}, fmt.Errorf("store.Get %q: %w", id, domain.ErrBatchNotFound)
	}
	if err != nil {
		return domain.Batch{}, err
	}

	b.CreatedAt, err = time.Parse(time.RFC3339, createdAt)
	if err != nil {
		return domain.Batch{}, err
	}
	if err := json.Unmarshal([]byte(resultsJSON), &b.Results); err != nil {
		return domain.Batch{}, err
	}
	return b, nil
}

func (s *SQLiteStore) List(_ context.Context, p domain.ListParams) (domain.ListResult, error) {
	where := ""
	args := []any{}
	if p.Status != "" {
		where = "WHERE status = ?"
		args = append(args, p.Status)
	}

	var total int
	countQ := "SELECT COUNT(*) FROM batches " + where
	if err := s.db.QueryRow(countQ, args...).Scan(&total); err != nil {
		return domain.ListResult{}, err
	}

	offset := (p.Page - 1) * p.Limit
	q := `
		SELECT id, created_at, status, summary_total, summary_up, summary_down, summary_duration_ms, results_json
		FROM batches ` + where + `
		ORDER BY created_at DESC
		LIMIT ? OFFSET ?`
	args = append(args, p.Limit, offset)

	rows, err := s.db.Query(q, args...)
	if err != nil {
		return domain.ListResult{}, err
	}
	defer rows.Close()

	items := []domain.Batch{}
	for rows.Next() {
		var b domain.Batch
		var createdAt string
		var resultsJSON string
		if err := rows.Scan(&b.ID, &createdAt, &b.Status,
			&b.Summary.Total, &b.Summary.Up, &b.Summary.Down, &b.Summary.DurationMS, &resultsJSON); err != nil {
			return domain.ListResult{}, err
		}
		b.CreatedAt, err = time.Parse(time.RFC3339, createdAt)
		if err != nil {
			return domain.ListResult{}, err
		}
		if err := json.Unmarshal([]byte(resultsJSON), &b.Results); err != nil {
			return domain.ListResult{}, err
		}
		items = append(items, b)
	}

	return domain.ListResult{Items: items, Page: p.Page, Limit: p.Limit, Total: total}, nil
}

func (s *SQLiteStore) Close() error {
	return s.db.Close()
}

// NewFromEnv choisit memory ou sqlite selon STORE
func NewFromEnv() (domain.Store, error) {
	switch envOrDefault("STORE", "memory") {
	case "sqlite":
		path := envOrDefault("SQLITE_PATH", "urlwatch.db")
		return NewSQLite(path)
	default:
		return NewMemory(), nil
	}
}

func envOrDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
