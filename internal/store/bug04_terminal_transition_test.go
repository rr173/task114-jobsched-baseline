package store

import (
	"errors"
	"testing"
	"time"

	"task114-jobsched/internal/model"
)

// TestBug04_TerminalJobCannotBeReopenedThroughUpdate asserts that the generic
// UpdateJob flow cannot roll a finished job back into the scheduling pool. A
// job that reached any terminal state (succeeded, dead, cancelled) is frozen
// for the generic update path; only explicit recovery (RequeueDead) may move
// it back to pending.
func TestBug04_TerminalJobCannotBeReopenedThroughUpdate(t *testing.T) {
	for _, tc := range []struct {
		name  string
		apply func(s *Store, id string) error
		want  model.State
	}{
		{
			name:  "succeeded",
			apply: func(s *Store, id string) error { return s.Succeed(id, "done") },
			want:  model.StateSucceeded,
		},
		{
			name: "dead",
			apply: func(s *Store, id string) error {
				return s.Fail(id, "boom", false, time.Now())
			},
			want: model.StateDead,
		},
		{
			name: "cancelled",
			apply: func(s *Store, id string) error {
				j, err := s.GetJob(id)
				if err != nil {
					return err
				}
				j.State = model.StateCancelled
				return s.UpdateJob(j) // pending -> cancelled is a forward transition, allowed
			},
			want: model.StateCancelled,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := openTest(t)
			j := newJob("terminal-"+tc.name, "q", "noop", time.Now())
			if err := s.CreateJob(j); err != nil {
				t.Fatalf("create: %v", err)
			}
			if err := tc.apply(s, j.ID); err != nil {
				t.Fatalf("seed terminal state: %v", err)
			}
			completed, err := s.GetJob(j.ID)
			if err != nil {
				t.Fatalf("get terminal: %v", err)
			}
			// Attempt to roll the terminal job back to pending via the generic
			// update flow, exactly as a naive caller would.
			completed.State = model.StatePending
			completed.RunAt = time.Now()
			err = s.UpdateJob(completed)
			if err == nil {
				t.Fatal("terminal job update unexpectedly reopened the job")
			}
			if !errors.Is(err, ErrTerminalState) {
				t.Fatalf("expected ErrTerminalState, got %v", err)
			}
			got, err := s.GetJob(j.ID)
			if err != nil {
				t.Fatalf("get after rejected update: %v", err)
			}
			if got.State != tc.want {
				t.Fatalf("terminal state changed from %q to %q", tc.want, got.State)
			}
		})
	}
}

// TestBug04_NonTerminalUpdateStillWorks guards against an over-eager fix: jobs
// that have not finished must still be mutable through UpdateJob (e.g. priority
// or run_at adjustments), and forward transitions into a terminal state remain
// valid.
func TestBug04_NonTerminalUpdateStillWorks(t *testing.T) {
	s := openTest(t)
	j := newJob("alive", "q", "noop", time.Now())
	if err := s.CreateJob(j); err != nil {
		t.Fatalf("create: %v", err)
	}
	got, err := s.GetJob(j.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	got.Priority = 5
	got.RunAt = got.RunAt.Add(time.Hour)
	if err := s.UpdateJob(got); err != nil {
		t.Fatalf("update pending job: %v", err)
	}
	refreshed, _ := s.GetJob(j.ID)
	if refreshed.Priority != 5 {
		t.Fatalf("priority not persisted: %d", refreshed.Priority)
	}
}

// TestBug04_RequeueDeadStillReopens confirms the explicit recovery path still
// moves a dead job back to pending, since retry depends on it.
func TestBug04_RequeueDeadStillReopens(t *testing.T) {
	s := openTest(t)
	j := newJob("dead-requeue", "q", "noop", time.Now())
	if err := s.CreateJob(j); err != nil {
		t.Fatalf("create: %v", err)
	}
	if err := s.Fail(j.ID, "boom", false, time.Now()); err != nil {
		t.Fatalf("fail: %v", err)
	}
	if err := s.RequeueDead(j.ID); err != nil {
		t.Fatalf("requeue dead: %v", err)
	}
	got, _ := s.GetJob(j.ID)
	if got.State != model.StatePending {
		t.Fatalf("requeue should move dead to pending, got %q", got.State)
	}
}
