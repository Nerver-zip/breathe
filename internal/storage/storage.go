package storage

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/Nerver-zip/breathing-tui/internal/session"
	"github.com/jmoiron/sqlx"
	_ "modernc.org/sqlite"
)

const schema = `
PRAGMA foreign_keys = ON;
CREATE TABLE IF NOT EXISTS sessions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    started_at TEXT NOT NULL,
    ended_at TEXT NOT NULL,
    local_date TEXT NOT NULL,
    planned_rounds INTEGER NOT NULL,
    completed_rounds INTEGER NOT NULL,
    active_duration_ms INTEGER NOT NULL,
    wall_duration_ms INTEGER NOT NULL,
    status TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS rounds (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    session_id INTEGER NOT NULL REFERENCES sessions(id) ON DELETE CASCADE,
    round_index INTEGER NOT NULL,
    breathing_ms INTEGER NOT NULL,
    retention_ms INTEGER NOT NULL,
    recovery_ms INTEGER NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_sessions_local_date ON sessions(local_date);
CREATE INDEX IF NOT EXISTS idx_rounds_session_id ON rounds(session_id);
`

type Store struct {
	db *sqlx.DB
}

type SessionRecord struct {
	StartedAt      time.Time
	EndedAt        time.Time
	PlannedRounds  int
	ActiveDuration time.Duration
	Status         string
	Rounds         []session.RoundResult
}

type Summary struct {
	TotalSessions      int
	TotalRounds        int
	TotalActive        time.Duration
	AverageRetention   time.Duration
	BestRetention      time.Duration
	TodaySessions      int
	TodayRounds        int
	TodayBestRetention time.Duration
}

func DataDir() (string, error) {
	if xdg := os.Getenv("XDG_DATA_HOME"); xdg != "" {
		return filepath.Join(xdg, "breath"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".local", "share", "breath"), nil
}

func DefaultPath() (string, error) {
	dir, err := DataDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "breath.db"), nil
}

func Open(path string) (*Store, error) {
	if path == "" {
		var err error
		path, err = DefaultPath()
		if err != nil {
			return nil, err
		}
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	db, err := sqlx.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	if _, err := db.Exec(schema); err != nil {
		db.Close()
		return nil, fmt.Errorf("initialize database: %w", err)
	}
	return &Store{db: db}, nil
}

func (s *Store) Close() error { return s.db.Close() }

func (s *Store) SaveSession(ctx context.Context, rec SessionRecord) error {
	if rec.Status == "" {
		rec.Status = "completed"
	}
	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	wall := rec.EndedAt.Sub(rec.StartedAt)
	result, err := tx.ExecContext(ctx, `
		INSERT INTO sessions(started_at, ended_at, local_date, planned_rounds, completed_rounds, active_duration_ms, wall_duration_ms, status)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		rec.StartedAt.UTC().Format(time.RFC3339Nano),
		rec.EndedAt.UTC().Format(time.RFC3339Nano),
		rec.EndedAt.In(time.Local).Format("2006-01-02"),
		rec.PlannedRounds,
		len(rec.Rounds),
		rec.ActiveDuration.Milliseconds(),
		wall.Milliseconds(),
		rec.Status,
	)
	if err != nil {
		return err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	for _, round := range rec.Rounds {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO rounds(session_id, round_index, breathing_ms, retention_ms, recovery_ms)
			VALUES (?, ?, ?, ?, ?)`,
			id, round.Index, round.Breathing.Milliseconds(), round.Retention.Milliseconds(), round.Recovery.Milliseconds(),
		); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (s *Store) Summary(ctx context.Context) (Summary, error) {
	var out Summary
	row := s.db.QueryRowxContext(ctx, `
		SELECT
			COUNT(*) AS total_sessions,
			COALESCE(SUM(completed_rounds), 0) AS total_rounds,
			COALESCE(SUM(active_duration_ms), 0) AS total_active_ms
		FROM sessions`)
	var totalActiveMS int64
	if err := row.Scan(&out.TotalSessions, &out.TotalRounds, &totalActiveMS); err != nil {
		return out, err
	}
	out.TotalActive = time.Duration(totalActiveMS) * time.Millisecond

	row = s.db.QueryRowxContext(ctx, `SELECT COALESCE(AVG(retention_ms), 0), COALESCE(MAX(retention_ms), 0) FROM rounds`)
	var avgMS float64
	var bestMS int64
	if err := row.Scan(&avgMS, &bestMS); err != nil {
		return out, err
	}
	out.AverageRetention = time.Duration(avgMS * float64(time.Millisecond))
	out.BestRetention = time.Duration(bestMS) * time.Millisecond

	today := time.Now().In(time.Local).Format("2006-01-02")
	row = s.db.QueryRowxContext(ctx, `SELECT COUNT(*), COALESCE(SUM(completed_rounds), 0) FROM sessions WHERE local_date = ?`, today)
	if err := row.Scan(&out.TodaySessions, &out.TodayRounds); err != nil {
		return out, err
	}
	row = s.db.QueryRowxContext(ctx, `
		SELECT COALESCE(MAX(r.retention_ms), 0)
		FROM rounds r JOIN sessions s ON s.id = r.session_id
		WHERE s.local_date = ?`, today)
	var todayBestMS int64
	if err := row.Scan(&todayBestMS); err != nil {
		return out, err
	}
	out.TodayBestRetention = time.Duration(todayBestMS) * time.Millisecond
	return out, nil
}
