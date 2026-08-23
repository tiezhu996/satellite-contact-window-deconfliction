package service

import (
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"satellite-contact-window-deconfliction/backend/internal/dto"
	"satellite-contact-window-deconfliction/backend/internal/model"
	"satellite-contact-window-deconfliction/backend/internal/repository"
)

func openRecord001DB(t *testing.T, name string, models ...any) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:record001-"+name+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	if err := db.AutoMigrate(models...); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}

func stationReq001(code string) dto.CreateGroundStationRequest {
	return dto.CreateGroundStationRequest{StationCode: code, Name: "Original", Latitude: 30.0, Longitude: 100.0, AntennaCount: 1, SupportedBands: []string{"S"}, SlewBufferSec: 60, StationStatus: "active"}
}

func satelliteReq001(code string) dto.CreateSatelliteAssetRequest {
	return dto.CreateSatelliteAssetRequest{SatelliteCode: code, Name: "Original", OrbitClass: "LEO", SupportedBands: []string{"S"}, PriorityWeight: 5, MinimumContactSec: 300, AssetStatus: "active"}
}

func TestStationUpdateReturnsFreshDataP101(t *testing.T) {
	db := openRecord001DB(t, "station", &model.GroundStation{}, &model.AuditEvent{})
	svc := NewGroundStationService(repository.NewGroundStationRepository(db), NewAuditService(repository.NewSystemRepository(db)))
	actor := dto.Actor{Username: "scheduler", Role: "scheduler"}
	created, err := svc.Create(stationReq001("GS-NEW01"), actor, "req-101")
	if err != nil {
		t.Fatal(err)
	}
	updated, err := svc.Update(created.ID, dto.UpdateGroundStationRequest{Name: "Renamed", Latitude: 31, Longitude: 101, AntennaCount: 2, SupportedBands: []string{"S", "X"}, SlewBufferSec: 90, StationStatus: "active", ExpectedVersion: created.Version}, actor, "req-101")
	if err != nil {
		t.Fatal(err)
	}
	if updated.Name != "Renamed" || updated.Version != created.Version+1 {
		t.Fatalf("update response stale: got %+v", updated)
	}
}

func TestSatelliteUpdateReturnsFreshDataP102(t *testing.T) {
	db := openRecord001DB(t, "satellite", &model.SatelliteAsset{}, &model.AuditEvent{})
	svc := NewSatelliteAssetService(repository.NewSatelliteAssetRepository(db), NewAuditService(repository.NewSystemRepository(db)))
	actor := dto.Actor{Username: "scheduler", Role: "scheduler"}
	created, err := svc.Create(satelliteReq001("SAT-NEW01"), actor, "req-102")
	if err != nil {
		t.Fatal(err)
	}
	updated, err := svc.Update(created.ID, dto.UpdateSatelliteAssetRequest{Name: "Renamed", OrbitClass: "MEO", SupportedBands: []string{"S"}, PriorityWeight: 7, MinimumContactSec: 360, AssetStatus: "active", ExpectedVersion: created.Version}, actor, "req-102")
	if err != nil {
		t.Fatal(err)
	}
	if updated.Name != "Renamed" || updated.Version != created.Version+1 {
		t.Fatalf("update response stale: got %+v", updated)
	}
}

func TestStationUpdateAuditBeforeAfterP103(t *testing.T) {
	db := openRecord001DB(t, "station-audit", &model.GroundStation{}, &model.AuditEvent{})
	sysRepo := repository.NewSystemRepository(db)
	svc := NewGroundStationService(repository.NewGroundStationRepository(db), NewAuditService(sysRepo))
	actor := dto.Actor{Username: "scheduler", Role: "scheduler"}
	created, err := svc.Create(stationReq001("GS-AUD01"), actor, "req-103")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Update(created.ID, dto.UpdateGroundStationRequest{Name: "Renamed", Latitude: 31, Longitude: 101, AntennaCount: 2, SupportedBands: []string{"S", "X"}, SlewBufferSec: 90, StationStatus: "active", ExpectedVersion: created.Version}, actor, "req-103"); err != nil {
		t.Fatal(err)
	}
	events, _, err := sysRepo.ListAudit(1, 20, "ground_station", "station.updated")
	if err != nil || len(events) != 1 {
		t.Fatalf("expected one station.updated audit event, got %d (%v)", len(events), err)
	}
	after := decodeObject(events[0].AfterSummary)
	if after["antenna_count"] != float64(2) {
		t.Fatalf("audit after_summary wrong: %v", after)
	}
	before := decodeObject(events[0].BeforeSummary)
	if before["antenna_count"] != float64(1) {
		t.Fatalf("audit before_summary wrong: %v", before)
	}
}

