package service

import (
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"satellite-contact-window-deconfliction/backend/internal/constants"
	"satellite-contact-window-deconfliction/backend/internal/dto"
	"satellite-contact-window-deconfliction/backend/internal/model"
	"satellite-contact-window-deconfliction/backend/internal/repository"
)

func TestCandidateSubmitSucceedsViaServiceP805(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:record008-window?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.GroundStation{}, &model.SatelliteAsset{}, &model.ContactWindow{}, &model.AuditEvent{}); err != nil {
		t.Fatal(err)
	}
	station := model.GroundStation{StationCode: "GS-008", Name: "T", AntennaCount: 1, SupportedBandsJSON: `["S"]`, SlewBufferSec: 30, StationStatus: "active", Version: 1}
	asset := model.SatelliteAsset{SatelliteCode: "SAT-008", Name: "T", OrbitClass: "LEO", SupportedBandsJSON: `["S"]`, PriorityWeight: 5, MinimumContactSec: 60, AssetStatus: "active", Version: 1}
	if err := db.Create(&station).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&asset).Error; err != nil {
		t.Fatal(err)
	}
	svc := NewContactWindowService(repository.NewContactWindowRepository(db), repository.NewGroundStationRepository(db), repository.NewSatelliteAssetRepository(db), NewAuditService(repository.NewSystemRepository(db)))
	base := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	created, err := svc.Create(dto.CreateContactWindowRequest{StationID: station.ID, SatelliteID: asset.ID, StartAt: base.Format(time.RFC3339), EndAt: base.Add(10 * time.Minute).Format(time.RFC3339), Band: "S", ElevationPeakDeg: 40, Priority: 5, SourceVersion: "v1.0"}, dto.Actor{Username: "scheduler", Role: constants.RoleScheduler}, "req-805")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Submit(created.ID, dto.WindowActionRequest{ExpectedVersion: created.Version}, dto.Actor{Username: "scheduler", Role: constants.RoleScheduler}, "req-805"); err != nil {
		t.Fatalf("candidate window submit failed: %v", err)
	}
}

func TestSubmittedLockSucceedsViaServiceP806(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:record008-lock?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.GroundStation{}, &model.SatelliteAsset{}, &model.ContactWindow{}, &model.AuditEvent{}); err != nil {
		t.Fatal(err)
	}
	station := model.GroundStation{StationCode: "GS-008B", Name: "T", AntennaCount: 1, SupportedBandsJSON: `["S"]`, SlewBufferSec: 30, StationStatus: "active", Version: 1}
	asset := model.SatelliteAsset{SatelliteCode: "SAT-008B", Name: "T", OrbitClass: "LEO", SupportedBandsJSON: `["S"]`, PriorityWeight: 5, MinimumContactSec: 60, AssetStatus: "active", Version: 1}
	if err := db.Create(&station).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&asset).Error; err != nil {
		t.Fatal(err)
	}
	base := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	window := model.ContactWindow{StationID: station.ID, SatelliteID: asset.ID, StartAt: base, EndAt: base.Add(10 * time.Minute), Band: "S", WindowStatus: constants.WindowStatusSubmitted, Priority: 5, SourceVersion: "v1.0", Version: 1}
	if err := db.Create(&window).Error; err != nil {
		t.Fatal(err)
	}
	svc := NewContactWindowService(repository.NewContactWindowRepository(db), repository.NewGroundStationRepository(db), repository.NewSatelliteAssetRepository(db), NewAuditService(repository.NewSystemRepository(db)))
	if _, err := svc.Lock(window.ID, dto.WindowActionRequest{ExpectedVersion: 1}, dto.Actor{Username: "scheduler", Role: constants.RoleScheduler}, "req-806"); err != nil {
		t.Fatalf("submitted window lock failed: %v", err)
	}
}
