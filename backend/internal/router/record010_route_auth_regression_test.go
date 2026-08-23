package router

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"satellite-contact-window-deconfliction/backend/internal/config"
)

func setupRecord010(t *testing.T, name string) *gin.Engine {
	t.Helper()
	cfg := config.Config{
		Port: "0", DBDriver: "sqlite", DBDSN: "file:record010-" + name + "?mode=memory&cache=shared", DBAutoMigrate: true,
		JWTSecret: "record010-secret-at-least-32-bytes", JWTTTL: time.Hour, CORSOrigin: "http://localhost:18527",
		RateLimitPerMinute: 5000,
		Weights:            config.Weights{PriorityLoss: 4, MovementDistance: .02, ContactDuration: .003, ResourceMargin: 2},
	}
	db, err := config.OpenDatabase(cfg)
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	return New(db, cfg)
}

func loginRecord010(t *testing.T, router *gin.Engine, username, password string) string {
	t.Helper()
	body := fmt.Sprintf(`{"username":%q,"password":%q}`, username, password)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("login %s failed: %d", username, rr.Code)
	}
	var resp struct {
		Data struct {
			Token string `json:"token"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode login: %v", err)
	}
	return resp.Data.Token
}

func authedRequest(method, target, token string, body string) *http.Request {
	req := httptest.NewRequest(method, target, strings.NewReader(body))
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	req.Header.Set("Content-Type", "application/json")
	return req
}

func TestUnauthenticatedStationsBlockedP1001(t *testing.T) {
	router := setupRecord010(t, "p1001")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/api/v1/stations", nil))
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for unauthenticated stations list, got %d", rr.Code)
	}
}

func TestExportRequiresReviewerP1002(t *testing.T) {
	router := setupRecord010(t, "p1002")
	token := loginRecord010(t, router, "scheduler", "Scheduler#527")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, authedRequest(http.MethodGet, "/api/v1/conflicts/1/export", token, ""))
	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for scheduler export, got %d", rr.Code)
	}
}

func TestLockRequiresSchedulerP1003(t *testing.T) {
	router := setupRecord010(t, "p1003")
	token := loginRecord010(t, router, "reviewer", "Reviewer#527")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, authedRequest(http.MethodPost, "/api/v1/windows/1/lock", token, `{"expected_version":1}`))
	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for reviewer lock, got %d", rr.Code)
	}
}

func TestAdminCanCreateSatelliteP1004(t *testing.T) {
	router := setupRecord010(t, "p1004")
	token := loginRecord010(t, router, "admin", "Admin#527")
	body := `{"satellite_code":"NOVA-01","name":"Nova","orbit_class":"LEO","supported_bands":["S","X"],"priority_weight":5,"minimum_contact_sec":300,"asset_status":"active"}`
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, authedRequest(http.MethodPost, "/api/v1/satellites", token, body))
	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201 for admin satellite create, got %d", rr.Code)
	}
}

func TestStationCreateRequiresSchedulerP1005(t *testing.T) {
	router := setupRecord010(t, "p1005")
	token := loginRecord010(t, router, "reviewer", "Reviewer#527")
	body := `{"station_code":"RVW-01","name":"Reviewer Station","latitude":1,"longitude":2,"antenna_count":1,"supported_bands":["S"],"slew_buffer_sec":30,"station_status":"active"}`
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, authedRequest(http.MethodPost, "/api/v1/stations", token, body))
	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for reviewer station create, got %d", rr.Code)
	}
}

func TestStationUpdateRequiresSchedulerP1006(t *testing.T) {
	router := setupRecord010(t, "p1006")
	token := loginRecord010(t, router, "reviewer", "Reviewer#527")
	body := `{"name":"Renamed","latitude":1,"longitude":2,"antenna_count":1,"supported_bands":["S"],"slew_buffer_sec":30,"station_status":"active","expected_version":1}`
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, authedRequest(http.MethodPut, "/api/v1/stations/1", token, body))
	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for reviewer station update, got %d", rr.Code)
	}
}

func TestSatelliteUpdateRequiresSchedulerP1007(t *testing.T) {
	router := setupRecord010(t, "p1007")
	token := loginRecord010(t, router, "reviewer", "Reviewer#527")
	body := `{"name":"Renamed","orbit_class":"LEO","supported_bands":["S"],"priority_weight":5,"minimum_contact_sec":300,"asset_status":"active","expected_version":1}`
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, authedRequest(http.MethodPut, "/api/v1/satellites/1", token, body))
	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for reviewer satellite update, got %d", rr.Code)
	}
}
