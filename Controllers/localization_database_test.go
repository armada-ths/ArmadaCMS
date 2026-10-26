package controllers

import (
	"ArmadaCMS/main/models"
	"database/sql"
	"fmt"
	"net/url"
	"os"
	"reflect"
	"strings"
	"testing"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// This fixture uses a fresh local database and rolls all schema/data back.
func TestPublicTranslationPersistence(t *testing.T) {
	testTranslationPersistence(t, []any{&models.HighlightCard{}, &models.Blogpost{}, &models.Event{}, &models.Exhibitor{}, &models.Employment{}, &models.Industry{}, &models.Program{}, &models.Team{}, &models.Profile{}, &models.RecruitmentPeriod{}, &models.RecruitmentRole{}, &models.FairDateConfig{}, &models.TimelineEra{}, &models.TimelineEntry{}})
}

func testTranslationPersistence(t *testing.T, resources []any) {
	t.Helper()
	dsn := os.Getenv("TRANSLATION_TEST_DSN")
	if dsn == "" {
		t.Skip("TRANSLATION_TEST_DSN is unset")
	}
	address, err := url.Parse(dsn)
	if err != nil || (address.Scheme != "postgres" && address.Scheme != "postgresql") || address.Path != "/translation_test" || (address.Hostname() != "localhost" && address.Hostname() != "127.0.0.1" && address.Hostname() != "::1") {
		t.Fatal("translation tests require a local translation_test database")
	}
	database, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	pool, err := database.DB()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = pool.Close() })
	tx := database.Begin()
	if tx.Error != nil {
		t.Fatal(tx.Error)
	}
	defer tx.Rollback()
	for _, resource := range resources {
		statement := &gorm.Statement{DB: tx}
		if err := statement.Parse(resource); err != nil {
			t.Fatal(err)
		}
		table := statement.Schema.Table
		columnByJSON := map[string]string{}
		for _, field := range statement.Schema.Fields {
			columnByJSON[strings.Split(field.Tag.Get("json"), ",")[0]] = field.DBName
		}
		if tx.Migrator().HasTable(table) {
			t.Fatalf("fixture table %s already exists; use a fresh database", table)
		}
		columns := []string{`id bigint PRIMARY KEY`, `original_text text NOT NULL`}
		fields := translationValues(resource)
		for key := range fields {
			column := columnByJSON[key]
			columns = append(columns, fmt.Sprintf(`"%s" text`, column))
		}
		if err := tx.Exec(fmt.Sprintf(`CREATE TABLE "%s" (%s)`, table, strings.Join(columns, ","))).Error; err != nil {
			t.Fatal(err)
		}
		if err := tx.Exec(fmt.Sprintf(`INSERT INTO "%s" (id,original_text) VALUES (1,'Original English')`, table)).Error; err != nil {
			t.Fatal(err)
		}
		reflect.ValueOf(resource).Elem().FieldByName("ID").SetUint(1)
		for key := range fields {
			column := columnByJSON[key]
			for _, body := range []string{`{}`, fmt.Sprintf(`{%q:null}`, key), fmt.Sprintf(`{%q:""}`, key), fmt.Sprintf(`{%q:"  "}`, key), fmt.Sprintf(`{%q:" Svenska "}`, key)} {
				t.Run(table+"/"+key+"/"+body, func(t *testing.T) {
					if err := tx.Exec(fmt.Sprintf(`UPDATE "%s" SET "%s"='Previous Swedish' WHERE id=1`, table, column)).Error; err != nil {
						t.Fatal(err)
					}
					raw, err := decodeWithFields(strings.NewReader(body), resource)
					if err != nil {
						t.Fatal(err)
					}
					updates := map[string]any{}
					mergeJSONTranslations(raw, resource, updates)
					if len(updates) > 0 {
						if err := tx.Model(resource).Updates(updates).Error; err != nil {
							t.Fatal(err)
						}
					}
					var row struct {
						Original    string
						Translation sql.NullString
					}
					if err := tx.Raw(fmt.Sprintf(`SELECT original_text AS original,"%s" AS translation FROM "%s" WHERE id=1`, column, table)).Scan(&row).Error; err != nil {
						t.Fatal(err)
					}
					if row.Original != "Original English" {
						t.Fatal("original changed")
					}
					if body == `{}` {
						if !row.Translation.Valid || row.Translation.String != "Previous Swedish" {
							t.Fatal("omitted translation changed")
						}
					} else if strings.Contains(body, "Svenska") {
						if !row.Translation.Valid || row.Translation.String != "Svenska" {
							t.Fatal("translation was not normalized")
						}
					} else if row.Translation.Valid {
						t.Fatalf("translation was not persisted as NULL: %q", row.Translation.String)
					}
				})
			}
		}
	}
}
