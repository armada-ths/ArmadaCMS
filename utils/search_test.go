package utils

import (
	"reflect"
	"strings"
	"testing"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func searchTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	database, err := gorm.Open(postgres.Open("host=localhost user=postgres dbname=search_test sslmode=disable"), &gorm.Config{DryRun: true, DisableAutomaticPing: true})
	if err != nil {
		t.Fatal(err)
	}
	return database
}

func TestApplySearchLiteralPatterns(t *testing.T) {
	for _, test := range []struct{ term, want string }{
		{"  Armada  ", "%Armada%"},
		{"100%_!", "%100!%!_!!%"},
		{`C:\photos`, `%C:\photos%`},
		{"ÅÄÖ", "%ÅÄÖ%"},
		{"' OR 1=1 --", "%' OR 1=1 --%"},
	} {
		t.Run(test.term, func(t *testing.T) {
			query := ApplySearch(searchTestDB(t).Table("items"), test.term, "name", "description")
			var count int64
			statement := query.Count(&count).Statement
			if strings.Contains(statement.SQL.String(), test.term) {
				t.Fatal("search input was interpolated into SQL")
			}
			if !reflect.DeepEqual(statement.Vars, []any{test.want, test.want}) {
				t.Fatalf("unexpected bindings: %#v", statement.Vars)
			}
			if !strings.Contains(statement.SQL.String(), "ILIKE") || !strings.Contains(statement.SQL.String(), "ESCAPE '!'") {
				t.Fatalf("unexpected SQL: %s", statement.SQL.String())
			}
		})
	}
}

func TestApplySearchPreservesFiltersAndPagination(t *testing.T) {
	query := searchTestDB(t).Table("timeline_entries").Where("published = ?", true)
	query = ApplySearch(query, "fair", "timeline_entries.id", "timeline_entries.title", "timeline_eras.title").Where("era_id = ?", 7)
	var total int64
	count := query.Session(&gorm.Session{}).Count(&total).Statement
	expected := `WHERE published = $1 AND (CAST("timeline_entries"."id" AS TEXT) ILIKE $2 ESCAPE '!' OR CAST("timeline_entries"."title" AS TEXT) ILIKE $3 ESCAPE '!' OR CAST("timeline_eras"."title" AS TEXT) ILIKE $4 ESCAPE '!') AND era_id = $5`
	if !strings.Contains(count.SQL.String(), expected) {
		t.Fatalf("search must not bypass other filters: %s", count.SQL.String())
	}
	var items []struct{ ID int }
	page := query.Offset(25).Limit(25).Find(&items).Statement
	if !strings.Contains(page.SQL.String(), expected) || !strings.Contains(page.SQL.String(), "LIMIT") || !strings.Contains(page.SQL.String(), "OFFSET") {
		t.Fatalf("unexpected page query: %s", page.SQL.String())
	}
	if strings.Contains(count.SQL.String(), "LIMIT") || strings.Contains(count.SQL.String(), "OFFSET") {
		t.Fatal("count was paginated")
	}
	if !reflect.DeepEqual(count.Vars, []any{true, "%fair%", "%fair%", "%fair%", 7}) {
		t.Fatalf("unexpected bindings: %#v", count.Vars)
	}
}

func TestApplySearchEmpty(t *testing.T) {
	query := searchTestDB(t).Table("items")
	for _, term := range []string{"", " \t\n "} {
		if ApplySearch(query, term, "name") != query {
			t.Fatal("blank search should preserve original query")
		}
	}
	if ApplySearch(query, "test") != query {
		t.Fatal("no columns should preserve original query")
	}
}
