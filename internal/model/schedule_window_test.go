package model

import (
	"testing"
	"time"
)

// TestScheduleWindowFirstFireVsBacklog pins the contract that lets the worker
// tell a first fire apart from a genuinely backlogged schedule. The store must
// preserve a never-fired schedule's zero LastRun across a round-trip so that
// these methods keep reporting "never fired" instead of centuries of backlog.
func TestScheduleWindowFirstFireVsBacklog(t *testing.T) {
	now := time.Date(2026, 8, 20, 12, 0, 0, 0, time.UTC)
	interval := time.Minute

	never := Schedule{Enabled: true, Interval: interval, LastRun: time.Time{}}
	if got := never.MissedRuns(now); got != 0 {
		t.Fatalf("never-fired MissedRuns = %d, want 0", got)
	}
	if got := never.NextRun(now); !got.Equal(now) {
		t.Fatalf("never-fired NextRun = %s, want %s", got, now)
	}

	// Three whole periods elapsed since the last fire: real backlog.
	backlogged := Schedule{Enabled: true, Interval: interval, LastRun: now.Add(-3 * interval)}
	if got := backlogged.MissedRuns(now); got != 3 {
		t.Fatalf("backlogged MissedRuns = %d, want 3", got)
	}
	if got := backlogged.NextRun(now); !got.Equal(now.Add(-2 * interval)) {
		t.Fatalf("backlogged NextRun = %s, want %s", got, now.Add(-2*interval))
	}

	// A schedule that has fired but whose next period has not elapsed yet is
	// neither first-fire nor backlog: nothing is missed and the next run is in
	// the future.
	recurring := Schedule{Enabled: true, Interval: interval, LastRun: now.Add(-30 * time.Second)}
	if got := recurring.MissedRuns(now); got != 0 {
		t.Fatalf("recurring MissedRuns = %d, want 0", got)
	}
	if got := recurring.NextRun(now); !got.Equal(now.Add(30 * time.Second)) {
		t.Fatalf("recurring NextRun = %s, want %s", got, now.Add(30*time.Second))
	}

	// A disabled schedule reports no missed runs regardless of timing.
	disabled := Schedule{Enabled: false, Interval: interval, LastRun: now.Add(-24 * time.Hour)}
	if got := disabled.MissedRuns(now); got != 0 {
		t.Fatalf("disabled MissedRuns = %d, want 0", got)
	}
}
