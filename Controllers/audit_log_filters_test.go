package controllers

import (
	"net/url"
	"os"
	"reflect"
	"strings"
	"testing"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func TestAuditLogActivityFilterSQL(t *testing.T) {
	database, err := gorm.Open(postgres.Open("host=localhost user=postgres dbname=audit_log_test sslmode=disable"), &gorm.Config{DryRun: true, DisableAutomaticPing: true})
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		name                           string
		filters                        map[string]string
		hideAuth, hidePhotos, children bool
	}{
		{name: "default", hideAuth: true, hidePhotos: true},
		{name: "login", filters: map[string]string{"include_auth": "true"}, hidePhotos: true},
		{name: "photos", filters: map[string]string{"include_guest_photos": "true"}, hideAuth: true},
		{name: "both", filters: map[string]string{"include_auth": "true", "include_guest_photos": "true"}},
		{name: "false", filters: map[string]string{"include_auth": "false", "include_guest_photos": "false"}, hideAuth: true, hidePhotos: true},
		{name: "nonliteral true", filters: map[string]string{"include_auth": "TRUE", "include_guest_photos": "1"}, hideAuth: true, hidePhotos: true},
		{name: "children", filters: map[string]string{"parent_id": "42"}, children: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			query := auditLogQuery(database.Table("audit_logs"), test.filters)
			var total int64
			sql := query.Count(&total).Statement.SQL.String()
			for fragment, want := range map[string]bool{
				"request_path NOT IN": test.hideAuth,
				"resource_type <>":    test.hidePhotos,
				"parent_id IS NULL":   !test.children,
				"parent_id =":         test.children,
			} {
				if strings.Contains(sql, fragment) != want {
					t.Errorf("presence of %q should be %v: %s", fragment, want, sql)
				}
			}
		})
	}
}

type auditFilterRecord struct {
	ID            int
	ParentID      *int
	Action        string
	ResourceType  string
	ResourceID    string
	ActorUsername string
	ActorName     string
	RequestPath   string
	HTTPMethod    string
	GroupStatus   string
}

func (auditFilterRecord) TableName() string { return "audit_logs" }

// The optional integration test uses only a temporary table and rolls it back.
func TestAuditLogFiltersPostgres(t *testing.T) {
	dsn := os.Getenv("AUDIT_LOG_TEST_DSN")
	if dsn == "" {
		t.Skip("set AUDIT_LOG_TEST_DSN to a local audit_log_test database")
	}
	parsed, err := url.Parse(dsn)
	if err != nil || parsed.Path != "/audit_log_test" || (parsed.Hostname() != "localhost" && parsed.Hostname() != "127.0.0.1") {
		t.Fatal("AUDIT_LOG_TEST_DSN must target a local audit_log_test database")
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
	tx := database.Begin()
	if tx.Error != nil {
		t.Fatal(tx.Error)
	}
	t.Cleanup(func() { tx.Rollback() })
	if err := tx.Exec(`CREATE TEMP TABLE audit_logs (
        id integer PRIMARY KEY, parent_id integer, action text, resource_type text,
        resource_id text, actor_username text, actor_name text, request_path text,
        http_method text, group_status text
    ) ON COMMIT DROP`).Error; err != nil {
		t.Fatal(err)
	}
	parent := 1
	records := []auditFilterRecord{
		{ID: 1, ResourceType: "users", RequestPath: "/api/v1/users/1"},
		{ID: 2, ResourceType: "refresh_tokens", RequestPath: "/api/v1/login"},
		{ID: 3, ResourceType: "eventphotos", RequestPath: "/api/v1/eventphotos/3/moderate"},
		{ID: 4, ResourceType: "photoevents", RequestPath: "/api/v1/photoevents/4"},
		{ID: 5, ResourceType: "photoexports", RequestPath: "/api/v1/photoevents/4/exports"},
		{ID: 6, ParentID: &parent, ResourceType: "eventphotos", RequestPath: "/api/v1/eventphotos/6/moderate"},
		{ID: 7, ParentID: &parent, ResourceType: "refresh_tokens", RequestPath: "/api/v1/login"},
		{ID: 8, ResourceType: "refresh_tokens", RequestPath: "/api/v1/refreshAccessToken"},
		{ID: 9, ResourceType: "users", RequestPath: "/api/v1/me/password"},
	}
	for i := range records {
		records[i].ActorUsername = "tester"
		records[i].Action = "update"
	}
	if err := tx.Create(&records).Error; err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		name    string
		filters map[string]string
		ids     []int
	}{
		{"default", nil, []int{1, 4, 5}},
		{"login", map[string]string{"include_auth": "true"}, []int{1, 2, 4, 5, 8, 9}},
		{"photos", map[string]string{"include_guest_photos": "true"}, []int{1, 3, 4, 5}},
		{"both", map[string]string{"include_auth": "true", "include_guest_photos": "true"}, []int{1, 2, 3, 4, 5, 8, 9}},
		{"search cannot bypass defaults", map[string]string{"q": "tester"}, []int{1, 4, 5}},
		{"resource remains hidden without opt-in", map[string]string{"resource_type": "eventphotos"}, []int{}},
		{"combined filters", map[string]string{"include_guest_photos": "true", "resource_type": "eventphotos", "action": "update", "actor_username": "tester", "q": "moderate"}, []int{3}},
		{"children", map[string]string{"parent_id": "1"}, []int{6, 7}},
	} {
		t.Run(test.name, func(t *testing.T) {
			query := auditLogQuery(tx.Model(&auditFilterRecord{}), test.filters)
			var count int64
			if err := query.Count(&count).Error; err != nil {
				t.Fatal(err)
			}
			if count != int64(len(test.ids)) {
				t.Errorf("count = %d, want %d", count, len(test.ids))
			}
			var got []auditFilterRecord
			if err := query.Order("id ASC").Find(&got).Error; err != nil {
				t.Fatal(err)
			}
			ids := make([]int, len(got))
			for i := range got {
				ids[i] = got[i].ID
			}
			if !reflect.DeepEqual(ids, test.ids) {
				t.Errorf("ids = %v, want %v", ids, test.ids)
			}
			if len(test.ids) > 1 {
				var page []auditFilterRecord
				if err := query.Offset(1).Limit(1).Find(&page).Error; err != nil {
					t.Fatal(err)
				}
				if len(page) != 1 || page[0].ID != test.ids[1] {
					t.Errorf("wrong second-page result: %+v", page)
				}
			}
		})
	}
}
