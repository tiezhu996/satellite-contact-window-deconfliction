package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"satellite-contact-window-deconfliction/backend/internal/model"
	"satellite-contact-window-deconfliction/backend/internal/repository"
	"satellite-contact-window-deconfliction/backend/internal/service"
)

func TestStationSearchFilterP903(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db, err := gorm.Open(sqlite.Open("file:record009-search?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.GroundStation{}, &model.AuditEvent{}); err != nil {
		t.Fatal(err)
	}
	stations := []model.GroundStation{
		{StationCode: "ALPHA-01", Name: "Alpha Relay", AntennaCount: 1, SupportedBandsJSON: `["S"]`, StationStatus: "active", Version: 1},
		{StationCode: "BETA-02", Name: "Beta Relay", AntennaCount: 1, SupportedBandsJSON: `["S"]`, StationStatus: "active", Version: 1},
	}
	if err := db.Create(&stations).Error; err != nil {
		t.Fatal(err)
	}
	svc := service.NewGroundStationService(repository.NewGroundStationRepository(db), service.NewAuditService(repository.NewSystemRepository(db)))
	h := NewGroundStationHandler(svc)
	router := gin.New()
	router.GET("/api/v1/stations", h.List)
	req := httptest.NewRequest("GET", "/api/v1/stations?search=alpha", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	var body struct {
		Data []struct {
			StationCode string `json:"station_code"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(body.Data) != 1 || body.Data[0].StationCode != "ALPHA-01" {
		t.Fatalf("search filter ignored: got %+v", body.Data)
	}
}

func TestSatelliteStatusFilterP904(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db, err := gorm.Open(sqlite.Open("file:record009-status?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.SatelliteAsset{}, &model.AuditEvent{}); err != nil {
		t.Fatal(err)
	}
	assets := []model.SatelliteAsset{
		{SatelliteCode: "ONE-01", Name: "One", OrbitClass: "LEO", SupportedBandsJSON: `["S"]`, PriorityWeight: 5, MinimumContactSec: 60, AssetStatus: "active", Version: 1},
		{SatelliteCode: "TWO-02", Name: "Two", OrbitClass: "LEO", SupportedBandsJSON: `["S"]`, PriorityWeight: 5, MinimumContactSec: 60, AssetStatus: "standby", Version: 1},
	}
	if err := db.Create(&assets).Error; err != nil {
		t.Fatal(err)
	}
	svc := service.NewSatelliteAssetService(repository.NewSatelliteAssetRepository(db), service.NewAuditService(repository.NewSystemRepository(db)))
	h := NewSatelliteAssetHandler(svc)
	router := gin.New()
	router.GET("/api/v1/satellites", h.List)
	req := httptest.NewRequest("GET", "/api/v1/satellites?status=active", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	var body struct {
		Data []struct {
			SatelliteCode string `json:"satellite_code"`
			AssetStatus   string `json:"asset_status"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	for _, item := range body.Data {
		if item.AssetStatus != "active" {
			t.Fatalf("status filter ignored: got %+v", item)
		}
	}
	if len(body.Data) != 1 {
		t.Fatalf("status filter returned %d rows, want 1", len(body.Data))
	}
}

func TestStationGetMissingReturns404P905(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db, err := gorm.Open(sqlite.Open("file:record009-gets?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.GroundStation{}, &model.AuditEvent{}); err != nil {
		t.Fatal(err)
	}
	svc := service.NewGroundStationService(repository.NewGroundStationRepository(db), service.NewAuditService(repository.NewSystemRepository(db)))
	h := NewGroundStationHandler(svc)
	router := gin.New()
	router.GET("/api/v1/stations/:id", h.Get)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, httptest.NewRequest("GET", "/api/v1/stations/99999", nil))
	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for missing station get, got %d", rr.Code)
	}
}

func TestSatelliteGetMissingReturns404P906(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db, err := gorm.Open(sqlite.Open("file:record009-getsat?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.SatelliteAsset{}, &model.AuditEvent{}); err != nil {
		t.Fatal(err)
	}
	svc := service.NewSatelliteAssetService(repository.NewSatelliteAssetRepository(db), service.NewAuditService(repository.NewSystemRepository(db)))
	h := NewSatelliteAssetHandler(svc)
	router := gin.New()
	router.GET("/api/v1/satellites/:id", h.Get)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, httptest.NewRequest("GET", "/api/v1/satellites/99999", nil))
	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for missing satellite get, got %d", rr.Code)
	}
}
