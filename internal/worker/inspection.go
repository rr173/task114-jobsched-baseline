package worker

import (
	"task114-jobsched/internal/model"
	"time"
)

type Inspection struct {
	Active   int       `json:"active"`
	Queued   int       `json:"queued"`
	Captured time.Time `json:"captured"`
}

func (p *Pool) Inspect() Inspection { return Inspection{Active: p.Active(), Captured: time.Now()} }
func Due(j *model.Job, now time.Time) bool {
	return j != nil && j.IsDue(now) && j.State != model.StateCancelled
}
