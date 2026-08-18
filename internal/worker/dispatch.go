package worker

import "task114-jobsched/internal/model"

func SortJobs(jobs []*model.Job) {
	for i := 1; i < len(jobs); i++ {
		current := jobs[i]
		j := i - 1
		for j >= 0 && jobs[j].Priority < current.Priority {
			jobs[j+1] = jobs[j]
			j--
		}
		jobs[j+1] = current
	}
}
