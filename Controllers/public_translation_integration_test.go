package controllers

import (
	"ArmadaCMS/main/models"
	"ArmadaCMS/main/utils"
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strconv"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"gorm.io/gorm"
)

func TestPublicTranslationIntegration(t *testing.T) {
	database := eventroSyncTestDatabase(t)
	period := models.RecruitmentPeriod{Name: "English period"}
	era := models.TimelineEra{Title: "English era", TitleSv: utils.StringPtr("Svensk epokrubrik")}
	for _, model := range []any{&period, &era} {
		if err := database.Create(model).Error; err != nil {
			t.Fatal(err)
		}
	}
	resources := []struct {
		name  string
		model any
		list  http.HandlerFunc
	}{
		{"photoevents", &models.PhotoEvent{Name: "English photo event"}, ListPhotoEvents},
		{"highlightcards", &models.HighlightCard{Title: "English highlight"}, GetHighlightCards},
		{"blogposts", &models.Blogpost{Title: "English blog", Published: true}, GetBlogposts},
		{"events", &models.Event{Name: "English event", Show: true}, GetEvents},
		{"exhibitors", &models.Exhibitor{Name: "English exhibitor"}, GetExhibitors},
		{"employments", &models.Employment{Name: "English employment"}, GetEmployments},
		{"industries", &models.Industry{Name: "English industry"}, GetIndustries},
		{"programs", &models.Program{Name: "English program"}, GetPrograms},
		{"teams", &models.Team{TeamName: "English team"}, GetTeams},
		{"profiles", &models.Profile{Name: "English person", Title: "English role"}, GetProfiles},
		{"recruitmentperiods", &models.RecruitmentPeriod{Name: "English recruitment"}, GetRecruitmentPeriods},
		{"recruitmentroles", &models.RecruitmentRole{Name: "English role", RecruitmentID: period.ID}, GetRecruitmentRoles},
		{"fairdates", &models.FairDateConfig{Description: "English fair"}, GetFairDateConfigs},
		{"timeline-eras", &models.TimelineEra{Title: "English timeline"}, GetTimelineEras},
		{"timeline-entries", &models.TimelineEntry{Title: "English entry", EraID: era.ID}, GetTimelineEntries},
	}
	t.Run("search and exact filters include every Swedish field", func(t *testing.T) {
		for _, resource := range resources {
			fields := translationValues(resource.model)
			for key, dest := range fields {
				translated := "Svenska ÅÄÖ " + resource.name + " " + key
				*dest = &translated
			}
			if err := database.Create(resource.model).Error; err != nil {
				t.Fatal(err)
			}
			id := reflect.ValueOf(resource.model).Elem().FieldByName("ID").Uint()
			for key, dest := range fields {
				for _, filterKey := range []string{"q", utils.ToSnakeCase(key)} {
					t.Run(resource.name+"/"+key+"/"+filterKey, func(t *testing.T) {
						filter, _ := json.Marshal(map[string]string{filterKey: **dest})
						request := httptest.NewRequest(http.MethodGet, "/"+resource.name+"?filter="+url.QueryEscape(string(filter)), nil)
						response := httptest.NewRecorder()
						resource.list(response, request)
						if response.Code != http.StatusOK {
							t.Fatalf("%d: %s", response.Code, response.Body.String())
						}
						var rows []map[string]any
						if err := json.Unmarshal(response.Body.Bytes(), &rows); err != nil {
							t.Fatal(err)
						}
						if len(rows) != 1 || rows[0]["id"] != float64(id) || rows[0][key] != **dest {
							t.Fatalf("wrong translated record: %s", response.Body.String())
						}
					})
				}
			}
		}
	})
	t.Run("employment creation accepts Swedish text", func(t *testing.T) {
		response := httptest.NewRecorder()
		CreateEmployment(response, httptest.NewRequest(http.MethodPost, "/employments", bytes.NewBufferString(`{"name":"New employment","nameSv":"Ny anställningsform"}`)))
		if response.Code != http.StatusCreated {
			t.Fatalf("%d: %s", response.Code, response.Body.String())
		}
		var created models.Employment
		if err := json.Unmarshal(response.Body.Bytes(), &created); err != nil {
			t.Fatal(err)
		}
		if created.ID == 0 || created.NameSv == nil || *created.NameSv != "Ny anställningsform" {
			t.Fatalf("missing persisted translation: %#v", created)
		}
	})
	t.Run("joined era search includes Swedish titles", func(t *testing.T) {
		response := httptest.NewRecorder()
		filter, _ := json.Marshal(map[string]string{"q": "Svensk epokrubrik"})
		GetTimelineEntries(response, httptest.NewRequest(http.MethodGet, "/timeline-entries?filter="+url.QueryEscape(string(filter)), nil))
		var entries []models.TimelineEntry
		if response.Code != http.StatusOK {
			t.Fatalf("%d: %s", response.Code, response.Body.String())
		}
		if err := json.Unmarshal(response.Body.Bytes(), &entries); err != nil {
			t.Fatal(err)
		}
		if len(entries) != 1 || entries[0].EraTitle != era.Title || entries[0].EraTitleSv == nil || *entries[0].EraTitleSv != *era.TitleSv {
			t.Fatalf("missing translated era presentation: %s", response.Body.String())
		}
	})
	t.Run("cache invalidation follows shared translations", func(t *testing.T) {
		for _, test := range []struct {
			resource string
			model    any
			key      string
			update   http.HandlerFunc
			tags     []string
		}{
			{"teams", &models.Team{TeamName: "Cache team"}, "team_name_sv", UpdateTeam, []string{"organization", "recruitment"}},
			{"employments", &models.Employment{Name: "Cache employment"}, "nameSv", UpdateEmployment, []string{"employments", "exhibitors"}},
			{"industries", &models.Industry{Name: "Cache industry"}, "nameSv", UpdateIndustry, []string{"industries", "exhibitors"}},
			{"programs", &models.Program{Name: "Cache program"}, "nameSv", UpdateProgram, []string{"programs", "exhibitors"}},
		} {
			t.Run(test.resource, func(t *testing.T) {
				if err := database.Create(test.model).Error; err != nil {
					t.Fatal(err)
				}
				tags := make(chan string, 8)
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					var body struct {
						Tag string `json:"tag"`
					}
					if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
						http.Error(w, err.Error(), 400)
						return
					}
					tags <- body.Tag
				}))
				defer server.Close()
				t.Setenv("REVALIDATION_URL", server.URL)
				t.Setenv("REVALIDATION_SECRET", "local-fixture")
				body, _ := json.Marshal(map[string]string{test.key: "Svenskt cachevärde"})
				id := reflect.ValueOf(test.model).Elem().FieldByName("ID").Uint()
				request := mux.SetURLVars(httptest.NewRequest(http.MethodPut, "/"+test.resource+"/"+strconv.FormatUint(id, 10), bytes.NewReader(body)), map[string]string{"id": strconv.FormatUint(id, 10)})
				response := httptest.NewRecorder()
				test.update(response, request)
				if response.Code != http.StatusOK {
					t.Fatalf("%d: %s", response.Code, response.Body.String())
				}
				wanted := map[string]bool{}
				for _, tag := range test.tags {
					wanted[tag] = true
				}
				timer := time.NewTimer(3 * time.Second)
				defer timer.Stop()
				for len(wanted) > 0 {
					select {
					case tag := <-tags:
						delete(wanted, tag)
					case <-timer.C:
						t.Fatalf("missing revalidation tags: %v", wanted)
					}
				}
			})
		}
	})
	t.Run("multipart audit keeps the old Swedish text", func(t *testing.T) {
		for _, test := range []struct {
			resource string
			model    any
			update   http.HandlerFunc
			fields   map[string]string
		}{
			{"profiles", &models.Profile{Name: "Audit person", Title: "Original title"}, UpdateProfile, map[string]string{"name": "Audit person", "title": "Original title"}},
			{"blogposts", &models.Blogpost{Title: "Original title", Published: true}, UpdateBlogpost, map[string]string{"title": "Original title", "text": "Original body", "author": "Author"}},
		} {
			t.Run(test.resource, func(t *testing.T) {
				old := "Tidigare svensk titel"
				*translationValues(test.model)["titleSv"] = &old
				if err := database.Create(test.model).Error; err != nil {
					t.Fatal(err)
				}
				id := reflect.ValueOf(test.model).Elem().FieldByName("ID").Uint()
				for _, translated := range []string{"Ny svensk titel", ""} {
					var body bytes.Buffer
					writer := multipart.NewWriter(&body)
					for key, value := range test.fields {
						if err := writer.WriteField(key, value); err != nil {
							t.Fatal(err)
						}
					}
					if err := writer.WriteField("titleSv", translated); err != nil {
						t.Fatal(err)
					}
					if err := writer.Close(); err != nil {
						t.Fatal(err)
					}
					request := httptest.NewRequest(http.MethodPut, "/"+test.resource+"/"+strconv.FormatUint(id, 10), &body)
					request.Header.Set("Content-Type", writer.FormDataContentType())
					request = mux.SetURLVars(request, map[string]string{"id": strconv.FormatUint(id, 10)})
					response := httptest.NewRecorder()
					test.update(response, request)
					if response.Code != http.StatusOK {
						t.Fatalf("%d: %s", response.Code, response.Body.String())
					}
					var log models.AuditLog
					if err := database.Where("resource_type = ? AND resource_id = ? AND action = ?", test.resource, strconv.FormatUint(id, 10), "update").Order("id DESC").First(&log).Error; err != nil {
						t.Fatal(err)
					}
					var before, after map[string]any
					if err := json.Unmarshal([]byte(log.OldData), &before); err != nil {
						t.Fatal(err)
					}
					if err := json.Unmarshal([]byte(log.NewData), &after); err != nil {
						t.Fatal(err)
					}
					var expected any
					if translated != "" {
						expected = translated
					}
					if before["titleSv"] != old || after["titleSv"] != expected {
						t.Fatalf("incorrect translation audit: old=%s new=%s", log.OldData, log.NewData)
					}
					old = translated
				}
			})
		}
	})
}

