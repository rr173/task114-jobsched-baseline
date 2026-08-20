package store

import (
	"path/filepath"
	"testing"
	"time"

	"task114-jobsched/internal/model"
)

func openTest(t *testing.T) *Store {
	t.Helper()
	path := filepath.Join(t.TempDir(), "test.db")
	s, err := Open(path)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

func newJob(id, queue, typ string, runAt time.Time) *model.Job {
	return &model.Job{
		ID:          id,
		Queue:       queue,
		Type:        typ,
		Args:        "{}",
		State:       model.StatePending,
		RunAt:       runAt,
		MaxAttempts: 3,
		Priority:    0,
	}
}

func TestCreateAndGet(t *testing.T) {
	s := openTest(t)
	j := newJob("j1", "q1", "noop", time.Now())
	if err := s.CreateJob(j); err != nil {
		t.Fatalf("create: %v", err)
	}
	got, err := s.GetJob("j1")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Queue != "q1" || got.Type != "noop" {
		t.Fatalf("unexpected job: %+v", got)
	}
}

func TestScanDue(t *testing.T) {
	s := openTest(t)
	now := time.Now()
	due := newJob("due", "q", "noop", now.Add(-time.Minute))
	future := newJob("future", "q", "noop", now.Add(time.Hour))
	if err := s.CreateJob(due); err != nil {
		t.Fatal(err)
	}
	if err := s.CreateJob(future); err != nil {
		t.Fatal(err)
	}
	got, err := s.ScanDue(10, now)
	if err != nil {
		t.Fatalf("scan: %v", err)
	}
	if len(got) != 1 || got[0].ID != "due" {
		t.Fatalf("expected only due job, got %v", got)
	}
}

func TestSucceed(t *testing.T) {
	s := openTest(t)
	j := newJob("j1", "q", "noop", time.Now())
	if err := s.CreateJob(j); err != nil {
		t.Fatal(err)
	}
	if err := s.Succeed("j1", "ok"); err != nil {
		t.Fatalf("succeed: %v", err)
	}
	got, _ := s.GetJob("j1")
	if got.State != model.StateSucceeded {
		t.Fatalf("expected succeeded, got %s", got.State)
	}
}

// TestSucceedAndFailDoNotOverwriteTerminal guards the cancel/complete race: a
// late completion (or retry) arriving after the job reached a terminal state
// must be discarded, not written back over the terminal outcome.
func TestSucceedAndFailDoNotOverwriteTerminal(t *testing.T) {
	for _, terminal := range []model.State{model.StateCancelled, model.StateDead, model.StateSucceeded} {
		s := openTest(t)
		j := newJob("j1", "q", "noop", time.Now())
		j.State = terminal
		j.Attempts = 1
		if err := s.CreateJob(j); err != nil {
			t.Fatal(err)
		}

		if err := s.Succeed("j1", "late success"); err != nil {
			t.Fatalf("Succeed(%s): %v", terminal, err)
		}
		if err := s.Fail("j1", "late failure", true, time.Now().Add(time.Second)); err != nil {
			t.Fatalf("Fail(%s, retry): %v", terminal, err)
		}
		if err := s.Fail("j1", "late failure", false, time.Time{}); err != nil {
			t.Fatalf("Fail(%s, dead): %v", terminal, err)
		}

		got, _ := s.GetJob("j1")
		if got.State != terminal {
			t.Fatalf("terminal %s overwritten by late completion: got %s", terminal, got.State)
		}
		if got.Attempts != 1 {
			t.Fatalf("terminal %s attempt counter mutated by late failure: got %d", terminal, got.Attempts)
		}
	}
}

func TestRecordAttempts(t *testing.T) {
	s := openTest(t)
	j := newJob("j1", "q", "noop", time.Now())
	if err := s.CreateJob(j); err != nil {
		t.Fatal(err)
	}
	_ = s.RecordAttempt(model.Attempt{JobID: "j1", Index: 0, StartedAt: time.Now(), EndedAt: time.Now()})
	attempts, err := s.ListAttempts("j1")
	if err != nil {
		t.Fatalf("list attempts: %v", err)
	}
	if len(attempts) != 1 {
		t.Fatalf("expected 1 attempt, got %d", len(attempts))
	}
}

func TestRequeueDead(t *testing.T) {
	s := openTest(t)
	j := newJob("j1", "q", "noop", time.Now())
	if err := s.CreateJob(j); err != nil {
		t.Fatal(err)
	}
	// Mark the job dead directly (no failure path involved) then requeue it.
	j.State = model.StateDead
	if err := s.UpdateJob(j); err != nil {
		t.Fatal(err)
	}
	if err := s.RequeueDead("j1"); err != nil {
		t.Fatalf("requeue dead: %v", err)
	}
	got, _ := s.GetJob("j1")
	if got.State != model.StatePending {
		t.Fatalf("requeue should move to pending, got %s", got.State)
	}
}

func TestQueuePause(t *testing.T) {
	s := openTest(t)
	if err := s.SetPaused("q1", true); err != nil {
		t.Fatalf("set paused: %v", err)
	}
	paused, err := s.IsPaused("q1")
	if err != nil {
		t.Fatal(err)
	}
	if !paused {
		t.Fatal("expected q1 paused")
	}
}

func TestListJobsLimit(t *testing.T) {
	s := openTest(t)
	now := time.Now()
	for i := 0; i < 5; i++ {
		j := newJob(string(rune('a'+i)), "q", "noop", now)
		j.ID = "id" + string(rune('0'+i))
		if err := s.CreateJob(j); err != nil {
			t.Fatal(err)
		}
	}
	got, err := s.ListJobs(ListFilter{Limit: 3})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("expected 3 jobs, got %d", len(got))
	}
}

func TestStats(t *testing.T) {
	s := openTest(t)
	for i := 0; i < 3; i++ {
		j := newJob("s"+string(rune('0'+i)), "q", "noop", time.Now())
		if err := s.CreateJob(j); err != nil {
			t.Fatal(err)
		}
	}
	st, err := s.Stats()
	if err != nil {
		t.Fatalf("stats: %v", err)
	}
	if st.Total != 3 {
		t.Fatalf("expected total 3, got %d", st.Total)
	}
}
