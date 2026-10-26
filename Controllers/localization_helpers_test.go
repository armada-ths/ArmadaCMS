package controllers

import (
	"ArmadaCMS/main/models"
	"encoding/json"
	"mime/multipart"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
)

func TestMultipartTranslations(t *testing.T) {
	old := "Gammal"
	for _, input := range []struct {
		present bool
		value   string
	}{{false, ""}, {true, ""}, {true, "  "}, {true, " Svenska "}} {
		req := httptest.NewRequest("PUT", "/", nil)
		req.MultipartForm = &multipart.Form{Value: map[string][]string{}}
		req.Form = map[string][]string{}
		if input.present {
			req.MultipartForm.Value["titleSv"] = []string{input.value}
			req.Form["titleSv"] = []string{input.value}
		}
		item := models.Blogpost{TitleSv: &old}
		updates := map[string]any{"title_sv": "stale"}
		mergeMultipartTranslations(req, &item, updates)
		if !input.present {
			if len(updates) != 0 || *item.TitleSv != old {
				t.Fatal("omitted translation changed")
			}
			continue
		}
		if strings.TrimSpace(input.value) == "" {
			if item.TitleSv != nil {
				t.Fatal("translation was not cleared")
			}
		} else if item.TitleSv == nil || *item.TitleSv != "Svenska" {
			t.Fatal("translation was not normalized")
		}
	}
}

func TestRecruitmentResponsePreservesSwedishCopyAndGroupKeys(t *testing.T) {
	name, description, team := "Utvecklare", "Bygg webbplatsen", "IT-teamet"
	result := mapRecruitmentPeriodToResponse(models.RecruitmentPeriod{Name: "Recruitment", NameSv: &name, Roles: []models.RecruitmentRole{{Name: "Developer", NameSv: &name, DescriptionSv: &description, Team: &models.Team{TeamName: "IT", TeamNameSv: &team}}}})
	if result.Groups["IT"][0].NameSv == nil || *result.Groups["IT"][0].NameSv != name || result.GroupNamesSv["IT"] != team {
		t.Fatalf("%#v", result)
	}
	data, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"nameSv", "descriptionSv", "groupNamesSv"} {
		if !strings.Contains(string(data), key) {
			t.Fatalf("missing %s: %s", key, data)
		}
	}
}

func TestDecodeTranslationFields(t *testing.T) {
	for _, input := range []string{`{}`, `{"nameSv":null}`, `{"nameSv":""}`, `{"nameSv":"  Svenska  "}`} {
		var item models.Industry
		raw, err := decodeWithFields(strings.NewReader(input), &item)
		if err != nil {
			t.Fatal(err)
		}
		updates := map[string]any{}
		mergeJSONTranslations(raw, &item, updates)
		if len(raw) == 0 {
			if len(updates) != 0 {
				t.Fatal(updates)
			}
			continue
		}
		got := updates["name_sv"].(*string)
		if strings.Contains(input, "Svenska") {
			if got == nil || *got != "Svenska" {
				t.Fatal(updates)
			}
		} else if !reflect.ValueOf(got).IsNil() {
			t.Fatal("not cleared")
		}
	}
}
