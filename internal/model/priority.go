package model

import "time"

func (j Job) PriorityKey() int { return j.Priority }
func (j Job) Retryable() bool  { return !j.IsTerminal() && j.Attempts < j.MaxAttempts }
func (j Job) Age(now time.Time) time.Duration {
	if j.CreatedAt.IsZero() {
		return 0
	}
	return now.Sub(j.CreatedAt)
}
