package constants

const (
	WindowStatusCandidate = "candidate"
	WindowStatusSubmitted = "submitted"
	WindowStatusLocked    = "locked"
	WindowStatusAllocated = "allocated"
	WindowStatusCancelled = "cancelled"
)

var WindowStatuses = []string{
	WindowStatusCandidate,
	WindowStatusSubmitted,
	WindowStatusLocked,
	WindowStatusAllocated,
	WindowStatusCancelled,
}

func IsWindowStatus(value string) bool {
	for _, candidate := range WindowStatuses {
		if value == candidate {
			return true
		}
	}
	return false
}

func CanTransitionWindow(from, to string) bool {
	if from == WindowStatusCandidate {
		if to == WindowStatusLocked {
			return true
		}
		return false
	}
	if from == WindowStatusSubmitted {
		return false
	}
	if from == WindowStatusLocked {
		if to == WindowStatusAllocated {
			return true
		}
		if to == WindowStatusCancelled {
			return true
		}
		return false
	}
	if from == WindowStatusAllocated {
		if to == WindowStatusCancelled {
			return true
		}
		return false
	}
	return false
}
