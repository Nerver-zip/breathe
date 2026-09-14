package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
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

type DayActivity struct {
	Date           string        `json:"date"`
	DayLabel       string        `json:"day_label"`
	Rounds         int           `json:"rounds"`
	ActiveDuration time.Duration `json:"active_duration"`
}

type HeatmapDay struct {
	Date   string `json:"date"`
	Rounds int    `json:"rounds"`
}

type StatsReport struct {
	// Today
	TodaySessions      int           `json:"today_sessions"`
	TodayRounds        int           `json:"today_rounds"`
	TodayActive        time.Duration `json:"today_active"`
	TodayBestRetention time.Duration `json:"today_best_retention"`

	// Retention
	AverageRetention time.Duration `json:"average_retention"`
	BestRetention    time.Duration `json:"best_retention"`
	LatestRetention  time.Duration `json:"latest_retention"`

	// Streak
	CurrentStreak int `json:"current_streak"`
	BestStreak    int `json:"best_streak"`

	// All time
	TotalSessions int           `json:"total_sessions"`
	TotalRounds   int           `json:"total_rounds"`
	TotalActive   time.Duration `json:"total_active"`

	// Time series
	Recent7Days []DayActivity `json:"recent_7_days"`
	HeatmapDays []HeatmapDay  `json:"heatmap_days"`
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
	if err := initSchema(db); err != nil {
		db.Close()
		return nil, fmt.Errorf("initialize database: %w", err)
	}
	return &Store{db: db}, nil
}

