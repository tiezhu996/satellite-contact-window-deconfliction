package handler

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"satellite-contact-window-deconfliction/backend/internal/config"
	"satellite-contact-window-deconfliction/backend/internal/model"
	"satellite-contact-window-deconfliction/backend/internal/repository"
	"satellite-contact-window-deconfliction/backend/internal/service"
)

func TestDetectInvalidRangePropagatesP504(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db, err := gorm.Open(sqlite.Open("file:record005-handler?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.ContactWindow{}, &model.GroundStation{}, &model.SatelliteAsset{}, &model.ConflictResolution{}, &model.AuditEvent{}); err != nil {
		t.Fatal(err)
	}
	conflictService := service.NewConflictResolutionService(repository.NewConflictResolutionRepository(db), repository.NewContactWindowRepository(db), repository.NewGroundStationRepository(db), repository.NewSatelliteAssetRepository(db), service.NewAuditService(repository.NewSystemRepository(db)), config.Weights{PriorityLoss: 4, MovementDistance: .02, ContactDuration: .003, ResourceMargin: 2})
	h := NewConflictResolutionHandler(conflictService)
	r := gin.New()
	r.POST("/api/v1/conflicts/detect", h.Detect)
	body := `{"from":"not-a-time","to":"2026-09-02T00:00:00Z"}`
	req := httptest.NewRequest("POST", "/api/v1/conflicts/detect", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid detection range, got %d", rr.Code)
	}
}
