package worker

import (
	"context"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"task114-jobsched/internal/clock"
	"task114-jobsched/internal/model"
	"task114-jobsched/internal/store"
)

func newPool(t *testing.T) (*store.Store, *Pool) {
	t.Helper()
	s, err := store.Open(filepath.Join(t.TempDir(), "w.db"))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	p := New(s, clock.New())
	p.RegisterHandler("noop", func(ctx context.Context, j *model.Job) (string, error) {
		return "", nil
	})
	p.RegisterHandler("echo", func(ctx context.Context, j *model.Job) (string, error) {
		return j.Args, nil
	})
	return s, p
}

func TestFlushSucceeds(t *testing.T) {
	s, p := newPool(t)
	now := time.Now()
	for i := 0; i < 4; i++ {
		j := &model.Job{
			ID:          "j" + string(rune('0'+i)),
			Queue:       "q",
			Type:        "echo",
			Args:        `{"i":` + string(rune('0'+i)) + `}`,
			State:       model.StatePending,
			RunAt:       now,
			MaxAttempts: 3,
		}
		if err := s.CreateJob(j); err != nil {
			t.Fatal(err)
		}
	}
	if err := p.Flush(context.Background()); err != nil {
		t.Fatalf("flush: %v", err)
	}
	p.Wait()
	for i := 0; i < 4; i++ {
		j, err := s.GetJob("j" + string(rune('0'+i)))
		if err != nil {
			t.Fatal(err)
		}
		if j.State != model.StateSucceeded {
			t.Fatalf("job %s expected succeeded, got %s", j.ID, j.State)
		}
		if j.Result != `{"i":`+string(rune('0'+i))+`}` {
			t.Fatalf("job %s wrong result: %q", j.ID, j.Result)
		}
	}
}

func TestFlushSkipsFuture(t *testing.T) {
	s, p := newPool(t)
	now := time.Now()
	due := &model.Job{ID: "due", Queue: "q", Type: "noop", State: model.StatePending, RunAt: now, MaxAttempts: 1}
	future := &model.Job{ID: "future", Queue: "q", Type: "noop", State: model.StateScheduled, RunAt: now.Add(time.Hour), MaxAttempts: 1}
	if err := s.CreateJob(due); err != nil {
		t.Fatal(err)
	}
	if err := s.CreateJob(future); err != nil {
		t.Fatal(err)
	}
	if err := p.Flush(context.Background()); err != nil {
		t.Fatal(err)
	}
	p.Wait()
	d, _ := s.GetJob("due")
	if d.State != model.StateSucceeded {
		t.Fatalf("due job expected succeeded, got %s", d.State)
	}
	f, _ := s.GetJob("future")
	if f.State != model.StateScheduled {
		t.Fatalf("future job expected scheduled, got %s", f.State)
	}
}

// TestFireDueSchedulesRejectsCorruptArgs covers the non-HTTP entry point
// described by bug 07: a recurring schedule fires jobs via CreateJob directly,
// bypassing the HTTP JSON decoder. A schedule whose args are not valid JSON
// must not be able to persist a job — the worker would only discover the
// unparseable payload after claiming it.
func TestFireDueSchedulesRejectsCorruptArgs(t *testing.T) {
	s, p := newPool(t)

	bad := &model.Schedule{
		ID:          "bad-sched",
		Queue:       "q",
		Type:        "noop",
		Args:        `{"broken"`,
		Interval:    time.Second,
		Enabled:     true,
		MaxAttempts: 3,
	}
	if err := s.CreateSchedule(bad); err != nil {
		t.Fatalf("create bad schedule: %v", err)
	}
	// A valid schedule must still fire through the same path so the rejection
	// does not silently break recurring scheduling.
	good := &model.Schedule{
		ID:          "good-sched",
		Queue:       "q",
		Type:        "noop",
		Args:        `{"ok":true}`,
		Interval:    time.Second,
		Enabled:     true,
		MaxAttempts: 3,
	}
	if err := s.CreateSchedule(good); err != nil {
		t.Fatalf("create good schedule: %v", err)
	}

	p.fireDueSchedules(context.Background())

	jobs, err := s.ListJobs(store.ListFilter{})
	if err != nil {
		t.Fatalf("list jobs: %v", err)
	}
	var goodFired bool
	for _, j := range jobs {
		if strings.HasPrefix(j.ID, "sched-bad-sched-") {
			t.Errorf("corrupt-args schedule persisted a job: %s args=%q", j.ID, j.Args)
		}
		if !json.Valid([]byte(j.Args)) {
			t.Errorf("job %s has invalid JSON args: %q", j.ID, j.Args)
		}
		if strings.HasPrefix(j.ID, "sched-good-sched-") {
			goodFired = true
		}
	}
	if !goodFired {
		t.Fatal("valid schedule did not fire; rejection must not break the happy path")
	}
}
