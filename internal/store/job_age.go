package store

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"task114-jobsched/internal/model"
)

func (s *Store) OldestPending(queue string) (*model.Job, error) {
	rows, err := s.ListJobs(ListFilter{Queue: queue, State: model.StatePending, Limit: 1})
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, ErrNotFound
	}
	return &rows[0], nil
}

// OldestReady returns the earliest processable job at now: a pending or
// scheduled job whose run_at has arrived, ordered the same way workers claim
// them (priority desc, then earliest run_at). A scheduled job that is already
// due is just as executable as a pending one, so it must be considered when
// reporting the real earliest executable time. Returns ErrNotFound when no
// job is currently due.
func (s *Store) OldestReady(queue string, now time.Time) (*model.Job, error) {
	q := `SELECT ` + jobColumns + ` FROM jobs
		WHERE state IN ('pending','scheduled') AND run_at <= ?`
	args := []interface{}{now.UnixNano()}
	if queue != "" {
		q += ` AND queue=?`
		args = append(args, queue)
	}
	q += ` ORDER BY priority DESC, run_at ASC, id ASC LIMIT 1`
	row := s.db.QueryRow(q, args...)
	j, err := s.scanJob(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("oldest ready: %w", err)
	}
	return j, nil
}
func (s *Store) DueCount(queue string, now time.Time) (int, error) {
	rows, err := s.ScanDue(10000, now)
	if err != nil {
		return 0, err
	}
	n := 0
	for _, j := range rows {
		if queue == "" || j.Queue == queue {
			n++
		}
	}
	return n, nil
}
