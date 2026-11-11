package controllers

import (
	"ArmadaCMS/main/auth"
	"ArmadaCMS/main/db"
	"ArmadaCMS/main/models"
	"ArmadaCMS/main/utils"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"gorm.io/gorm"
)

func TestEventroSyncBlocksDatabaseFlow(t *testing.T) {
	database := eventroSyncTestDatabase(t)
	syncRequest := httptest.NewRequest(http.MethodGet, "/eventrosync?fairId=fixture", nil)
	const label = "Eventro example"
	parentKey := "parent"
	parent := models.RecruitmentPeriod{EventroID: &parentKey, Name: "Parent period"}
	if err := database.Create(&parent).Error; err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		resource, column, jsonKey string
		handler                   http.HandlerFunc
	}{
		{"exhibitors", "eventro_id", "eventroId", DeleteExhibitor},
		{"events", "eventro_id", "eventroId", DeleteEvent},
		{"profiles", "eventro_key", "eventro_key", DeleteProfile},
		{"fairdates", "eventro_id", "eventroId", DeleteFairDateConfig},
		{"recruitmentperiods", "eventro_id", "eventroId", DeleteRecruitmentPeriod},
		{"recruitmentroles", "eventro_role_id", "eventroRoleId", DeleteRecruitmentRole},
	} {
		t.Run(test.resource, func(t *testing.T) {
			key := "blocked-" + test.resource
			sync := func(key string) (bool, error) {
				switch test.resource {
				case "exhibitors":
					created, _, err := syncEventroExhibitorWithAudit(syncRequest, eventroExhibitorResponse{ID: key, Organization: eventroOrg{Name: label}}, "fixture", true)
					return created, err
				case "fairdates":
					stats, err := syncFairDateConfigWithAudit(syncRequest, models.FairDateConfig{EventroID: &key, Description: label})
					if stats.skipped == 1 {
						return false, errEventroSyncSkipped
					}
					return stats.inserted == 1, err
				case "recruitmentroles":
					var created bool
					err := database.Transaction(func(tx *gorm.DB) error {
						var err error
						created, _, err = upsertRecruitmentRole(tx, syncRequest, parent.ID, eventroRecruitmentRoleDetail{ID: key, Role: label})
						return err
					})
					return created, err
				}
				entity := eventroResourceSpecs[test.resource].newModel()
				if err := applyEventroStructValue(entity, test.jsonKey, json.RawMessage(strconv.Quote(key))); err != nil {
					t.Fatal(err)
				}
				if err := applyEventroStructValue(entity, "name", json.RawMessage(strconv.Quote(label))); err != nil {
					t.Fatal(err)
				}
				var created bool
				err := database.Transaction(func(tx *gorm.DB) error {
					var err error
					_, created, _, err = syncEventroRecordWithAudit(tx, syncRequest, test.resource, entity, test.column, key, nil, nil)
					return err
				})
				return created, err
			}
			if created, err := sync(key); err != nil || !created {
				t.Fatalf("initial sync: created=%v, err=%v", created, err)
			}
			record := eventroResourceSpecs[test.resource].newModel()
			if err := database.Where(test.column+" = ?", key).First(record).Error; err != nil {
				t.Fatal(err)
			}
			var child models.RecruitmentRole
			if period, ok := record.(*models.RecruitmentPeriod); ok {
				childKey := "child-role"
				child = models.RecruitmentRole{EventroRoleID: &childKey, RecruitmentID: period.ID, Name: "Child"}
				if err := database.Create(&child).Error; err != nil {
					t.Fatal(err)
				}
			}

			response := eventroBlockTestRequest(t, test.handler, http.MethodDelete, "/"+test.resource, getResourceID(record), nil)
			if response.Code != http.StatusNoContent {
				t.Fatalf("delete: %d %s", response.Code, response.Body)
			}
			if child.ID != 0 {
				var roleBlock models.EventroSyncBlock
				if err := database.Where("resource = ? AND eventro_key = ?", "recruitmentroles", *child.EventroRoleID).First(&roleBlock).Error; err != nil {
					t.Fatal(err)
				}
				for _, candidate := range []struct {
					periodID uint
					key      string
				}{
					{parent.ID, *child.EventroRoleID}, {child.RecruitmentID, "new-role"},
				} {
					err := database.Transaction(func(tx *gorm.DB) error {
						_, _, err := upsertRecruitmentRole(tx, syncRequest, candidate.periodID, eventroRecruitmentRoleDetail{ID: candidate.key, Group: "Skipped team"})
						return err
					})
					if !errors.Is(err, errEventroSyncSkipped) {
						t.Fatalf("unexpected role result: %v", err)
					}
				}
				var teams int64
				if err := database.Model(&models.Team{}).Where("team_name = ?", "Skipped team").Count(&teams).Error; err != nil || teams != 0 {
					t.Fatalf("role skip created a team: count=%d, err=%v", teams, err)
				}
			}

			for range 2 {
				if created, err := sync(key); created || !errors.Is(err, errEventroSyncSkipped) {
					t.Fatalf("blocked sync: created=%v, err=%v", created, err)
				}
			}
			var live int64
			if err := database.Model(record).Where(test.column+" = ?", key).Count(&live).Error; err != nil || live != 0 {
				t.Fatalf("blocked record is live: count=%d, err=%v", live, err)
			}
			if created, err := sync("other-" + key); err != nil || !created {
				t.Fatalf("unblocked sync: created=%v, err=%v", created, err)
			}

			list := auth.Middleware(auth.RequirePermission("eventrosync.access", ListEventroSyncBlocks))
			unblock := auth.Middleware(auth.RequirePermission("eventrosync.access", DeleteEventroSyncBlock))
			listPath := "/eventrosyncblocks?resource=" + test.resource
			blockPath := "/eventrosyncblocks/"
			permissions := []string{"eventrosync.access", test.resource + ".delete"}
			response = eventroBlockTestRequest(t, list.ServeHTTP, http.MethodGet, listPath, "", permissions)
			var blocks []models.EventroSyncBlock
			if response.Code != http.StatusOK || json.Unmarshal(response.Body.Bytes(), &blocks) != nil {
				t.Fatalf("list: %d %s", response.Code, response.Body)
			}
			var block models.EventroSyncBlock
			for _, item := range blocks {
				if item.EventroKey == key {
					block = item
				}
			}
			if block.ID == 0 || block.Label != label || block.DeletedAt.IsZero() {
				t.Fatalf("missing block for %s: %+v", key, block)
			}
			blockID := getResourceID(&block)
			for _, insufficient := range [][]string{{permissions[0]}, {permissions[1]}} {
				response = eventroBlockTestRequest(t, list.ServeHTTP, http.MethodGet, listPath, "", insufficient)
				if response.Code != http.StatusForbidden {
					t.Fatalf("forbidden list: %d %s", response.Code, response.Body)
				}
				response = eventroBlockTestRequest(t, unblock.ServeHTTP, http.MethodDelete, blockPath+blockID, blockID, insufficient)
				if response.Code != http.StatusForbidden {
					t.Fatalf("forbidden unblock: %d %s", response.Code, response.Body)
				}
			}
			response = eventroBlockTestRequest(t, unblock.ServeHTTP, http.MethodDelete, blockPath+blockID, blockID, permissions)
			if response.Code != http.StatusNoContent {
				t.Fatalf("unblock: %d %s", response.Code, response.Body)
			}
			var logged int64
			if err := database.Model(&models.AuditLog{}).Where("resource_type = ? AND resource_id = ? AND action IN ?", "eventro_sync_blocks", blockID, []string{"create", "delete"}).Count(&logged).Error; err != nil || logged != 2 {
				t.Fatalf("block audit: count=%d, err=%v", logged, err)
			}
			if created, err := sync(key); err != nil || !created {
				t.Fatalf("sync after unblock: created=%v, err=%v", created, err)
			}
		})
	}

	t.Run("rollback keeps the record and discards its block", func(t *testing.T) {
		event := models.Event{EventroID: "rollback", Name: "Rollback"}
		if err := database.Create(&event).Error; err != nil {
			t.Fatal(err)
		}
		response := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodDelete, "/events/"+getResourceID(&event), nil)
		writeDeleteResponseWithAudit(response, request, "events", getResourceID(&event), "not found", nil,
			func(*gorm.DB, *models.Event) error { return errors.New("injected delete failure") })
		if response.Code != http.StatusInternalServerError {
			t.Fatalf("expected failure: %d %s", response.Code, response.Body)
		}
		if err := database.First(&event, event.ID).Error; err != nil {
			t.Fatal(err)
		}
		var blocks int64
		if err := database.Model(&models.EventroSyncBlock{}).Where("eventro_key = ?", event.EventroID).Count(&blocks).Error; err != nil || blocks != 0 {
			t.Fatalf("rollback left a block: count=%d, err=%v", blocks, err)
		}
	})

	t.Run("deleting a CMS-only record leaves no block", func(t *testing.T) {
		profile := models.Profile{Name: "CMS-only profile"}
		if err := database.Create(&profile).Error; err != nil {
			t.Fatal(err)
		}
		response := eventroBlockTestRequest(t, DeleteProfile, http.MethodDelete, "/profiles", getResourceID(&profile), nil)
		if response.Code != http.StatusNoContent {
			t.Fatalf("delete: %d %s", response.Code, response.Body)
		}
		var blocks int64
		if err := database.Model(&models.EventroSyncBlock{}).Where("label = ?", profile.Name).Count(&blocks).Error; err != nil || blocks != 0 {
			t.Fatalf("CMS-only delete left a block: count=%d, err=%v", blocks, err)
		}
	})
}

