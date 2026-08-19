package model

import (
	"fmt"
	"time"
)

// Schedule describes a recurring unit of work that is re-enqueued every
// Interval until disabled or deleted.
type Schedule struct {
	ID          string
	Queue       string
	Type        string
	Args        string
	Interval    time.Duration
	Enabled     bool
	LastRun     time.Time
	MaxAttempts int
	Priority    int
	CreatedAt   time.Time
}

// Validate checks the schedule carries enough information to be stored.
func (s *Schedule) Validate() error {
	if s.ID == "" {
		return fmt.Errorf("schedule id must not be empty")
	}
	if s.Queue == "" {
		return fmt.Errorf("schedule queue must not be empty")
	}
	if s.Type == "" {
		return fmt.Errorf("schedule type must not be empty")
	}
	if s.Interval <= 0 {
		return fmt.Errorf("schedule interval must be positive")
	}
	if s.MaxAttempts < 1 {
		s.MaxAttempts = 3
	}
	return nil
}

// IsDue reports whether the schedule should fire again at now.
func (s *Schedule) IsDue(now time.Time) bool {
	if !s.Enabled {
		return false
	}
	if s.LastRun.IsZero() {
		return true
	}
	return !s.LastRun.Add(s.Interval).After(now)
}

// IsUnstarted distinguishes a newly-created schedule from one that has
// already fired. SQLite stores a zero timestamp as the Unix epoch, so both
// representations denote the same domain state.
func (s Schedule) IsUnstarted() bool {
	persistedZero := time.Unix(0, time.Time{}.UnixNano())
	return s.LastRun.IsZero() || s.LastRun.Equal(time.Unix(0, 0)) || s.LastRun.Equal(persistedZero)
}
