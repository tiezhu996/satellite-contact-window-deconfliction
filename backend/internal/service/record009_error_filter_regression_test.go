package service

import (
	"errors"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"satellite-contact-window-deconfliction/backend/internal/model"
	"satellite-contact-window-deconfliction/backend/internal/repository"
)

func openRecord009DB(t *testing.T, name string, models ...any) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:record009-"+name+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	if err := db.AutoMigrate(models...); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}

func TestStationMissingReturns404P901(t *testing.T) {
	db := openRecord009DB(t, "station", &model.GroundStation{}, &model.AuditEvent{})
	svc := NewGroundStationService(repository.NewGroundStationRepository(db), NewAuditService(repository.NewSystemRepository(db)))
	_, err := svc.Get(99999)
	var appErr *AppError
	if !errors.As(err, &appErr) || appErr.Status != 404 {
		t.Fatalf("expected 404 for missing station, got %v", err)
	}
}

func TestSatelliteMissingReturns404P902(t *testing.T) {
	db := openRecord009DB(t, "satellite", &model.SatelliteAsset{}, &model.AuditEvent{})
	svc := NewSatelliteAssetService(repository.NewSatelliteAssetRepository(db), NewAuditService(repository.NewSystemRepository(db)))
	_, err := svc.Get(99999)
	var appErr *AppError
	if !errors.As(err, &appErr) || appErr.Status != 404 {
		t.Fatalf("expected 404 for missing satellite, got %v", err)
	}
}