func eventroBlockTestRequest(t *testing.T, handler http.HandlerFunc, method, path, id string, perms []string) *httptest.ResponseRecorder {
	t.Helper()
	request := httptest.NewRequest(method, path, nil)
	if id != "" {
		request = mux.SetURLVars(request, map[string]string{"id": id})
	}
	if perms != nil {
		token, err := utils.GenerateAccessToken(0, nil, perms)
		if err != nil {
			t.Fatal(err)
		}
		request.Header.Set("Authorization", "Bearer "+token)
	}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	return response
}

func TestEventroSyncBlockConcurrentDelete(t *testing.T) {
	database := openEventroSyncTestDatabase(t)
	previous := db.DB
	db.DB = database
	t.Cleanup(func() { db.DB = previous })
	t.Setenv("REVALIDATION_URL", "")
	key := fmt.Sprintf("eventro-concurrent-test-%d", time.Now().UnixNano())
	profile := models.Profile{EventroKey: &key, Name: key}
	if err := database.Create(&profile).Error; err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		database.Where("eventro_key = ?", key).Delete(&models.EventroSyncBlock{})
		database.Where("eventro_key = ?", key).Delete(&models.Profile{})
		database.Where("resource_type IN ? AND (old_data ->> 'eventro_key' = ? OR new_data ->> 'eventroKey' = ?)", []string{"profiles", "eventro_sync_blocks"}, key, key).Delete(&models.AuditLog{})
	})

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	enteredDelete := make(chan struct{})
	finishDelete := make(chan struct{})
	defer func() {
		select {
		case <-finishDelete:
		default:
			close(finishDelete)
		}
	}()
	deleteStatus := make(chan int, 1)
	go func() {
		request := httptest.NewRequest(http.MethodDelete, "/profiles/"+getResourceID(&profile), nil).WithContext(ctx)
		response := httptest.NewRecorder()
		writeDeleteResponseWithAudit(response, request, "profiles", getResourceID(&profile), "not found", nil,
			func(*gorm.DB, *models.Profile) error {
				close(enteredDelete)
				<-finishDelete
				return nil
			})
		deleteStatus <- response.Code
	}()
	select {
	case <-enteredDelete:
	case status := <-deleteStatus:
		t.Fatalf("delete exited before holding the lock: %d", status)
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}

	syncPID := make(chan int, 1)
	syncError := make(chan error, 1)
	go func() {
		syncError <- database.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			var pid int
			if err := tx.Raw("SELECT pg_backend_pid()").Scan(&pid).Error; err != nil {
				return err
			}
			syncPID <- pid
			incoming := models.Profile{EventroKey: &key, Name: key}
			_, _, _, err := syncEventroRecordWithAudit(tx, httptest.NewRequest(http.MethodGet, "/eventromembers", nil), "profiles", &incoming, "eventro_key", key, nil, nil)
			return err
		})
	}()
	var pid int
	select {
	case pid = <-syncPID:
	case err := <-syncError:
		t.Fatalf("sync exited before waiting for the lock: %v", err)
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
	for {
		var waiting bool
		err := database.WithContext(ctx).Raw("SELECT EXISTS (SELECT 1 FROM pg_locks WHERE pid = ? AND locktype = 'advisory' AND NOT granted)", pid).Scan(&waiting).Error
		if err != nil {
			t.Fatal(err)
		}
		if waiting {
			break
		}
		select {
		case err := <-syncError:
			t.Fatalf("sync did not wait for delete: %v", err)
		case <-ctx.Done():
			t.Fatal(ctx.Err())
		case <-time.After(10 * time.Millisecond):
		}
	}
	close(finishDelete)
	if status := <-deleteStatus; status != http.StatusNoContent {
		t.Fatalf("delete: %d", status)
	}
	if err := <-syncError; !errors.Is(err, errEventroSyncSkipped) {
		t.Fatalf("sync after concurrent delete: %v", err)
	}
	if err := database.First(&profile, profile.ID).Error; !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("deleted profile still present: %v", err)
	}
}

