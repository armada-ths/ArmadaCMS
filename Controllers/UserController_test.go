package controllers

import (
	"ArmadaCMS/main/db"
	"ArmadaCMS/main/models"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// Run against a disposable local database with USER_DELETE_TEST_DSN set.
func TestDeleteUserWithRefreshTokens(t *testing.T) {
	dsn := os.Getenv("USER_DELETE_TEST_DSN")
	if dsn == "" {
		t.Skip("set USER_DELETE_TEST_DSN to a disposable local user_delete_test database")
	}
	parsed, err := url.Parse(dsn)
	if err != nil || parsed.Path != "/user_delete_test" ||
		(parsed.Hostname() != "localhost" && parsed.Hostname() != "127.0.0.1" && parsed.Hostname() != "host.docker.internal") {
		t.Fatal("USER_DELETE_TEST_DSN must target a local user_delete_test database")
	}
	database, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := database.DB()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })

	for _, scenario := range []struct {
		name       string
		withTokens bool
		block      bool
	}{
		{name: "active disabled and expired tokens", withTokens: true},
		{name: "no tokens"},
		{name: "failed user deletion rolls back tokens and audit", withTokens: true, block: true},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			tx := database.Begin()
			if tx.Error != nil {
				t.Fatal(tx.Error)
			}
			previousDB := db.DB
			db.DB = tx
			t.Cleanup(func() {
				_ = tx.Rollback().Error
				db.DB = previousDB
			})
			if err := tx.AutoMigrate(&models.User{}, &models.Role{}, &models.RefreshToken{}, &models.AuditLog{}); err != nil {
				t.Fatal(err)
			}
			user := models.User{Username: "delete-test", Password: "unused"}
			other := models.User{Username: "keep-test", Password: "unused"}
			for _, u := range []*models.User{&user, &other} {
				if err := tx.Create(u).Error; err != nil {
					t.Fatal(err)
				}
			}
			now := time.Now().UTC()
			createToken := func(userID uint, secret string, enabled bool, validTo time.Time) {
				t.Helper()
				// A map preserves false despite the model's default:true tag.
				if err := tx.Model(&models.RefreshToken{}).Create(map[string]any{
					"user_id": userID, "refresh_token": secret, "enabled": enabled,
					"valid_from": now.Add(-time.Hour), "valid_to": validTo,
				}).Error; err != nil {
					t.Fatal(err)
				}
			}
			createToken(other.ID, "other-secret", true, now.Add(time.Hour))
			if scenario.withTokens {
				createToken(user.ID, "active-secret", true, now.Add(time.Hour))
				createToken(user.ID, "disabled-secret", false, now.Add(time.Hour))
				createToken(user.ID, "expired-secret", true, now.Add(-time.Minute))
			}
			if scenario.block {
				if err := tx.Exec("CREATE TEMP TABLE user_delete_blocker (user_id bigint REFERENCES users(id))").Error; err != nil {
					t.Fatal(err)
				}
				if err := tx.Exec("INSERT INTO user_delete_blocker (user_id) VALUES (?)", user.ID).Error; err != nil {
					t.Fatal(err)
				}
			}
			request := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/v1/customusers/%d", user.ID), nil)
			request = mux.SetURLVars(request, map[string]string{"id": fmt.Sprint(user.ID)})
			response := httptest.NewRecorder()
			DeleteUser(response, request)
			wantStatus := http.StatusNoContent
			if scenario.block {
				wantStatus = http.StatusInternalServerError
			}
			if response.Code != wantStatus {
				t.Fatalf("status = %d, want %d: %s", response.Code, wantStatus, response.Body.String())
			}
			assertCount := func(model any, condition string, id uint, want int64) {
				t.Helper()
				var count int64
				if err := tx.Model(model).Where(condition, id).Count(&count).Error; err != nil {
					t.Fatal(err)
				}
				if count != want {
					t.Fatalf("%T count = %d, want %d", model, count, want)
				}
			}
			var wantUsers, wantTokens int64
			if scenario.block {
				wantUsers, wantTokens = 1, 3
			}
			assertCount(&models.User{}, "id = ?", user.ID, wantUsers)
			assertCount(&models.RefreshToken{}, "user_id = ?", user.ID, wantTokens)
			assertCount(&models.User{}, "id = ?", other.ID, 1)
			assertCount(&models.RefreshToken{}, "user_id = ?", other.ID, 1)
			var entries []models.AuditLog
			if err := tx.Order("id").Find(&entries).Error; err != nil {
				t.Fatal(err)
			}
			wantEntries := 1
			if scenario.withTokens {
				wantEntries += 3
			}
			if scenario.block {
				wantEntries = 0
			}
			if len(entries) != wantEntries {
				t.Fatalf("audit entries = %d, want %d", len(entries), wantEntries)
			}
			for i, entry := range entries {
				if entry.Action != "delete" || strings.Contains(entry.OldData, "secret") || strings.Contains(entry.OldData, "refresh_token") {
					t.Fatalf("incorrect or sensitive audit entry: %+v", entry)
				}
				if i == 0 {
					if entry.ResourceType != "customusers" || entry.GroupStatus != "completed" || entry.ChildCount != len(entries)-1 {
						t.Fatalf("incorrect audit group: %+v", entry)
					}
				} else if entry.ResourceType != "refreshtokens" || entry.ParentID == nil || *entry.ParentID != entries[0].ID {
					t.Fatalf("incorrect child audit entry: %+v", entry)
				}
			}
		})
	}
}
