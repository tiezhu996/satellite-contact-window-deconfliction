package service

import (
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"satellite-contact-window-deconfliction/backend/internal/config"
	"satellite-contact-window-deconfliction/backend/internal/constants"
	"satellite-contact-window-deconfliction/backend/internal/dto"
	"satellite-contact-window-deconfliction/backend/internal/model"
	"satellite-contact-window-deconfliction/backend/internal/repository"
)

func TestBandAndDurationConflictsBothReported002(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:record002-service?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.GroundStation{}, &model.SatelliteAsset{}, &model.ContactWindow{}, &model.ConflictResolution{}, &model.AuditEvent{}); err != nil {
		t.Fatal(err)
	}
	station := model.GroundStation{StationCode: "GS-SRV", Name: "T", AntennaCount: 1, SupportedBandsJSON: `["S"]`, SlewBufferSec: 0, StationStatus: "active", Version: 1}
	asset := model.SatelliteAsset{SatelliteCode: "SAT-SRV", Name: "T", OrbitClass: "LEO", SupportedBandsJSON: `["S"]`, PriorityWeight: 5, MinimumContactSec: 600, AssetStatus: "active", Version: 1}
	if err := db.Create(&station).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&asset).Error; err != nil {
		t.Fatal(err)
	}
	base := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	window := model.ContactWindow{StationID: station.ID, SatelliteID: asset.ID, StartAt: base, EndAt: base.Add(2 * time.Minute), Band: "Ku", WindowStatus: constants.WindowStatusCandidate, Priority: 5, SourceVersion: "v1.0", Version: 1}
	if err := db.Create(&window).Error; err != nil {
		t.Fatal(err)
	}
	svc := NewConflictResolutionService(repository.NewConflictResolutionRepository(db), repository.NewContactWindowRepository(db), repository.NewGroundStationRepository(db), repository.NewSatelliteAssetRepository(db), NewAuditService(repository.NewSystemRepository(db)), config.Weights{PriorityLoss: 4, MovementDistance: .02, ContactDuration: .003, ResourceMargin: 2})
	result, err := svc.Detect(dto.DetectConflictsRequest{From: base.Add(-time.Minute).Format(time.RFC3339), To: base.Add(time.Hour).Format(time.RFC3339)}, dto.Actor{Username: "scheduler", Role: constants.RoleScheduler}, "req-204")
	if err != nil {
		t.Fatal(err)
	}
	types := map[string]bool{}
	for _, resolution := range result.Resolutions {
		types[resolution.ConflictType] = true
	}
	if !types[constants.ConflictTypeBandMismatch] || !types[constants.ConflictTypeDurationShortfall] {
		t.Fatalf("expected both band_mismatch and duration_shortfall resolutions, got %v", types)
	}
}
