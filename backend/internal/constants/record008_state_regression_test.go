package constants

import (
	"testing"
)

func TestCandidateWindowCanSubmitP801(t *testing.T) {
	if !CanTransitionWindow(WindowStatusCandidate, WindowStatusSubmitted) {
		t.Fatal("candidate window must be submittable")
	}
}

func TestSubmittedWindowCanLockP802(t *testing.T) {
	if !CanTransitionWindow(WindowStatusSubmitted, WindowStatusLocked) {
		t.Fatal("submitted window must be lockable")
	}
}

func TestAcceptedCannotRollbackP803(t *testing.T) {
	if CanTransitionResolution(ResolutionStatusAccepted, ResolutionStatusProposed) {
		t.Fatal("accepted resolution must not roll back to proposed")
	}
}

func TestDetectedCannotJumpToAcceptedP804(t *testing.T) {
	if CanTransitionResolution(ResolutionStatusDetected, ResolutionStatusAccepted) {
		t.Fatal("detected resolution must not jump straight to accepted")
	}
}

func TestCandidateWindowCanCancelP807(t *testing.T) {
	if !CanTransitionWindow(WindowStatusCandidate, WindowStatusCancelled) {
		t.Fatal("candidate window must be cancellable")
	}
}

func TestSubmittedWindowCanCancelP808(t *testing.T) {
	if !CanTransitionWindow(WindowStatusSubmitted, WindowStatusCancelled) {
		t.Fatal("submitted window must be cancellable")
	}
}
