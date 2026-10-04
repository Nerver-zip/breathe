package storage

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/Nerver-zip/breathing-tui/internal/session"
)

func newTestStore(t *testing.T) *Store {
	t.Helper()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test_breath.db")
	store, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open() error: %v", err)
	}
	t.Cleanup(func() {
		store.Close()
	})
	return store
}

func TestEmptyStoreStats(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	report, err := store.GetStats(ctx, time.Now())
	if err != nil {
		t.Fatalf("GetStats error: %v", err)
	}

	if report.TotalSessions != 0 || report.TotalRounds != 0 || report.TotalActive != 0 {
		t.Fatalf("expected all zeros for empty db, got %#v", report)
	}
	if report.CurrentStreak != 0 || report.BestStreak != 0 {
		t.Fatalf("expected 0 streak, got current=%d best=%d", report.CurrentStreak, report.BestStreak)
	}
	if len(report.Recent7Days) != 7 {
		t.Fatalf("expected 7 days, got %d", len(report.Recent7Days))
	}
	for _, d := range report.Recent7Days {
		if d.Rounds != 0 || d.ActiveDuration != 0 {
			t.Fatalf("expected 0 rounds for day %s, got %d", d.Date, d.Rounds)
		}
	}
	if len(report.HeatmapDays) != 126 {
		t.Fatalf("expected 126 heatmap days, got %d", len(report.HeatmapDays))
	}

	plain := report.PlainText()
	if plain == "" {
		t.Fatal("empty plain text report")
	}
	jsonStr, err := report.JSON()
	if err != nil || jsonStr == "" {
		t.Fatalf("failed JSON report: %v", err)
	}
}

func TestIncrementalSessionPersistence(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	startTime := time.Date(2026, 9, 13, 10, 0, 0, 0, time.Local)
	sessID, err := store.CreateSession(ctx, 3, startTime)
	if err != nil {
		t.Fatalf("CreateSession error: %v", err)
	}

	// Save round 1
	err = store.SaveRound(ctx, sessID, session.RoundResult{
		Index:     1,
		Breathing: 3 * time.Minute,
		Retention: 45 * time.Second,
		Recovery:  30 * time.Second,
	})
	if err != nil {
		t.Fatalf("SaveRound 1 error: %v", err)
	}

	// Save round 2
	err = store.SaveRound(ctx, sessID, session.RoundResult{
		Index:     2,
		Breathing: 3 * time.Minute,
		Retention: 60 * time.Second,
		Recovery:  30 * time.Second,
	})
	if err != nil {
		t.Fatalf("SaveRound 2 error: %v", err)
	}

	// End session
	endTime := startTime.Add(10 * time.Minute)
	err = store.EndSession(ctx, sessID, endTime, 8*time.Minute+45*time.Second, 10*time.Minute, "completed")
	if err != nil {
		t.Fatalf("EndSession error: %v", err)
	}

	report, err := store.GetStats(ctx, startTime)
	if err != nil {
		t.Fatalf("GetStats error: %v", err)
	}

	if report.TodaySessions != 1 {
		t.Fatalf("expected 1 today session, got %d", report.TodaySessions)
	}
	if report.TodayRounds != 2 {
		t.Fatalf("expected 2 today rounds, got %d", report.TodayRounds)
	}
	if report.TodayBestRetention != 60*time.Second {
		t.Fatalf("expected 60s best retention, got %s", report.TodayBestRetention)
	}
	if report.LatestRetention != 60*time.Second {
		t.Fatalf("expected 60s latest retention, got %s", report.LatestRetention)
	}
	if report.AverageRetention != (45+60)*time.Second/2 {
		t.Fatalf("expected 52.5s avg retention, got %s", report.AverageRetention)
	}
}

func TestAbandonedSessionPreservesCompletedRounds(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	startTime := time.Date(2026, 9, 13, 14, 0, 0, 0, time.Local)
	sessID, err := store.CreateSession(ctx, 3, startTime)
	if err != nil {
		t.Fatalf("CreateSession error: %v", err)
	}

	// Complete round 1 only
	err = store.SaveRound(ctx, sessID, session.RoundResult{
		Index:     1,
		Breathing: 3 * time.Minute,
		Retention: 50 * time.Second,
		Recovery:  30 * time.Second,
	})
	if err != nil {
		t.Fatalf("SaveRound error: %v", err)
	}

	// User abandons session on round 2
	endTime := startTime.Add(5 * time.Minute)
	err = store.EndSession(ctx, sessID, endTime, 4*time.Minute+20*time.Second, 5*time.Minute, "abandoned")
	if err != nil {
		t.Fatalf("EndSession error: %v", err)
	}

	report, err := store.GetStats(ctx, startTime)
	if err != nil {
		t.Fatalf("GetStats error: %v", err)
	}

	// Abandoned session does NOT count as a completed session
	if report.TodaySessions != 0 {
		t.Fatalf("expected 0 completed sessions today, got %d", report.TodaySessions)
	}
	// But completed round 1 IS counted!
	if report.TodayRounds != 1 {
		t.Fatalf("expected 1 completed round, got %d", report.TodayRounds)
	}
	if report.TotalRounds != 1 {
		t.Fatalf("expected 1 total round, got %d", report.TotalRounds)
	}
	if report.CurrentStreak != 1 {
		t.Fatalf("expected streak 1 from completed round, got %d", report.CurrentStreak)
	}
}