func TestSatelliteUpdateAuditBeforeAfterP104(t *testing.T) {
	db := openRecord001DB(t, "sat-audit", &model.SatelliteAsset{}, &model.AuditEvent{})
	sysRepo := repository.NewSystemRepository(db)
	svc := NewSatelliteAssetService(repository.NewSatelliteAssetRepository(db), NewAuditService(sysRepo))
	actor := dto.Actor{Username: "scheduler", Role: "scheduler"}
	created, err := svc.Create(satelliteReq001("SAT-AUD01"), actor, "req-104")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Update(created.ID, dto.UpdateSatelliteAssetRequest{Name: "Renamed", OrbitClass: "MEO", SupportedBands: []string{"S"}, PriorityWeight: 7, MinimumContactSec: 360, AssetStatus: "active", ExpectedVersion: created.Version}, actor, "req-104"); err != nil {
		t.Fatal(err)
	}
	events, _, err := sysRepo.ListAudit(1, 20, "satellite_asset", "satellite.updated")
	if err != nil || len(events) != 1 {
		t.Fatalf("expected one satellite.updated audit event, got %d (%v)", len(events), err)
	}
	after := decodeObject(events[0].AfterSummary)
	if after["priority_weight"] != float64(7) {
		t.Fatalf("audit after_summary wrong: %v", after)
	}
	before := decodeObject(events[0].BeforeSummary)
	if before["priority_weight"] != float64(5) {
		t.Fatalf("audit before_summary wrong: %v", before)
	}
}

func TestStationCreateAuditBeforeEmptyP105(t *testing.T) {
	db := openRecord001DB(t, "station-create", &model.GroundStation{}, &model.AuditEvent{})
	sysRepo := repository.NewSystemRepository(db)
	svc := NewGroundStationService(repository.NewGroundStationRepository(db), NewAuditService(sysRepo))
	actor := dto.Actor{Username: "scheduler", Role: "scheduler"}
	if _, err := svc.Create(stationReq001("GS-CRT01"), actor, "req-105"); err != nil {
		t.Fatal(err)
	}
	events, _, err := sysRepo.ListAudit(1, 20, "ground_station", "station.created")
	if err != nil || len(events) != 1 {
		t.Fatalf("expected one station.created audit event, got %d (%v)", len(events), err)
	}
	if events[0].BeforeSummary != "{}" {
		t.Fatalf("create audit before_summary should be empty, got %s", events[0].BeforeSummary)
	}
}

func TestSatelliteCreateAuditBeforeEmptyP106(t *testing.T) {
	db := openRecord001DB(t, "sat-create", &model.SatelliteAsset{}, &model.AuditEvent{})
	sysRepo := repository.NewSystemRepository(db)
	svc := NewSatelliteAssetService(repository.NewSatelliteAssetRepository(db), NewAuditService(sysRepo))
	actor := dto.Actor{Username: "scheduler", Role: "scheduler"}
	if _, err := svc.Create(satelliteReq001("SAT-CRT01"), actor, "req-106"); err != nil {
		t.Fatal(err)
	}
	events, _, err := sysRepo.ListAudit(1, 20, "satellite_asset", "satellite.created")
	if err != nil || len(events) != 1 {
		t.Fatalf("expected one satellite.created audit event, got %d (%v)", len(events), err)
	}
	if events[0].BeforeSummary != "{}" {
		t.Fatalf("create audit before_summary should be empty, got %s", events[0].BeforeSummary)
	}
}
