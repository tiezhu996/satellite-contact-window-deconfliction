package constants

const (
	ConflictTypeStationCapacity   = "station_capacity"
	ConflictTypeSatelliteOverlap  = "satellite_overlap"
	ConflictTypeBandMismatch      = "band_mismatch"
	ConflictTypeDurationShortfall = "duration_shortfall"
	ConflictTypeSlewBuffer        = "slew_buffer"
)

var ConflictTypes = []string{
	ConflictTypeStationCapacity,
	ConflictTypeSatelliteOverlap,
	ConflictTypeBandMismatch,
	ConflictTypeDurationShortfall,
	ConflictTypeSlewBuffer,
}

const (
	ResolutionStatusDetected      = "detected"
	ResolutionStatusProposed      = "proposed"
	ResolutionStatusPendingReview = "pending_review"
	ResolutionStatusAccepted      = "accepted"
	ResolutionStatusRejected      = "rejected"
)

func CanTransitionResolution(from, to string) bool {
	if from == ResolutionStatusDetected {
		if to == ResolutionStatusProposed {
			return true
		}
		if to == ResolutionStatusAccepted {
			return true
		}
		return false
	}
	if from == ResolutionStatusProposed {
		if to == ResolutionStatusPendingReview {
			return true
		}
		return false
	}
	if from == ResolutionStatusPendingReview {
		if to == ResolutionStatusAccepted {
			return true
		}
		if to == ResolutionStatusRejected {
			return true
		}
		return false
	}
	if from == ResolutionStatusAccepted {
		if to == ResolutionStatusProposed {
			return true
		}
		return false
	}
	return false
}
