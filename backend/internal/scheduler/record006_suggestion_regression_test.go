package scheduler

import (
	"testing"
	"time"

	"satellite-contact-window-deconfliction/backend/internal/constants"
	"satellite-contact-window-deconfliction/backend/internal/model"
)

func TestNoLockedAlternateP601(t *testing.T) {
	base := time.Date(2026, 8, 23, 10, 0, 0, 0, time.UTC)
	station := model.GroundStation{ID: 1, StationCode: "GS-A", Latitude: 0, Longitude: 0, AntennaCount: 1, SupportedBandsJSON: `["S"]`, StationStatus: "active"}
	asset := model.SatelliteAsset{ID: 1, SatelliteCode: "SAT", PriorityWeight: 5, SupportedBandsJSON: `["S"]`}
	affected := model.ContactWindow{ID: 1, StationID: 1, SatelliteID: 1, StartAt: base, EndAt: base.Add(10 * time.Minute), Band: "S", SourceVersion: "v1", Priority: 5, Version: 1}
	lockedAlt := model.ContactWindow{ID: 2, StationID: 1, SatelliteID: 1, StartAt: base.Add(30 * time.Minute), EndAt: base.Add(40 * time.Minute), Band: "S", SourceVersion: "v1", Priority: 3, Locked: true, Version: 1}
	group := newGroup(constants.ConflictTypeStationCapacity, []model.ContactWindow{affected}, 1, 1, 0, "capacity", nil)
	generator := NewCandidateGenerator(Weights{PriorityLoss: 4, MovementDistance: .02, ContactDuration: .003, ResourceMargin: 2}, []model.GroundStation{station}, []model.SatelliteAsset{asset}, []model.ContactWindow{affected, lockedAlt})
	for _, suggestion := range generator.Generate(group) {
		if suggestion.AlternateWindowID != nil && *suggestion.AlternateWindowID == lockedAlt.ID {
			t.Fatalf("suggestion proposed locked window %d as alternate", lockedAlt.ID)
		}
	}
}

func TestNoOverCapacityRelocationP602(t *testing.T) {
	base := time.Date(2026, 8, 23, 10, 0, 0, 0, time.UTC)
	stationA := model.GroundStation{ID: 1, StationCode: "GS-A", Latitude: 0, Longitude: 0, AntennaCount: 1, SupportedBandsJSON: `["S"]`, StationStatus: "active"}
	stationB := model.GroundStation{ID: 2, StationCode: "GS-B", Latitude: 1, Longitude: 1, AntennaCount: 1, SupportedBandsJSON: `["S"]`, StationStatus: "active"}
	asset := model.SatelliteAsset{ID: 1, SatelliteCode: "SAT", PriorityWeight: 5, SupportedBandsJSON: `["S"]`}
	affected := model.ContactWindow{ID: 1, StationID: 1, SatelliteID: 1, StartAt: base, EndAt: base.Add(10 * time.Minute), Band: "S", SourceVersion: "v1", Priority: 5, Version: 1}
	occupiesB := model.ContactWindow{ID: 2, StationID: 2, SatelliteID: 1, StartAt: base.Add(time.Minute), EndAt: base.Add(9 * time.Minute), Band: "S", SourceVersion: "v1", Priority: 4, Version: 1}
	group := newGroup(constants.ConflictTypeStationCapacity, []model.ContactWindow{affected, occupiesB}, 1, 2, 0, "capacity", nil)
	generator := NewCandidateGenerator(Weights{PriorityLoss: 4, MovementDistance: .02, ContactDuration: .003, ResourceMargin: 2}, []model.GroundStation{stationA, stationB}, []model.SatelliteAsset{asset}, []model.ContactWindow{affected, occupiesB})
	for _, suggestion := range generator.Generate(group) {
		if suggestion.TargetStationID != nil && *suggestion.TargetStationID == stationB.ID {
			t.Fatalf("suggestion relocated to over-capacity station %d", stationB.ID)
		}
	}
}

func TestAlternateNearestP604(t *testing.T) {
	base := time.Date(2026, 8, 23, 10, 0, 0, 0, time.UTC)
	station := model.GroundStation{ID: 1, StationCode: "GS-A", Latitude: 0, Longitude: 0, AntennaCount: 1, SupportedBandsJSON: `["S"]`, StationStatus: "active"}
	asset := model.SatelliteAsset{ID: 1, SatelliteCode: "SAT", PriorityWeight: 5, SupportedBandsJSON: `["S"]`}
	affected := model.ContactWindow{ID: 1, StationID: 1, SatelliteID: 1, StartAt: base, EndAt: base.Add(10 * time.Minute), Band: "S", SourceVersion: "v1", Priority: 5, Version: 1}
	altNear := model.ContactWindow{ID: 2, StationID: 1, SatelliteID: 1, StartAt: base.Add(30 * time.Minute), EndAt: base.Add(40 * time.Minute), Band: "S", SourceVersion: "v1", Priority: 3, Version: 1}
	altFar := model.ContactWindow{ID: 3, StationID: 1, SatelliteID: 1, StartAt: base.Add(90 * time.Minute), EndAt: base.Add(100 * time.Minute), Band: "S", SourceVersion: "v1", Priority: 3, Version: 1}
	group := newGroup(constants.ConflictTypeStationCapacity, []model.ContactWindow{affected}, 1, 1, 0, "capacity", nil)
	generator := NewCandidateGenerator(Weights{PriorityLoss: 4, MovementDistance: .02, ContactDuration: .003, ResourceMargin: 2}, []model.GroundStation{station}, []model.SatelliteAsset{asset}, []model.ContactWindow{affected, altNear, altFar})
	var chosen *uint
	for _, suggestion := range generator.Generate(group) {
		if suggestion.AlternateWindowID != nil {
			chosen = suggestion.AlternateWindowID
			break
		}
	}
	if chosen == nil || *chosen != altNear.ID {
		t.Fatalf("expected nearest alternate window %d, got %v", altNear.ID, chosen)
	}
}

func TestScoreFormulaP603(t *testing.T) {
	weights := Weights{PriorityLoss: 4, MovementDistance: .02, ContactDuration: .003, ResourceMargin: 2}
	got := Score(weights, 10, 0, 60, 0)
	want := 60*0.003 - 10*4
	if got.TotalScore != want {
		t.Fatalf("score total = %v, want %v", got.TotalScore, want)
	}
}
