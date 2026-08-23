package service

import (
	"errors"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"satellite-contact-window-deconfliction/backend/internal/config"
	"satellite-contact-window-deconfliction/backend/internal/constants"
	"satellite-contact-window-deconfliction/backend/internal/dto"
	"satellite-contact-window-deconfliction/backend/internal/model"
	"satellite-contact-window-deconfliction/backend/internal/repository"
)

func openRecord005DB(t *testing.T, name string, models ...any) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:record005-"+name+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	if err := db.AutoMigrate(models...); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}

func newConflictService005(db *gorm.DB) *ConflictResolutionService {
	return NewConflictResolutionService(repository.NewConflictResolutionRepository(db), repository.NewContactWindowRepository(db), repository.NewGroundStationRepository(db), repository.NewSatelliteAssetRepository(db), NewAuditService(repository.NewSystemRepository(db)), config.Weights{PriorityLoss: 4, MovementDistance: .02, ContactDuration: .003, ResourceMargin: 2})
}

func TestSubmitVersionConflictNotSwallowedP501(t *testing.T) {
	db := openRecord005DB(t, "submit", &model.ConflictResolution{})
	svc := newConflictService005(db)
	resolution := model.ConflictResolution{ConflictKey: "k-501", WindowIDsJSON: "[]", WindowVersionsJSON: "{}", ConflictType: constants.ConflictTypeStationCapacity, EvidenceJSON: "{}", SuggestionsJSON: "[]", WeightsJSON: "{}", SelectedAction: "{}", ResolutionStatus: constants.ResolutionStatusProposed, Version: 2}
	if err := db.Create(&resolution).Error; err != nil {
		t.Fatal(err)
	}
	_, err := svc.Submit(resolution.ID, dto.ConflictActionRequest{ExpectedVersion: 9}, dto.Actor{Username: "scheduler", Role: constants.RoleScheduler}, "req-501")
	var appErr *AppError
	if !errors.As(err, &appErr) || appErr.Status != 409 || appErr.Code != "version_conflict" {
		t.Fatalf("expected 409 version_conflict, got %v", err)
	}
}

func TestExportRejectedBlockedP502(t *testing.T) {
	db := openRecord005DB(t, "export", &model.ConflictResolution{})
	svc := newConflictService005(db)
	resolution := model.ConflictResolution{ConflictKey: "k-502", WindowIDsJSON: "[]", WindowVersionsJSON: "{}", ConflictType: constants.ConflictTypeStationCapacity, EvidenceJSON: "{}", SuggestionsJSON: "[]", WeightsJSON: "{}", SelectedAction: "{}", ResolutionStatus: constants.ResolutionStatusRejected, Version: 3}
	if err := db.Create(&resolution).Error; err != nil {
		t.Fatal(err)
	}
	_, err := svc.Export(resolution.ID)
	var appErr *AppError
	if !errors.As(err, &appErr) || appErr.Status != 409 {
		t.Fatalf("expected export of rejected resolution to be blocked, got %v", err)
	}
}

func TestGetMissingConflictReturns404P503(t *testing.T) {
	db := openRecord005DB(t, "get", &model.ConflictResolution{})
	svc := newConflictService005(db)
	_, err := svc.Get(99999)
	var appErr *AppError
	if !errors.As(err, &appErr) || appErr.Status != 404 {
		t.Fatalf("expected 404 for missing conflict, got %v", err)
	}
}