func initSchema(db *sqlx.DB) error {
	var version int
	if err := db.Get(&version, "PRAGMA user_version"); err != nil {
		return err
	}
	if version == 0 {
		if _, err := db.Exec(schema); err != nil {
			return err
		}
		if _, err := db.Exec("PRAGMA user_version = 1"); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) Close() error { return s.db.Close() }

func (s *Store) DB() *sqlx.DB { return s.db }

func (s *Store) CreateSession(ctx context.Context, plannedRounds int, startedAt time.Time) (int64, error) {
	localDate := startedAt.In(time.Local).Format("2006-01-02")
	startedStr := startedAt.UTC().Format(time.RFC3339Nano)
	res, err := s.db.ExecContext(ctx, `
		INSERT INTO sessions(started_at, ended_at, local_date, planned_rounds, completed_rounds, active_duration_ms, wall_duration_ms, status)
		VALUES (?, ?, ?, ?, 0, 0, 0, 'active')`,
		startedStr, startedStr, localDate, plannedRounds,
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (s *Store) SaveRound(ctx context.Context, sessionID int64, res session.RoundResult) error {
	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.ExecContext(ctx, `
		INSERT INTO rounds(session_id, round_index, breathing_ms, retention_ms, recovery_ms)
		VALUES (?, ?, ?, ?, ?)`,
		sessionID, res.Index, res.Breathing.Milliseconds(), res.Retention.Milliseconds(), res.Recovery.Milliseconds(),
	)
	if err != nil {
		return err
	}

	_, err = tx.ExecContext(ctx, `
		UPDATE sessions
		SET completed_rounds = (SELECT COUNT(*) FROM rounds WHERE session_id = ?)
		WHERE id = ?`,
		sessionID, sessionID,
	)
	if err != nil {
		return err
	}

	return tx.Commit()
}

func (s *Store) EndSession(ctx context.Context, sessionID int64, endedAt time.Time, activeDuration, wallDuration time.Duration, status string) error {
	if status == "" {
		status = "completed"
	}
	endedStr := endedAt.UTC().Format(time.RFC3339Nano)
	_, err := s.db.ExecContext(ctx, `
		UPDATE sessions
		SET ended_at = ?,
		    active_duration_ms = ?,
		    wall_duration_ms = ?,
		    status = ?
		WHERE id = ?`,
		endedStr, activeDuration.Milliseconds(), wallDuration.Milliseconds(), status, sessionID,
	)
	return err
}

func (s *Store) CleanupUnfinishedSessions(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE sessions
		SET status = 'abandoned'
		WHERE status = 'active'`)
	return err
}

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
	report, err := s.GetStats(ctx, time.Now())
	if err != nil {
		return Summary{}, err
	}
	return Summary{
		TotalSessions:      report.TotalSessions,
		TotalRounds:        report.TotalRounds,
		TotalActive:        report.TotalActive,
		AverageRetention:   report.AverageRetention,
		BestRetention:      report.BestRetention,
		TodaySessions:      report.TodaySessions,
		TodayRounds:        report.TodayRounds,
		TodayBestRetention: report.TodayBestRetention,
	}, nil
}

func (s *Store) GetStats(ctx context.Context, refTime time.Time) (StatsReport, error) {
	var out StatsReport
	localRef := refTime.In(time.Local)
	todayStr := localRef.Format("2006-01-02")

	// All-time totals
	row := s.db.QueryRowxContext(ctx, `
		SELECT
			COALESCE(SUM(CASE WHEN status = 'completed' THEN 1 ELSE 0 END), 0) AS completed_sessions,
			COALESCE(SUM(completed_rounds), 0) AS total_rounds,
			COALESCE(SUM(active_duration_ms), 0) AS total_active_ms
		FROM sessions`)
	var totalActiveMS int64
	if err := row.Scan(&out.TotalSessions, &out.TotalRounds, &totalActiveMS); err != nil {
		return out, err
	}
	out.TotalActive = time.Duration(totalActiveMS) * time.Millisecond

	// Retention metrics
	row = s.db.QueryRowxContext(ctx, `SELECT COALESCE(AVG(retention_ms), 0), COALESCE(MAX(retention_ms), 0) FROM rounds`)
	var avgMS float64
	var bestMS int64
	if err := row.Scan(&avgMS, &bestMS); err != nil {
		return out, err
	}
	out.AverageRetention = time.Duration(avgMS * float64(time.Millisecond))
	out.BestRetention = time.Duration(bestMS) * time.Millisecond

	var latestMS int64
	err := s.db.QueryRowxContext(ctx, `SELECT retention_ms FROM rounds ORDER BY id DESC LIMIT 1`).Scan(&latestMS)
	if err == nil {
		out.LatestRetention = time.Duration(latestMS) * time.Millisecond
	}

	// Today metrics
	row = s.db.QueryRowxContext(ctx, `
		SELECT
			COALESCE(SUM(CASE WHEN status = 'completed' THEN 1 ELSE 0 END), 0),
			COALESCE(SUM(completed_rounds), 0),
			COALESCE(SUM(active_duration_ms), 0)
		FROM sessions
		WHERE local_date = ?`, todayStr)
	var todayActiveMS int64
	if err := row.Scan(&out.TodaySessions, &out.TodayRounds, &todayActiveMS); err != nil {
		return out, err
	}
	out.TodayActive = time.Duration(todayActiveMS) * time.Millisecond

	var todayBestMS int64
	_ = s.db.QueryRowxContext(ctx, `
		SELECT COALESCE(MAX(r.retention_ms), 0)
		FROM rounds r JOIN sessions s ON s.id = r.session_id
		WHERE s.local_date = ?`, todayStr).Scan(&todayBestMS)
	out.TodayBestRetention = time.Duration(todayBestMS) * time.Millisecond

	// Streak calculation: distinct active dates with completed_rounds > 0
	rows, err := s.db.QueryxContext(ctx, `SELECT DISTINCT local_date FROM sessions WHERE completed_rounds > 0 ORDER BY local_date ASC`)
	if err != nil {
		return out, err
	}
	var activeDates []string
	for rows.Next() {
		var d string
		if err := rows.Scan(&d); err == nil {
			activeDates = append(activeDates, d)
		}
	}
	rows.Close()
	out.CurrentStreak, out.BestStreak = CalculateStreak(activeDates, localRef)

	// Recent 7 days (including today)
	sevenDaysAgo := localRef.AddDate(0, 0, -6).Format("2006-01-02")
	dayQueryRows, err := s.db.QueryxContext(ctx, `
		SELECT local_date, COALESCE(SUM(completed_rounds), 0) AS rounds, COALESCE(SUM(active_duration_ms), 0) AS active_ms
		FROM sessions
		WHERE local_date >= ? AND local_date <= ?
		GROUP BY local_date`, sevenDaysAgo, todayStr)
	if err != nil {
		return out, err
	}
	roundsByDate := make(map[string]int)
	activeByDate := make(map[string]int64)
	for dayQueryRows.Next() {
		var d string
		var r int
		var a int64
		if err := dayQueryRows.Scan(&d, &r, &a); err == nil {
			roundsByDate[d] = r
			activeByDate[d] = a
		}
	}
	dayQueryRows.Close()

	out.Recent7Days = make([]DayActivity, 7)
	for i := 0; i < 7; i++ {
		cur := localRef.AddDate(0, 0, -(6 - i))
		curStr := cur.Format("2006-01-02")
		out.Recent7Days[i] = DayActivity{
			Date:           curStr,
			DayLabel:       cur.Format("Mon"),
			Rounds:         roundsByDate[curStr],
			ActiveDuration: time.Duration(activeByDate[curStr]) * time.Millisecond,
		}
	}

	// ~4-month heatmap: 18 weeks (126 days) ending on today
	totalHeatmapDays := 18 * 7
	heatmapStart := localRef.AddDate(0, 0, -(totalHeatmapDays - 1)).Format("2006-01-02")
	heatRows, err := s.db.QueryxContext(ctx, `
		SELECT local_date, COALESCE(SUM(completed_rounds), 0) AS rounds
		FROM sessions
		WHERE local_date >= ? AND local_date <= ?
		GROUP BY local_date`, heatmapStart, todayStr)
	if err != nil {
		return out, err
	}
	heatMapData := make(map[string]int)
	for heatRows.Next() {
		var d string
		var r int
		if err := heatRows.Scan(&d, &r); err == nil {
			heatMapData[d] = r
		}
	}
	heatRows.Close()

	out.HeatmapDays = make([]HeatmapDay, totalHeatmapDays)
	for i := 0; i < totalHeatmapDays; i++ {
		cur := localRef.AddDate(0, 0, -(totalHeatmapDays - 1 - i))
		curStr := cur.Format("2006-01-02")
		out.HeatmapDays[i] = HeatmapDay{
			Date:   curStr,
			Rounds: heatMapData[curStr],
		}
	}

	return out, nil
}

func CalculateStreak(activeDates []string, refTime time.Time) (currentStreak int, bestStreak int) {
	if len(activeDates) == 0 {
		return 0, 0
	}

	sort.Strings(activeDates)
	activeSet := make(map[string]bool)
	var parsedDates []time.Time
	for _, s := range activeDates {
		s = strings.TrimSpace(s)
		if s == "" || activeSet[s] {
			continue
		}
		activeSet[s] = true
		t, err := time.Parse("2006-01-02", s)
		if err == nil {
			parsedDates = append(parsedDates, t)
		}
	}

	if len(parsedDates) == 0 {
		return 0, 0
	}

	// Best streak
	curRun := 1
	bestStreak = 1
	for i := 1; i < len(parsedDates); i++ {
		prev := parsedDates[i-1]
		curr := parsedDates[i]
		expected := prev.AddDate(0, 0, 1)
		if curr.Year() == expected.Year() && curr.Month() == expected.Month() && curr.Day() == expected.Day() {
			curRun++
		} else {
			curRun = 1
		}
		if curRun > bestStreak {
			bestStreak = curRun
		}
	}

	// Current streak
	todayStr := refTime.Format("2006-01-02")
	yesterdayStr := refTime.AddDate(0, 0, -1).Format("2006-01-02")

	var streakStart time.Time
	if activeSet[todayStr] {
		streakStart, _ = time.Parse("2006-01-02", todayStr)
	} else if activeSet[yesterdayStr] {
		streakStart, _ = time.Parse("2006-01-02", yesterdayStr)
	} else {
		return 0, bestStreak
	}

	currentStreak = 0
	checkDate := streakStart
	for {
		ds := checkDate.Format("2006-01-02")
		if activeSet[ds] {
			currentStreak++
			checkDate = checkDate.AddDate(0, 0, -1)
		} else {
			break
		}
	}

	return currentStreak, bestStreak
}

func FormatDuration(d time.Duration) string {
	if d <= 0 {
		return "00:00"
	}
	totalSecs := int(d.Round(time.Second).Seconds())
	mins := totalSecs / 60
	secs := totalSecs % 60
	if mins >= 60 {
		hrs := mins / 60
		mins = mins % 60
		return fmt.Sprintf("%dh %02dm", hrs, mins)
	}
	return fmt.Sprintf("%02d:%02d", mins, secs)
}

func (r StatsReport) PlainText() string {
	var b strings.Builder
	fmt.Fprintf(&b, "Today: %d sessions • %d rounds • active %s • best retention %s\n",
		r.TodaySessions, r.TodayRounds, FormatDuration(r.TodayActive), FormatDuration(r.TodayBestRetention))
	fmt.Fprintf(&b, "Retention: average %s • best %s • latest %s\n",
		FormatDuration(r.AverageRetention), FormatDuration(r.BestRetention), FormatDuration(r.LatestRetention))
	fmt.Fprintf(&b, "Streak: current %d days • best %d days\n",
		r.CurrentStreak, r.BestStreak)
	fmt.Fprintf(&b, "All time: %d sessions • %d rounds • active %s\n",
		r.TotalSessions, r.TotalRounds, FormatDuration(r.TotalActive))

	if len(r.Recent7Days) > 0 {
		b.WriteString("\nRecent 7 days:\n")
		for _, day := range r.Recent7Days {
			fmt.Fprintf(&b, "  %s (%s): %d rounds (%s)\n",
				day.Date, day.DayLabel, day.Rounds, FormatDuration(day.ActiveDuration))
		}
	}
	return b.String()
}

func (r StatsReport) JSON() (string, error) {
	data, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}
