package worker

import (
	"context"
	"testing"
	"time"

	"task114-jobsched/internal/model"
	"task114-jobsched/internal/store"
)

func TestBug05_NewScheduleCreatesExactlyOneCurrentRun(t *testing.T) {
	s, p := newPool(t)
	before := time.Now()
	sc := &model.Schedule{ID: "first-schedule", Queue: "q", Type: "noop", Args: "{}", Interval: time.Minute, Enabled: true, MaxAttempts: 1}
	if err := s.CreateSchedule(sc); err != nil {
		t.Fatalf("create schedule: %v", err)
	}
	p.fireDueSchedules(context.Background())
	jobs, err := s.ListJobs(store.ListFilter{})
	if err != nil {
		t.Fatalf("list jobs: %v", err)
	}
	if len(jobs) != 1 {
		t.Fatalf("first schedule fire created %d jobs, want exactly one", len(jobs))
	}
	if jobs[0].RunAt.Before(before) {
		t.Fatalf("first schedule run was backdated to %s", jobs[0].RunAt)
	}
}

// TestBug05_BackloggedScheduleBackfillsMissedPeriods is the other half of the
// bug 05 contract: a schedule that genuinely fell behind (it fired before, then
// several periods elapsed while no job was created) must still backfill one job
// per missed period. Only a first fire collapses to a single current run.
func TestBug05_BackloggedScheduleBackfillsMissedPeriods(t *testing.T) {
	s, p := newPool(t)
	now := time.Now()
	interval := time.Minute
	// The schedule already fired three intervals ago and was neglected, so the
	// three elapsed periods are real backlog, not a first-fire burst.
	sc := &model.Schedule{
		ID:          "backlogged",
		Queue:       "q",
		Type:        "noop",
		Args:        "{}",
		Interval:    interval,
		Enabled:     true,
		MaxAttempts: 1,
		LastRun:     now.Add(-3 * interval),
	}
	if err := s.CreateSchedule(sc); err != nil {
		t.Fatalf("create schedule: %v", err)
	}
	p.fireDueSchedules(context.Background())
	jobs, err := s.ListJobs(store.ListFilter{})
	if err != nil {
		t.Fatalf("list jobs: %v", err)
	}
	if len(jobs) != 3 {
		t.Fatalf("backlogged schedule created %d jobs, want 3 backfilled periods", len(jobs))
	}
	// Every backfilled run must be due at or before now; catch-up never schedules
	// a job into the future.
	for _, j := range jobs {
		if j.RunAt.After(now.Add(time.Second)) {
			t.Fatalf("backfilled job %s run_at %s is in the future", j.ID, j.RunAt)
		}
	}
}
