package metrics

import "strconv"

func (s Snapshot) Text() string {
	return "enqueued=" + strconv.FormatInt(s.Enqueued, 10) + " succeeded=" + strconv.FormatInt(s.Succeeded, 10) + " failed=" + strconv.FormatInt(s.Failed, 10) + " retried=" + strconv.FormatInt(s.Retried, 10) + " dead=" + strconv.FormatInt(s.Dead, 10)
}
