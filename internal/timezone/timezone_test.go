package timezone

import (
	"os"
	"testing"
	"time"
)

func TestDetectWithTZEnv(t *testing.T) {
	origTZ := os.Getenv("TZ")
	defer func() {
		_ = os.Setenv("TZ", origTZ)
		_, _ = DetectAndSet("")
	}()

	_ = os.Setenv("TZ", "America/Sao_Paulo")
	loc, name, err := Detect()
	if err != nil {
		t.Fatalf("expected no error detecting with TZ, got %v", err)
	}
	if name != "America/Sao_Paulo" {
		t.Errorf("expected America/Sao_Paulo, got %s", name)
	}
	if loc.String() != "America/Sao_Paulo" {
		t.Errorf("expected loc string America/Sao_Paulo, got %s", loc.String())
	}
}

func TestSetExplicitTimezone(t *testing.T) {
	origLoc := time.Local
	defer func() {
		time.Local = origLoc
	}()

	loc, err := Set("Asia/Tokyo")
	if err != nil {
		t.Fatalf("unexpected error setting Asia/Tokyo: %v", err)
	}
	if loc.String() != "Asia/Tokyo" {
		t.Errorf("expected Asia/Tokyo, got %s", loc.String())
	}
	if time.Local != loc {
		t.Errorf("expected time.Local to be set to %v", loc)
	}

	curLoc, curName := Current()
	if curLoc != loc || curName != "Asia/Tokyo" {
		t.Errorf("expected Current() to return Asia/Tokyo, got %v, %s", curLoc, curName)
	}

	_, err = Set("Invalid/NonExistent_Zone")
	if err == nil {
		t.Error("expected error setting invalid timezone, got nil")
	}
}

func TestMidnightCrossingLocalDate(t *testing.T) {
	// Reproduce the issue: session done at 22:30 BRT (01:30 UTC next day)
	loc, err := time.LoadLocation("America/Sao_Paulo")
	if err != nil {
		t.Fatalf("failed to load America/Sao_Paulo: %v", err)
	}

	utcTime1, err := time.Parse(time.RFC3339, "2026-10-03T01:30:29Z")
	if err != nil {
		t.Fatalf("failed to parse utcTime1: %v", err)
	}

	localDate1 := utcTime1.In(loc).Format("2006-01-02")
	if localDate1 != "2026-10-02" {
		t.Errorf("expected local date 2026-10-02 for 01:30Z in Sao Paulo, got %s", localDate1)
	}

	utcTime2, err := time.Parse(time.RFC3339, "2026-10-03T13:03:20Z")
	if err != nil {
		t.Fatalf("failed to parse utcTime2: %v", err)
	}

	localDate2 := utcTime2.In(loc).Format("2006-01-02")
	if localDate2 != "2026-10-03" {
		t.Errorf("expected local date 2026-10-03 for 13:03Z in Sao Paulo, got %s", localDate2)
	}
}
