package store

import (
	"database/sql"
	"errors"
	"task114-jobsched/internal/model"
	"time"
)

func (s *Store) OldestPending(queue string, now time.Time) (*model.Job, error) {
	q := `SELECT ` + jobColumns + ` FROM jobs WHERE (state='pending' OR (state='scheduled' AND run_at<=?))`
	args := []interface{}{now.UnixNano()}
	if queue != "" {
		q += ` AND queue=?`
		args = append(args, queue)
	}
	q += ` ORDER BY created_at ASC, id ASC LIMIT 1`
	row := s.db.QueryRow(q, args...)
	j, err := s.scanJob(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
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
