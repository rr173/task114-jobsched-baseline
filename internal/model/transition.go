package model

func CanCancel(state State) bool {
	return state == StatePending || state == StateScheduled || state == StateRunning
}
func CanRetry(state State) bool {
	return state == StateDead || state == StateRunning || state == StatePending
}

// CanTransition reports whether a general persistence update may move a job
// from one lifecycle state to another. Terminal outcomes are immutable; the
// dedicated requeue and cancellation operations own their explicit paths.
func CanTransition(from, to State) bool {
	if from == StateSucceeded || from == StateDead || from == StateCancelled {
		return from == to
	}
	return ValidStates[to]
}

func (j Job) Retryable() bool {
	return !j.IsTerminal() && j.Attempts < j.MaxAttempts
}
