package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"satellite-contact-window-deconfliction/backend/internal/constants"
	"satellite-contact-window-deconfliction/backend/internal/model"
	"satellite-contact-window-deconfliction/backend/internal/repository"
	"satellite-contact-window-deconfliction/backend/internal/service"
)

func setupRecord004(t *testing.T, name string) (*gin.Engine, *gorm.DB) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	db, err := gorm.Open(sqlite.Open("file:record004-"+name+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	if err := db.AutoMigrate(&model.GroundStation{}, &model.SatelliteAsset{}, &model.ContactWindow{}, &model.AuditEvent{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	station := model.GroundStation{StationCode: "GS-004", Name: "Test", AntennaCount: 2, SupportedBandsJSON: `["S","X"]`, SlewBufferSec: 60, StationStatus: "active", Version: 1}
	asset := model.SatelliteAsset{SatelliteCode: "SAT-004", Name: "Test", OrbitClass: "LEO", SupportedBandsJSON: `["S","X"]`, PriorityWeight: 5, MinimumContactSec: 60, AssetStatus: "active", Version: 1}
	if err := db.Create(&station).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&asset).Error; err != nil {
		t.Fatal(err)
	}
	windowService := service.NewContactWindowService(repository.NewContactWindowRepository(db), repository.NewGroundStationRepository(db), repository.NewSatelliteAssetRepository(db), service.NewAuditService(repository.NewSystemRepository(db)))
	h := NewContactWindowHandler(windowService)
	r := gin.New()
	r.POST("/api/v1/windows", h.Create)
	r.GET("/api/v1/windows", h.List)
	r.POST("/api/v1/windows/:id/submit", h.Submit)
	r.PUT("/api/v1/windows/:id", h.Update)
	return r, db
}

func canceledRequest(method, target, body string) *http.Request {
	req := httptest.NewRequest(method, target, strings.NewReader(body))
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	return req.WithContext(ctx)
}

func windowCount004(db *gorm.DB) int64 {
	var count int64
	db.Model(&model.ContactWindow{}).Count(&count)
	return count
}

func TestCreateHonorsCancelledContextP401(t *testing.T) {
	router, db := setupRecord004(t, "TestCreateHonorsCancelledContextP401")
	body := `{"station_id":1,"satellite_id":1,"start_at":"2026-09-01T00:00:00Z","end_at":"2026-09-01T00:10:00Z","band":"S","elevation_peak_deg":45,"priority":5,"source_version":"v1.0"}`
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, canceledRequest("POST", "/api/v1/windows", body))
	if rr.Code == http.StatusCreated {
		t.Fatalf("create succeeded on cancelled request: %d", rr.Code)
	}
	if windowCount004(db) != 0 {
		t.Fatalf("window was persisted despite cancelled request")
	}
}

func TestListHonorsCancelledContextP402(t *testing.T) {
	router, _ := setupRecord004(t, "TestListHonorsCancelledContextP402")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, canceledRequest("GET", "/api/v1/windows", ""))
	if rr.Code == http.StatusOK {
		t.Fatalf("list returned 200 on cancelled request")
	}
}

func TestSubmitHonorsCancelledContextP403(t *testing.T) {
	router, db := setupRecord004(t, "TestSubmitHonorsCancelledContextP403")
	base := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	window := model.ContactWindow{StationID: 1, SatelliteID: 1, StartAt: base, EndAt: base.Add(10 * time.Minute), Band: "S", WindowStatus: constants.WindowStatusCandidate, Priority: 5, SourceVersion: "v1", Version: 1}
	if err := db.Create(&window).Error; err != nil {
		t.Fatal(err)
	}
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, canceledRequest("POST", "/api/v1/windows/1/submit", `{"expected_version":1}`))
	if rr.Code == http.StatusOK {
		t.Fatalf("submit succeeded on cancelled request")
	}
	var after model.ContactWindow
	if err := db.First(&after, window.ID).Error; err != nil {
		t.Fatal(err)
	}
	if after.WindowStatus != constants.WindowStatusCandidate {
		t.Fatalf("window status changed to %s despite cancelled request", after.WindowStatus)
	}
}

func TestUpdateHonorsCancelledContextP404(t *testing.T) {
	router, db := setupRecord004(t, "TestUpdateHonorsCancelledContextP404")
	base := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	window := model.ContactWindow{StationID: 1, SatelliteID: 1, StartAt: base, EndAt: base.Add(10 * time.Minute), Band: "S", WindowStatus: constants.WindowStatusCandidate, Priority: 5, SourceVersion: "v1", Version: 1}
	if err := db.Create(&window).Error; err != nil {
		t.Fatal(err)
	}
	body := `{"station_id":1,"satellite_id":1,"start_at":"2026-09-01T00:00:00Z","end_at":"2026-09-01T00:20:00Z","band":"S","elevation_peak_deg":50,"priority":6,"window_status":"candidate","source_version":"v2.0","expected_version":1}`
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, canceledRequest("PUT", "/api/v1/windows/1", body))
	if rr.Code == http.StatusOK {
		t.Fatalf("update succeeded on cancelled request")
	}
	var after model.ContactWindow
	if err := db.First(&after, window.ID).Error; err != nil {
		t.Fatal(err)
	}
	if after.EndAt != window.EndAt || after.Priority != 5 {
		t.Fatalf("window was modified despite cancelled request")
	}
	_ = json.Valid
}