func TestAbandonedSessionWithZeroRoundsDoesNotCount(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	startTime := time.Date(2026, 9, 13, 14, 0, 0, 0, time.Local)
	sessID, err := store.CreateSession(ctx, 3, startTime)
	if err != nil {
		t.Fatalf("CreateSession error: %v", err)
	}

	// User quits immediately without completing any round
	endTime := startTime.Add(30 * time.Second)
	err = store.EndSession(ctx, sessID, endTime, 20*time.Second, 30*time.Second, "abandoned")
	if err != nil {
		t.Fatalf("EndSession error: %v", err)
	}

	report, err := store.GetStats(ctx, startTime)
	if err != nil {
		t.Fatalf("GetStats error: %v", err)
	}

	if report.TodaySessions != 0 || report.TodayRounds != 0 {
		t.Fatalf("expected 0 sessions and 0 rounds, got %d sessions %d rounds", report.TodaySessions, report.TodayRounds)
	}
	if report.CurrentStreak != 0 {
		t.Fatalf("expected 0 streak, got %d", report.CurrentStreak)
	}
}

func TestCleanupUnfinishedSessions(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	// Simulate a crash where session was left "active"
	_, err := store.CreateSession(ctx, 3, time.Now())
	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}

	if err := store.CleanupUnfinishedSessions(ctx); err != nil {
		t.Fatalf("CleanupUnfinishedSessions: %v", err)
	}

	var status string
	err = store.DB().GetContext(ctx, &status, `SELECT status FROM sessions LIMIT 1`)
	if err != nil {
		t.Fatalf("Query status: %v", err)
	}
	if status != "abandoned" {
		t.Fatalf("expected abandoned, got %s", status)
	}
}

func TestRollbackLastRound(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	sessID, err := store.CreateSession(ctx, 3, time.Now())
	if err != nil {
		t.Fatalf("CreateSession error: %v", err)
	}

	// Save Round 1
	err = store.SaveRound(ctx, sessID, session.RoundResult{
		Index:     1,
		Breathing: 3 * time.Minute,
		Retention: 45 * time.Second,
		Recovery:  15 * time.Second,
	})
	if err != nil {
		t.Fatalf("SaveRound 1 error: %v", err)
	}

	// Save Round 2
	err = store.SaveRound(ctx, sessID, session.RoundResult{
		Index:     2,
		Breathing: 3 * time.Minute,
		Retention: 60 * time.Second,
		Recovery:  15 * time.Second,
	})
	if err != nil {
		t.Fatalf("SaveRound 2 error: %v", err)
	}

	var count int
	_ = store.DB().GetContext(ctx, &count, `SELECT completed_rounds FROM sessions WHERE id = ?`, sessID)
	if count != 2 {
		t.Fatalf("expected completed_rounds = 2, got %d", count)
	}

	// Rollback Round 2
	err = store.RollbackLastRound(ctx, sessID)
	if err != nil {
		t.Fatalf("RollbackLastRound error: %v", err)
	}

	_ = store.DB().GetContext(ctx, &count, `SELECT completed_rounds FROM sessions WHERE id = ?`, sessID)
	if count != 1 {
		t.Fatalf("expected completed_rounds = 1 after rollback, got %d", count)
	}

	var roundCount int
	_ = store.DB().GetContext(ctx, &roundCount, `SELECT COUNT(*) FROM rounds WHERE session_id = ?`, sessID)
	if roundCount != 1 {
		t.Fatalf("expected 1 round in rounds table, got %d", roundCount)
	}

	// Rollback Round 1
	err = store.RollbackLastRound(ctx, sessID)
	if err != nil {
		t.Fatalf("RollbackLastRound 2nd error: %v", err)
	}

	_ = store.DB().GetContext(ctx, &count, `SELECT completed_rounds FROM sessions WHERE id = ?`, sessID)
	if count != 0 {
		t.Fatalf("expected completed_rounds = 0 after 2nd rollback, got %d", count)
	}
}

