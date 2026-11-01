package controllers

import (
	"ArmadaCMS/main/models"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"reflect"
	"testing"
)

func TestHighlightSwedishPartialUpdates(t *testing.T) {
	database, err := gorm.Open(postgres.Open("host=localhost user=test dbname=test sslmode=disable"), &gorm.Config{DryRun: true, DisableAutomaticPing: true, SkipDefaultTransaction: true})
	if err != nil {
		t.Fatal(err)
	}
	for _, field := range []string{"titleSv", "subtitleSv", "descriptionSv", "linkTextSv"} {
		for _, value := range []any{nil, "", "   ", "  Svenska  "} {
			updates := BuildNormalizedSnakeCaseUpdateMap(map[string]any{field: value}, map[string]struct{}{field: {}}, nil)
			var want any
			if value == "  Svenska  " {
				want = "Svenska"
			}
			for _, got := range updates {
				if !reflect.DeepEqual(got, want) {
					t.Fatalf("%s: got %#v, want %#v", field, got, want)
				}
			}
			result := database.Model(&models.HighlightCard{ID: 1}).Updates(updates)
			if result.Error != nil {
				t.Fatal(result.Error)
			}
			if len(result.Statement.Vars) != 2 || !reflect.DeepEqual(result.Statement.Vars[0], want) {
				t.Fatalf("SQL did not write explicit value: %s %#v", result.Statement.SQL.String(), result.Statement.Vars)
			}
		}
	}
	if got := BuildNormalizedSnakeCaseUpdateMap(map[string]any{"title": "English"}, map[string]struct{}{"titleSv": {}}, nil); len(got) != 1 {
		t.Fatalf("omitted translation was not preserved: %#v", got)
	}
}