func TestEventroExhibitorCategoryToggleDatabaseFlow(t *testing.T) {
	database := eventroSyncTestDatabase(t)
	request := httptest.NewRequest(http.MethodGet, "/eventroexhibitors?fairId=fixture", nil)
	incoming := eventroExhibitorResponse{
		ID:           "category-toggle",
		Organization: eventroOrg{Name: "Initial exhibitor"},
		Catalogue: eventroCatalogue{
			Industries:  []string{"Original industry"},
			Employments: []string{"Original employment"},
			Educations:  []string{"Original programme"},
		},
	}
	sync := func(include bool) {
		t.Helper()
		if _, _, err := syncEventroExhibitorWithAudit(request, incoming, "fixture", include); err != nil {
			t.Fatal(err)
		}
	}
	read := func() models.Exhibitor {
		t.Helper()
		var current models.Exhibitor
		if err := eventroReviewQuery(database, "exhibitors").Where("eventro_id = ?", incoming.ID).First(&current).Error; err != nil {
			t.Fatal(err)
		}
		return current
	}
	sync(false)
	current := read()
	if len(current.Industries)+len(current.Employments)+len(current.Programs) != 0 {
		t.Fatal("new exhibitor gained categories while the toggle was off")
	}
	sync(true)
	incoming.Organization.Name = "Updated exhibitor"
	incoming.Catalogue = eventroCatalogue{Industries: []string{"New industry"}}
	sync(false)
	current = read()
	if current.Name != incoming.Organization.Name ||
		len(current.Industries) != 1 || current.Industries[0].Name != "Original industry" ||
		len(current.Employments) != 1 || len(current.Programs) != 1 {
		t.Fatalf("sync with categories off: %+v", current)
	}
	var count int64
	if err := database.Model(&models.Industry{}).Where("name = ?", "New industry").Count(&count).Error; err != nil || count != 0 {
		t.Fatalf("sync with categories off created a taxonomy row: count=%d err=%v", count, err)
	}
	sync(true)
	current = read()
	if len(current.Industries) != 1 || current.Industries[0].Name != "New industry" ||
		len(current.Employments)+len(current.Programs) != 0 {
		t.Fatalf("category sync did not resume: %+v", current)
	}
}