func TestStreakCalculation(t *testing.T) {
	refTime := time.Date(2026, 9, 13, 12, 0, 0, 0, time.Local) // 2026-09-13 is today

	tests := []struct {
		name        string
		dates       []string
		wantCurrent int
		wantBest    int
	}{
		{
			name:        "empty",
			dates:       nil,
			wantCurrent: 0,
			wantBest:    0,
		},
		{
			name:        "active today only",
			dates:       []string{"2026-09-13"},
			wantCurrent: 1,
			wantBest:    1,
		},
		{
			name:        "active yesterday only (today pending)",
			dates:       []string{"2026-09-12"},
			wantCurrent: 1,
			wantBest:    1,
		},
		{
			name:        "active 3 days ending today",
			dates:       []string{"2026-09-11", "2026-09-12", "2026-09-13"},
			wantCurrent: 3,
			wantBest:    3,
		},
		{
			name:        "active 3 days ending yesterday",
			dates:       []string{"2026-09-10", "2026-09-11", "2026-09-12"},
			wantCurrent: 3,
			wantBest:    3,
		},
		{
			name:        "streak broken 2 days ago",
			dates:       []string{"2026-09-08", "2026-09-09", "2026-09-10"},
			wantCurrent: 0,
			wantBest:    3,
		},
		{
			name:        "historic best longer than current",
			dates:       []string{"2026-08-01", "2026-08-02", "2026-08-03", "2026-08-04", "2026-08-05", "2026-09-12", "2026-09-13"},
			wantCurrent: 2,
			wantBest:    5,
		},
		{
			name:        "duplicates and unsorted",
			dates:       []string{"2026-09-13", "2026-09-11", "2026-09-12", "2026-09-12", "2026-09-13"},
			wantCurrent: 3,
			wantBest:    3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotCurrent, gotBest := CalculateStreak(tt.dates, refTime)
			if gotCurrent != tt.wantCurrent || gotBest != tt.wantBest {
				t.Fatalf("CalculateStreak() = (%d, %d), want (%d, %d)", gotCurrent, gotBest, tt.wantCurrent, tt.wantBest)
			}
		})
	}
}

func TestDeleteSessionByOffsetAndAll(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	// Initially empty -> error
	if _, err := store.DeleteSessionByOffset(ctx, 0); err == nil {
		t.Fatal("expected error deleting from empty store")
	}

	// Create 3 sessions
	t1 := time.Date(2026, 9, 13, 10, 0, 0, 0, time.Local)
	s1, _ := store.CreateSession(ctx, 2, t1)
	_ = store.SaveRound(ctx, s1, session.RoundResult{Index: 1, Retention: 30 * time.Second})
	_ = store.EndSession(ctx, s1, t1.Add(5*time.Minute), 5*time.Minute, 5*time.Minute, "completed")

	t2 := time.Date(2026, 9, 13, 12, 0, 0, 0, time.Local)
	s2, _ := store.CreateSession(ctx, 2, t2)
	_ = store.SaveRound(ctx, s2, session.RoundResult{Index: 1, Retention: 40 * time.Second})
	_ = store.EndSession(ctx, s2, t2.Add(5*time.Minute), 5*time.Minute, 5*time.Minute, "completed")

	t3 := time.Date(2026, 9, 13, 14, 0, 0, 0, time.Local)
	s3, _ := store.CreateSession(ctx, 2, t3)
	_ = store.SaveRound(ctx, s3, session.RoundResult{Index: 1, Retention: 50 * time.Second})
	_ = store.EndSession(ctx, s3, t3.Add(5*time.Minute), 5*time.Minute, 5*time.Minute, "completed")

	// Delete offset ~1 (the past session, which is s2: 12:00)
	del, err := store.DeleteSessionByOffset(ctx, 1)
	if err != nil {
		t.Fatalf("DeleteSessionByOffset(1) error: %v", err)
	}
	if del.ID != s2 {
		t.Fatalf("expected deleted session ID to be %d, got %d", s2, del.ID)
	}

	// Now we have 2 sessions: s3 (offset 0) and s1 (offset 1)
	// Delete offset ~0 (the latest session, which is s3: 14:00)
	del, err = store.DeleteSessionByOffset(ctx, 0)
	if err != nil {
		t.Fatalf("DeleteSessionByOffset(0) error: %v", err)
	}
	if del.ID != s3 {
		t.Fatalf("expected deleted session ID to be %d, got %d", s3, del.ID)
	}

	// Only s1 remains
	report, err := store.GetStats(ctx, t3)
	if err != nil {
		t.Fatalf("GetStats error: %v", err)
	}
	if report.TotalSessions != 1 {
		t.Fatalf("expected 1 session remaining, got %d", report.TotalSessions)
	}

	// DeleteAllSessions
	count, err := store.DeleteAllSessions(ctx)
	if err != nil {
		t.Fatalf("DeleteAllSessions error: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected 1 session deleted by DeleteAllSessions, got %d", count)
	}

	// Verify completely empty
	report, err = store.GetStats(ctx, t3)
	if err != nil {
		t.Fatalf("GetStats error: %v", err)
	}
	if report.TotalSessions != 0 || report.TotalRounds != 0 {
		t.Fatalf("expected empty stats after DeleteAllSessions, got %#v", report)
	}
}
