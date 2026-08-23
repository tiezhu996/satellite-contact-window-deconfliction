package scheduler

import (
	"testing"
	"time"

	"satellite-contact-window-deconfliction/backend/internal/constants"
	"satellite-contact-window-deconfliction/backend/internal/model"
)

func TestSweepGroupsIndependent002(t *testing.T) {
	base := time.Date(2026, 8, 23, 10, 0, 0, 0, time.UTC)
	windows := []model.ContactWindow{
		{ID: 1, StartAt: base, EndAt: base.Add(10 * time.Minute)},
		{ID: 2, StartAt: base.Add(2 * time.Minute), EndAt: base.Add(8 * time.Minute)},
		{ID: 3, StartAt: base.Add(20 * time.Minute), EndAt: base.Add(30 * time.Minute)},
		{ID: 4, StartAt: base.Add(22 * time.Minute), EndAt: base.Add(28 * time.Minute)},
	}
	groups := SweepCapacity(windows, 1, 0)
	sets := map[string]bool{}
	for _, group := range groups {
		key := ""
		for _, w := range group {
			key += string(rune('0' + w.ID))
		}
		sets[key] = true
	}
	if len(sets) != 2 || !sets["12"] || !sets["34"] {
		t.Fatalf("expected two independent groups {1,2} and {3,4}, got %v", sets)
	}
}

func TestNoSlewDuplicateForRawCapacity002(t *testing.T) {
	base := time.Date(2026, 8, 23, 10, 0, 0, 0, time.UTC)
	station := model.GroundStation{ID: 1, StationCode: "GS1", AntennaCount: 1, SupportedBandsJSON: `["S"]`, SlewBufferSec: 120, StationStatus: "active"}
	asset := model.SatelliteAsset{ID: 1, SatelliteCode: "SAT1", SupportedBandsJSON: `["S"]`, MinimumContactSec: 60, AssetStatus: "active"}
	windows := []model.ContactWindow{
		{ID: 1, StationID: 1, SatelliteID: 1, StartAt: base, EndAt: base.Add(10 * time.Minute), Band: "S", Version: 1},
		{ID: 2, StationID: 1, SatelliteID: 1, StartAt: base.Add(2 * time.Minute), EndAt: base.Add(8 * time.Minute), Band: "S", Version: 1},
	}
	groups := Detect(DetectionContext{Windows: windows, Stations: map[uint]model.GroundStation{1: station}, Satellites: map[uint]model.SatelliteAsset{1: asset}})
	slew := 0
	for _, group := range groups {
		if group.ConflictType == constants.ConflictTypeSlewBuffer {
			slew++
		}
	}
	if slew != 0 {
		t.Fatalf("expected no slew_buffer duplicate for raw capacity conflict, got %d", slew)
	}
}

func TestNoSubsetDuplicateGroups002(t *testing.T) {
	base := time.Date(2026, 8, 23, 10, 0, 0, 0, time.UTC)
	station := model.GroundStation{ID: 1, StationCode: "GS2", AntennaCount: 1, SupportedBandsJSON: `["S"]`, SlewBufferSec: 0, StationStatus: "active"}
	asset := model.SatelliteAsset{ID: 1, SatelliteCode: "SAT2", SupportedBandsJSON: `["S"]`, MinimumContactSec: 60, AssetStatus: "active"}
	windows := []model.ContactWindow{
		{ID: 1, StationID: 1, SatelliteID: 1, StartAt: base, EndAt: base.Add(20 * time.Minute), Band: "S", Version: 1},
		{ID: 2, StationID: 1, SatelliteID: 1, StartAt: base.Add(2 * time.Minute), EndAt: base.Add(12 * time.Minute), Band: "S", Version: 1},
		{ID: 3, StationID: 1, SatelliteID: 1, StartAt: base.Add(4 * time.Minute), EndAt: base.Add(8 * time.Minute), Band: "S", Version: 1},
	}
	groups := Detect(DetectionContext{Windows: windows, Stations: map[uint]model.GroundStation{1: station}, Satellites: map[uint]model.SatelliteAsset{1: asset}})
	capacity := 0
	var ids map[uint]bool
	for _, group := range groups {
		if group.ConflictType == constants.ConflictTypeStationCapacity {
			capacity++
			ids = map[uint]bool{}
			for _, w := range group.Windows {
				ids[w.ID] = true
			}
		}
	}
	if capacity != 1 {
		t.Fatalf("expected exactly one station_capacity group (superset only), got %d", capacity)
	}
	if len(ids) != 3 || !ids[1] || !ids[2] || !ids[3] {
		t.Fatalf("expected the superset group {1,2,3}, got %v", ids)
	}
}

func TestBandAndDurationConflictsBothReported002(t *testing.T) {
	// covered at service level in record002_service_regression_test.go
}
