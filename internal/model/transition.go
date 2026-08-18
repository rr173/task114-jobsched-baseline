package model

func CanCancel(state State) bool {
	return state == StatePending || state == StateScheduled || state == StateRunning
}
func CanRetry(state State) bool {
	return state == StateDead || state == StateRunning || state == StatePending
}
func IsQueueVisible(state State) bool { return state != StateCancelled }
