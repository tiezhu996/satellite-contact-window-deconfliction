package handler

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"satellite-contact-window-deconfliction/backend/internal/middleware"
	"satellite-contact-window-deconfliction/backend/internal/model"
	"satellite-contact-window-deconfliction/backend/internal/repository"
	"satellite-contact-window-deconfliction/backend/internal/service"
)

func TestRateLimitConcurrentSafeP701(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(middleware.RateLimit(100000))
	router.GET("/ping", func(context *gin.Context) { context.Status(http.StatusOK) })
	start := make(chan struct{})
	var wg sync.WaitGroup
	for i := 0; i < 16; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			for j := 0; j < 120; j++ {
				req := httptest.NewRequest(http.MethodGet, "/ping", nil)
				rr := httptest.NewRecorder()
				router.ServeHTTP(rr, req)
				if rr.Code != http.StatusOK {
					t.Errorf("unexpected status %d", rr.Code)
				}
			}
		}()
	}
	close(start)
	wg.Wait()
}

func TestRequestIDUniqueConcurrentP702(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(middleware.RequestID())
	router.GET("/ping", func(context *gin.Context) { context.Status(http.StatusOK) })
	const workers = 16
	const perWorker = 40
	ids := make(chan string, workers*perWorker)
	start := make(chan struct{})
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			for j := 0; j < perWorker; j++ {
				req := httptest.NewRequest(http.MethodGet, "/ping", nil)
				rr := httptest.NewRecorder()
				router.ServeHTTP(rr, req)
				ids <- rr.Header().Get("X-Request-ID")
			}
		}()
	}
	close(start)
	wg.Wait()
	close(ids)
	seen := map[string]bool{}
	for id := range ids {
		if id == "" {
			t.Fatal("empty request id")
		}
		if seen[id] {
			t.Fatalf("duplicate request id %q", id)
		}
		seen[id] = true
	}
}

func TestAuditRequestIDPerRequestP703(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db, err := gorm.Open(sqlite.Open("file:record007-audit?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.GroundStation{}, &model.SatelliteAsset{}, &model.ContactWindow{}, &model.AuditEvent{}); err != nil {
		t.Fatal(err)
	}
	station := model.GroundStation{StationCode: "GS-007", Name: "T", AntennaCount: 1, SupportedBandsJSON: `["S"]`, SlewBufferSec: 30, StationStatus: "active", Version: 1}
	asset := model.SatelliteAsset{SatelliteCode: "SAT-007", Name: "T", OrbitClass: "LEO", SupportedBandsJSON: `["S"]`, PriorityWeight: 5, MinimumContactSec: 60, AssetStatus: "active", Version: 1}
	if err := db.Create(&station).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&asset).Error; err != nil {
		t.Fatal(err)
	}
	windowService := service.NewContactWindowService(repository.NewContactWindowRepository(db), repository.NewGroundStationRepository(db), repository.NewSatelliteAssetRepository(db), service.NewAuditService(repository.NewSystemRepository(db)))
	h := NewContactWindowHandler(windowService)
	router := gin.New()
	router.Use(middleware.RequestID())
	router.POST("/api/v1/windows", h.Create)
	for i := 0; i < 4; i++ {
		rid := fmt.Sprintf("req-%d", i)
		body := fmt.Sprintf(`{"station_id":%d,"satellite_id":%d,"start_at":"2026-09-01T01:0%d:00Z","end_at":"2026-09-01T01:1%d:00Z","band":"S","elevation_peak_deg":40,"priority":5,"source_version":"v1.0"}`, station.ID, asset.ID, i, i)
		req := httptest.NewRequest("POST", "/api/v1/windows", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Request-ID", rid)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)
		if rr.Code != http.StatusCreated {
			t.Fatalf("create window %d failed: %d %s", i, rr.Code, rr.Body.String())
		}
	}
	events := []model.AuditEvent{}
	if err := db.Order("id ASC").Find(&events).Error; err != nil {
		t.Fatal(err)
	}
	got := []string{}
	for _, event := range events {
		if event.Action == "window.created" {
			got = append(got, event.RequestID)
		}
	}
	want := []string{"req-0", "req-1", "req-2", "req-3"}
	if len(got) != len(want) {
		t.Fatalf("expected %d audit events, got %d: %v", len(want), len(got), got)
	}
	for idx := range want {
		if got[idx] != want[idx] {
			t.Fatalf("audit request_id sequence wrong: got %v, want %v", got, want)
		}
	}
}

func TestRequestIDHelperNoSharedRaceP704(t *testing.T) {
	start := make(chan struct{})
	var wg sync.WaitGroup
	for i := 0; i < 16; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			ctx := &gin.Context{}
			for j := 0; j < 300; j++ {
				_ = RequestID(ctx)
			}
		}()
	}
	close(start)
	wg.Wait()
}
