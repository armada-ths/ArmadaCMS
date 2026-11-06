package controllers

import (
	"ArmadaCMS/main/models"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"
)

func TestEventroFieldOwnershipAcrossAllResources(t *testing.T) {
	for resource, spec := range eventroResourceSpecs {
		t.Run(resource, func(t *testing.T) {
			field := spec.fields[0]
			values := func(value string) map[string]json.RawMessage {
				raw, _ := json.Marshal(value)
				return map[string]json.RawMessage{field: raw}
			}
			state := models.EventroSyncState{}
			first := mergeEventroValues(nil, values("original"), &state, true)
			if !sameEventroValue(first[field], values("original")[field]) {
				t.Fatal("new value not imported")
			}
			updated := mergeEventroValues(values("original"), values("updated"), &state, false)
			if !sameEventroValue(updated[field], values("updated")[field]) {
				t.Fatal("Eventro correction not imported")
			}
			manual := mergeEventroValues(values("CMS correction"), values("updated"), &state, false)
			if _, overwritten := manual[field]; overwritten || !state.Overrides[field] {
				t.Fatal("CMS correction overwritten")
			}
			mergeEventroValues(values("CMS correction"), values("another update"), &state, false)
			if !state.Conflicts[field] {
				t.Fatal("simultaneous changes not flagged")
			}
			delete(state.Conflicts, field) // Keep CMS acknowledges only this source version.
			mergeEventroValues(values("CMS correction"), values("another update"), &state, false)
			if state.Conflicts[field] {
				t.Fatal("acknowledged conflict reappeared")
			}
			mergeEventroValues(values("CMS correction"), values("third update"), &state, false)
			if !state.Conflicts[field] {
				t.Fatal("new source change not flagged")
			}
			delete(state.Overrides, field) // Use Eventro resumes normal updates.
			delete(state.Conflicts, field)
			resumed := mergeEventroValues(values("third update"), values("fourth update"), &state, false)
			if _, ok := resumed[field]; !ok {
				t.Fatal("Eventro synchronization did not resume")
			}
		})
	}
}

func TestEventroTimeZoneRoundTrip(t *testing.T) {
	sourceTime, _ := time.Parse(time.RFC3339, "2026-10-29T12:00:00+02:00")
	storedTime := sourceTime.UTC()
	source := eventroStructValues(&models.Event{EventStart: sourceTime}, []string{"eventStart"})
	stored := eventroStructValues(&models.Event{EventStart: storedTime}, []string{"eventStart"})
	state := models.EventroSyncState{Source: source}
	mergeEventroValues(stored, source, &state, false)
	if state.Overrides["eventStart"] {
		t.Fatal("database time zone normalization mistaken for a CMS edit")
	}
}

func TestEventroArchivedFieldsCannotApplyStaleSource(t *testing.T) {
	state := models.EventroSyncState{Source: map[string]json.RawMessage{"food": json.RawMessage(`"Old meal"`)}, Overrides: map[string]bool{"food": true}, Conflicts: map[string]bool{"food": true}}
	mergeEventroValues(map[string]json.RawMessage{"food": json.RawMessage(`"CMS meal"`)}, map[string]json.RawMessage{}, &state, false)
	if _, exists := state.Source["food"]; exists || state.Conflicts["food"] || !state.Overrides["food"] {
		t.Fatal("archived field still offered a stale source or lost CMS ownership")
	}
}

func TestEventroLegacyCorrectionsAndExplicitClearing(t *testing.T) {
	state := models.EventroSyncState{}
	current := map[string]json.RawMessage{"name": json.RawMessage(`"CMS"`), "about": json.RawMessage(`null`)}
	source := map[string]json.RawMessage{"name": json.RawMessage(`"Eventro"`), "about": json.RawMessage(`"New content"`)}
	updates := mergeEventroValues(current, source, &state, false)
	if _, ok := updates["name"]; ok || !state.Conflicts["name"] {
		t.Fatal("legacy correction lost")
	}
	if _, ok := updates["about"]; !ok {
		t.Fatal("missing legacy content not filled")
	}
	updates = mergeEventroValues(map[string]json.RawMessage{"about": source["about"]}, map[string]json.RawMessage{"about": json.RawMessage(`null`)}, &state, false)
	if string(updates["about"]) != "null" {
		t.Fatal("Eventro clear not propagated")
	}
	state = models.EventroSyncState{Source: map[string]json.RawMessage{"name": json.RawMessage(`"old"`)}}
	updates = mergeEventroValues(map[string]json.RawMessage{"name": json.RawMessage(`""`)}, map[string]json.RawMessage{"name": json.RawMessage(`"new"`)}, &state, false)
	if _, ok := updates["name"]; ok {
		t.Fatal("manual clearing overwritten")
	}
}

func TestEventroThumbnailAndCategoryCorrections(t *testing.T) {
	original, thumb, manual := "original", "thumbnail", "manual"
	current := models.Exhibitor{LogoFreesizeUrl: &original, Industries: []models.Industry{{ID: 5, Name: "Technology"}}}
	incoming := models.Exhibitor{LogoFreesizeUrl: &original, LogoThumbnail: &thumb, Industries: []models.Industry{{Name: "Technology"}}}
	state := models.EventroSyncState{}
	got := mergeEventroExhibitor(current, incoming, &state, false)
	if got.LogoThumbnail == nil || *got.LogoThumbnail != thumb || state.Overrides["logo"] || state.Overrides["industries"] {
		t.Fatal("safe thumbnail addition or name-based category comparison failed")
	}
	current = got
	current.LogoFreesizeUrl, current.LogoThumbnail = &manual, nil
	got = mergeEventroExhibitor(current, incoming, &state, false)
	if got.LogoThumbnail != nil || *got.LogoFreesizeUrl != manual {
		t.Fatal("manual logo mixed with Eventro thumbnail")
	}
	if !reflect.DeepEqual(normalizedEventroLabels([]string{" Stockholm ", "Stockholm", "", "Göteborg"}), []string{"Göteborg", "Stockholm"}) {
		t.Fatal("locations not normalized")
	}
}

func TestEventroPaginationRejectsIncompleteExports(t *testing.T) {
	type item struct {
		ID string `json:"id"`
	}
	for _, scenario := range []struct {
		name  string
		pages []string
		want  int
		fail  bool
	}{
		{"complete", []string{`{"count":2,"items":[{"id":"a"}]}`, `{"count":"2","items":[{"id":"b"}]}`}, 2, false},
		{"empty", []string{`{"count":0,"items":[]}`}, 0, false},
		{"truncated", []string{`{"count":2,"items":[{"id":"a"}]}`, `{"count":2,"items":[]}`}, 0, true},
		{"duplicate", []string{`{"count":2,"items":[{"id":"a"},{"id":"a"}]}`}, 0, true},
		{"changing count", []string{`{"count":2,"items":[{"id":"a"}]}`, `{"count":3,"items":[{"id":"b"}]}`}, 0, true},
		{"missing list", []string{`{"count":0}`}, 0, true},
		{"null list", []string{`{"count":0,"items":null}`}, 0, true},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			page := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if page >= len(scenario.pages) {
					t.Error("unexpected extra page")
					http.Error(w, "extra", 500)
					return
				}
				fmt.Fprint(w, scenario.pages[page])
				page++
			}))
			defer server.Close()
			got, err := fetchEventroPages[item](context.Background(), server.Client(), server.URL, "items", "", "")
			if (err != nil) != scenario.fail || len(got) != scenario.want {
				t.Fatalf("got %v, %v", got, err)
			}
		})
	}
}