func TestEventroTranslationPreservation(t *testing.T) {
	database := eventroSyncTestDatabase(t)
	period := models.RecruitmentPeriod{Name: "Parent period"}
	if err := database.Create(&period).Error; err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		resource, column, field string
		model                   any
	}{
		{"events", "eventro_id", "name", &models.Event{Name: "Original event"}},
		{"profiles", "eventro_key", "title", &models.Profile{Name: "Original person", Title: "Original role"}},
		{"fairdates", "eventro_id", "description", &models.FairDateConfig{Description: "Original fair"}},
		{"recruitmentperiods", "eventro_id", "name", &models.RecruitmentPeriod{Name: "Original period"}},
		{"recruitmentroles", "eventro_role_id", "description", &models.RecruitmentRole{Name: "Original role", Description: "Original description", RecruitmentID: period.ID}},
	} {
		t.Run(test.resource, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, "/eventrosync?fairId=fixture", nil)
			identity := "translation-" + test.resource
			identityField := map[string]string{"eventro_id": "eventroId", "eventro_key": "eventro_key", "eventro_role_id": "eventroRoleId"}[test.column]
			if err := applyEventroStructValue(test.model, identityField, json.RawMessage(strconv.Quote(identity))); err != nil {
				t.Fatal(err)
			}
			var id uint
			if err := database.Transaction(func(tx *gorm.DB) error {
				var err error
				id, _, _, err = syncEventroRecordWithAudit(tx, request, test.resource, test.model, test.column, identity, nil, nil)
				return err
			}); err != nil {
				t.Fatal(err)
			}
			expected := map[string]any{}
			for key := range translationValues(test.model) {
				expected[utils.ToSnakeCase(key)] = "Redaktionell svenska " + key
			}
			if err := database.Model(test.model).Updates(expected).Error; err != nil {
				t.Fatal(err)
			}
			if err := applyEventroStructValue(test.model, test.field, json.RawMessage(strconv.Quote("Changed English source"))); err != nil {
				t.Fatal(err)
			}
			if err := database.Transaction(func(tx *gorm.DB) error {
				_, _, _, err := syncEventroRecordWithAudit(tx, request, test.resource, test.model, test.column, identity, nil, nil)
				return err
			}); err != nil {
				t.Fatal(err)
			}
			current := eventroResourceSpecs[test.resource].newModel()
			if err := database.First(current, id).Error; err != nil {
				t.Fatal(err)
			}
			values := eventroStructValues(current, []string{test.field})
			if string(values[test.field]) != strconv.Quote("Changed English source") {
				t.Fatalf("source did not update: %s", values[test.field])
			}
			for key, dest := range translationValues(current) {
				if *dest == nil || **dest != expected[utils.ToSnakeCase(key)] {
					t.Fatalf("lost translation %s", key)
				}
			}
			var state models.EventroSyncState
			if err := database.First(&state, "resource = ? AND record_id = ?", test.resource, id).Error; err != nil {
				t.Fatal(err)
			}
			if len(state.Conflicts) != 0 || len(state.Overrides) != 0 {
				t.Fatalf("Swedish edits affected source ownership: %#v", state)
			}
		})
	}
	t.Run("exhibitor categories and copy", func(t *testing.T) {
		translated := "Svensk redaktionell text"
		existing := models.Exhibitor{Name: "English company", About: utils.StringPtr("Original"), AboutSv: &translated, Industries: []models.Industry{{Name: "Technology", NameSv: &translated}}, Employments: []models.Employment{{Name: "Internship", NameSv: &translated}}, Programs: []models.Program{{Name: "Physics", NameSv: &translated}}}
		if err := database.Create(&existing).Error; err != nil {
			t.Fatal(err)
		}
		incoming := models.Exhibitor{Name: "English company", About: utils.StringPtr("Changed source"), Industries: []models.Industry{{Name: "Technology"}}, Employments: []models.Employment{{Name: "Internship"}}, Programs: []models.Program{{Name: "Physics"}}}
		state := models.EventroSyncState{Source: exhibitorSyncValues(existing)}
		merged := mergeEventroExhibitor(existing, incoming, &state, false)
		request := httptest.NewRequest(http.MethodGet, "/eventroexhibitors", nil)
		if err := persistEventroExhibitorCategories(database, request, &merged, existing); err != nil {
			t.Fatal(err)
		}
		if merged.AboutSv == nil || *merged.AboutSv != translated || *merged.About != "Changed source" {
			t.Fatalf("exhibitor translation/source mismatch: %#v", merged)
		}
		for _, item := range []struct {
			id          uint
			original    uint
			translation *string
		}{
			{merged.Industries[0].ID, existing.Industries[0].ID, merged.Industries[0].NameSv},
			{merged.Employments[0].ID, existing.Employments[0].ID, merged.Employments[0].NameSv},
			{merged.Programs[0].ID, existing.Programs[0].ID, merged.Programs[0].NameSv},
		} {
			if item.id != item.original || item.translation == nil || *item.translation != translated {
				t.Fatalf("category identity/translation lost: %#v", item)
			}
		}
	})
}
