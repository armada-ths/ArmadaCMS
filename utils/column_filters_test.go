package utils

import (
	"ArmadaCMS/main/models"
	"reflect"
	"strings"
	"testing"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func columnFilterDB(t *testing.T) *gorm.DB {
	t.Helper()
	database, err := gorm.Open(postgres.Open("host=localhost user=postgres dbname=column_filter_test sslmode=disable"), &gorm.Config{DryRun: true, DisableAutomaticPing: true})
	if err != nil {
		t.Fatal(err)
	}
	return database
}

func TestColumnFiltersAliasesAndTypes(t *testing.T) {
	for _, test := range []struct {
		name    string
		model   any
		filters map[string]string
		column  string
		value   any
	}{
		{"JSON alias", &models.TimelineEntry{}, map[string]string{"eraId": "7"}, `"timeline_entries"."era_id"`, uint64(7)},
		{"database name", &models.TimelineEntry{}, map[string]string{"era_id": "7"}, `"timeline_entries"."era_id"`, uint64(7)},
		{"false", &models.FeatureFlag{}, map[string]string{"enabled": "false"}, `"feature_flags"."enabled"`, false},
		{"zero", &models.TimelineEra{}, map[string]string{"sort_order": "0"}, `"timeline_eras"."sort_order"`, int64(0)},
		{"nullable foreign key", &models.Profile{}, map[string]string{"team_id": "3"}, `"profiles"."team_id"`, int64(3)},
		{"literal exact text", &models.Industry{}, map[string]string{"name": "  A%_' OR 1=1 --  "}, `"industries"."name"`, "  A%_' OR 1=1 --  "},
		{"named string", &models.Exhibitor{}, map[string]string{"tier": "Gold"}, `"exhibitors"."tier"`, "Gold"},
		{"timestamp", &models.Event{}, map[string]string{"eventStart": "2026-11-01T13:30:00+01:00"}, `"events"."event_start"`, time.Date(2026, 11, 1, 12, 30, 0, 0, time.UTC)},
		{"precise timestamp", &models.User{}, map[string]string{"created_at": "2026-11-01T13:30:00.123456+01:00"}, `"users"."created_at"`, time.Date(2026, 11, 1, 12, 30, 0, 123456000, time.UTC)},
	} {
		t.Run(test.name, func(t *testing.T) {
			query, err := ApplyColumnFilters(columnFilterDB(t).Model(test.model), test.filters)
			if err != nil {
				t.Fatal(err)
			}
			var count int64
			statement := query.Count(&count).Statement
			if !strings.Contains(statement.SQL.String(), test.column+" = $1") {
				t.Fatalf("unexpected predicate: %s", statement.SQL.String())
			}
			if len(statement.Vars) != 1 {
				t.Fatalf("unexpected bindings: %#v", statement.Vars)
			}
			if expected, ok := test.value.(time.Time); ok {
				actual, ok := statement.Vars[0].(time.Time)
				if !ok || !actual.Equal(expected) {
					t.Fatalf("unexpected timestamp: %#v", statement.Vars[0])
				}
			} else if !reflect.DeepEqual(statement.Vars[0], test.value) {
				t.Fatalf("unexpected binding: %#v", statement.Vars[0])
			}
		})
	}
}

func TestColumnFiltersRejectInvalidFieldsAndValues(t *testing.T) {
	for _, test := range []struct {
		model      any
		key, value string
	}{
		{&models.User{}, "password", "secret"},
		{&models.User{}, "roles", "admin"},
		{&models.Blogpost{}, "image_urls", "[]"},
		{&models.Role{}, "permissions", "[\"*\"]"},
		{&models.TimelineEntry{}, "eraTitle", "Legacy"},
		{&models.Industry{}, "missing_column", "x"},
		{&models.Industry{}, "name OR 1=1", "x"},
		{&models.Industry{}, "id", "1.5"},
		{&models.Industry{}, "id", "-1"},
		{&models.Industry{}, "id", "9223372036854775808"},
		{&models.FeatureFlag{}, "enabled", "maybe"},
		{&models.Event{}, "event_start", "tomorrow"},
	} {
		t.Run(reflect.TypeOf(test.model).Elem().Name()+"/"+test.key+"/"+test.value, func(t *testing.T) {
			if _, err := ApplyColumnFilters(columnFilterDB(t).Model(test.model), map[string]string{test.key: test.value}); err == nil {
				t.Fatal("invalid filter was accepted")
			}
		})
	}
}

func TestColumnFiltersComposeWithSearchAndVisibility(t *testing.T) {
	query := columnFilterDB(t).Model(&models.Blogpost{}).Where("published = ?", true)
	query = ApplySearch(query, "Armada", "title", "text")
	query, err := ApplyColumnFilters(query, map[string]string{"q": "Armada", "author": "Tester", "published": "false", "title": ""})
	if err != nil {
		t.Fatal(err)
	}
	var count int64
	countStatement := query.Session(&gorm.Session{}).Count(&count).Statement
	sql := countStatement.SQL.String()
	if !strings.Contains(sql, `published = $1 AND (CAST("title" AS TEXT) ILIKE $2 ESCAPE '!' OR CAST("text" AS TEXT) ILIKE $3 ESCAPE '!') AND "blogposts"."author" = $4 AND "blogposts"."published" = $5`) {
		t.Fatalf("search or filters bypassed visibility: %s", sql)
	}
	if !reflect.DeepEqual(countStatement.Vars, []any{true, "%Armada%", "%Armada%", "Tester", false}) {
		t.Fatalf("unexpected bindings: %#v", countStatement.Vars)
	}
	var records []models.Blogpost
	page := query.Offset(25).Limit(25).Find(&records).Statement
	if !strings.Contains(page.SQL.String(), `"blogposts"."author" = $4`) || !strings.Contains(page.SQL.String(), "LIMIT") || strings.Contains(sql, "LIMIT") {
		t.Fatalf("count/page filter mismatch: %s / %s", sql, page.SQL.String())
	}
}

func TestColumnFiltersSupportAllResourceModels(t *testing.T) {
	for _, model := range []any{
		&models.Industry{}, &models.Program{}, &models.Employment{}, &models.Team{},
		&models.Role{}, &models.User{}, &models.Profile{}, &models.Event{}, &models.Exhibitor{},
		&models.FairDateConfig{}, &models.FeatureFlag{}, &models.HighlightCard{}, &models.Blogpost{},
		&models.TimelineEntry{}, &models.TimelineEra{}, &models.RecruitmentPeriod{}, &models.RecruitmentRole{}, &models.PhotoEvent{},
	} {
		t.Run(reflect.TypeOf(model).Elem().Name(), func(t *testing.T) {
			query, err := ApplyColumnFilters(columnFilterDB(t).Model(model), map[string]string{"id": "7"})
			if err != nil {
				t.Fatal(err)
			}
			var count int64
			statement := query.Count(&count).Statement
			if !strings.Contains(statement.SQL.String(), `."id" = $1`) || !reflect.DeepEqual(statement.Vars, []any{uint64(7)}) {
				t.Fatalf("missing ID filter: %s %#v", statement.SQL.String(), statement.Vars)
			}
		})
	}
}
