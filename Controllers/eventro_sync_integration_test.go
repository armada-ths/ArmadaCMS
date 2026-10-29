package controllers

import (
	"ArmadaCMS/main/auth"
	"ArmadaCMS/main/db"
	"ArmadaCMS/main/models"
	"ArmadaCMS/main/utils"
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func TestEventroReviewDatabaseFlow(t *testing.T) {
	dsn := os.Getenv("EVENTRO_SYNC_TEST_DSN")
	if dsn == "" {
		t.Skip("set EVENTRO_SYNC_TEST_DSN to a disposable local eventro_sync_test database")
	}
	u, err := url.Parse(dsn)
	if err != nil || u.Path != "/eventro_sync_test" || (u.Hostname() != "localhost" && u.Hostname() != "127.0.0.1" && u.Hostname() != "host.docker.internal") {
		t.Fatal("requires disposable local eventro_sync_test database")
	}
	database, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, _ := database.DB()
	t.Cleanup(func() { _ = sqlDB.Close() })
	if err := database.AutoMigrate(&models.User{}, &models.Role{}, &models.Profile{}, &models.Event{}, &models.Exhibitor{}, &models.FairDateConfig{}, &models.RecruitmentPeriod{}, &models.RecruitmentRole{}, &models.AuditLog{}, &models.EventroSyncState{}); err != nil {
		t.Fatal(err)
	}
	tx := database.Begin()
	if tx.Error != nil {
		t.Fatal(tx.Error)
	}
	t.Cleanup(func() { _ = tx.Rollback().Error })
	database = tx
	if err := database.Exec("TRUNCATE eventro_sync_states, profiles, events, exhibitors, fair_date_configs, recruitment_periods, recruitment_roles, audit_logs RESTART IDENTITY CASCADE").Error; err != nil {
		t.Fatal(err)
	}
	previous := db.DB
	db.DB = database
	t.Cleanup(func() { db.DB = previous })
	t.Setenv("jwtsecret_laganda", "local-eventro-test-secret")
	r := httptest.NewRequest(http.MethodGet, "/eventromembers", nil)
	key := "test-person"
	for _, name := range []string{"Original", "Eventro correction"} {
		profile := models.Profile{EventroKey: &key, Name: name}
		if err := database.Transaction(func(tx *gorm.DB) error {
			_, _, _, err := syncEventroRecordWithAudit(tx, r, "profiles", &profile, "eventro_key", key, nil, nil)
			return err
		}); err != nil {
			t.Fatal(err)
		}
	}
	var current models.Profile
	if err := database.Where("eventro_key = ?", key).First(&current).Error; err != nil {
		t.Fatal(err)
	}
	if current.Name != "Eventro correction" {
		t.Fatal("source update not persisted")
	}
	if err := database.Model(&current).Update("name", "CMS correction").Error; err != nil {
		t.Fatal(err)
	}
	incoming := models.Profile{EventroKey: &key, Name: "New Eventro value"}
	if err := database.Transaction(func(tx *gorm.DB) error {
		_, _, _, err := syncEventroRecordWithAudit(tx, r, "profiles", &incoming, "eventro_key", key, nil, nil)
		return err
	}); err != nil {
		t.Fatal(err)
	}
	if err := database.First(&current, current.ID).Error; err != nil {
		t.Fatal(err)
	}
	if current.Name != "CMS correction" {
		t.Fatal("CMS correction overwritten")
	}
	state, err := loadEventroSyncState(database, "profiles", current.ID)
	if err != nil || !state.Conflicts["name"] {
		t.Fatalf("conflict not persisted: %v %#v", err, state)
	}
	decision := EventroReviewDecision{Resource: "profiles", RecordID: current.ID, Field: "name", Choice: "eventro", CMS: json.RawMessage(`"CMS correction"`), Eventro: json.RawMessage(`"New Eventro value"`)}
	request := func(perms []string, d EventroReviewDecision) *httptest.ResponseRecorder {
		body, _ := json.Marshal(d)
		req := httptest.NewRequest(http.MethodPost, "/eventrosyncreview", bytes.NewReader(body))
		token, err := utils.GenerateAccessToken(0, nil, perms)
		if err != nil {
			t.Fatal(err)
		}
		req.Header.Set("Authorization", "Bearer "+token)
		response := httptest.NewRecorder()
		auth.Middleware(auth.RequirePermission("eventrosync.access", ResolveEventroSyncReview)).ServeHTTP(response, req)
		return response
	}
	if response := request([]string{"eventrosync.access"}, decision); response.Code != 403 {
		t.Fatalf("missing edit permission: %d %s", response.Code, response.Body.String())
	}
	stale := decision
	stale.CMS = json.RawMessage(`"Old CMS value"`)
	if response := request([]string{"eventrosync.access", "profiles.edit"}, stale); response.Code != 409 {
		t.Fatalf("stale review: %d %s", response.Code, response.Body.String())
	}
	if response := request([]string{"eventrosync.access", "profiles.edit"}, decision); response.Code != 200 {
		t.Fatalf("resolve: %d %s", response.Code, response.Body.String())
	}
	if err := database.First(&current, current.ID).Error; err != nil {
		t.Fatal(err)
	}
	if current.Name != "New Eventro value" {
		t.Fatal("review decision not persisted")
	}
	state, err = loadEventroSyncState(database, "profiles", current.ID)
	if err != nil || state.Overrides["name"] || state.Conflicts["name"] {
		t.Fatal("ownership not reset")
	}
	other := models.FairDateConfig{Description: "Preserved CMS fair"}
	if err := database.Create(&other).Error; err != nil {
		t.Fatal(err)
	}
	fairID := "fair-test"
	if _, err := syncFairDateConfigWithAudit(r, models.FairDateConfig{EventroID: &fairID, Description: "Imported fair"}); err != nil {
		t.Fatal(err)
	}
	var count int64
	database.Model(&models.FairDateConfig{}).Count(&count)
	if count != 2 {
		t.Fatal("fair sync deleted another configuration")
	}
}
