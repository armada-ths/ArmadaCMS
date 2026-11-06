package controllers

import (
	"ArmadaCMS/main/models"
	"encoding/json"
	"testing"
)

func TestEventroEventCustomFields(t *testing.T) {
	var input eventroEventResponse
	err := json.Unmarshal([]byte(`{"id":"event-1","name":"Lunch","customFields":{"location":" Nymble ","food":"Lunch included","opento":"All students","mapurl":"https://maps.google.com/?q=59.3,18.1","other":true}}`), &input)
	if err != nil {
		t.Fatal(err)
	}
	got := mapEventroToEvent(input)
	if got.Location != "Nymble" || got.Food == nil || *got.Food != "Lunch included" || got.OpenTo == nil || *got.OpenTo != "All students" || got.MapURL == nil || *got.MapURL != "https://maps.google.com/?q=59.3,18.1" {
		t.Fatalf("custom field mapping = %#v", got)
	}
}

func TestEventroOptionalFields(t *testing.T) {
	old := "Existing value"
	existing := models.Event{Location: old, Food: &old, OpenTo: &old, MapURL: &old}
	for _, test := range []struct {
		name, payload string
		cleared       bool
	}{
		{"omitted", `{}`, false},
		{"null object", `{"customFields":null}`, false},
		{"archived", `{"customFields":{}}`, false},
		{"unset", `{"customFields":{"location":null,"food":null,"opento":null,"mapurl":null}}`, true},
		{"blank", `{"customFields":{"location":"  ","food":"","opento":"  ","mapurl":""}}`, true},
		{"wrong types", `{"customFields":{"location":42,"food":true,"opento":["Students"],"mapurl":{}}}`, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			var input eventroEventResponse
			if err := json.Unmarshal([]byte(test.payload), &input); err != nil {
				t.Fatal(err)
			}
			got := mapEventroToEvent(input)
			if got.Location != "" || got.Food != nil || got.OpenTo != nil || got.MapURL != nil {
				t.Fatalf("new optional fields = %#v", got)
			}
			mergeEventroEventForTest(t, &got, existing, input)
			if test.cleared {
				if got.Location != "" || got.Food != nil || got.OpenTo != nil || got.MapURL != nil {
					t.Fatalf("unset fields were not cleared: %#v", got)
				}
			} else if got.Location != old || !stringPointersEqual(got.Food, &old) || !stringPointersEqual(got.OpenTo, &old) || !stringPointersEqual(got.MapURL, &old) {
				t.Fatalf("omitted fields were not preserved: %#v", got)
			}
		})
	}
}

func TestEventroEventSignupLink(t *testing.T) {
	for _, test := range []struct {
		name, payload string
		want          *string
	}{
		{"public event URL", `{"url":"https://app.eventro.se/event/123e4567-e89b-12d3-a456-426614174000"}`, stringPtrForEventroTest("https://app.eventro.se/event/123e4567-e89b-12d3-a456-426614174000")},
		{"trimmed URL", `{"url":" https://app.eventro.se/event/event-1 "}`, stringPtrForEventroTest("https://app.eventro.se/event/event-1")},
		{"empty URL", `{"url":""}`, nil},
		{"blank URL", `{"url":"  "}`, nil},
		{"null URL", `{"url":null}`, nil},
		{"omitted URL", `{}`, nil},
	} {
		t.Run(test.name, func(t *testing.T) {
			var input eventroEventResponse
			if err := json.Unmarshal([]byte(test.payload), &input); err != nil {
				t.Fatal(err)
			}
			got := mapEventroToEvent(input)
			if !stringPointersEqual(got.SignupLink, test.want) {
				t.Fatalf("signup link = %v, want %v", got.SignupLink, test.want)
			}
			if got.RegistrationRequired != (test.want != nil) {
				t.Fatalf("registration required = %v, want %v", got.RegistrationRequired, test.want != nil)
			}
			existing := models.Event{SignupLink: stringPtrForEventroTest("https://example.com/old-signup")}
			mergeEventroEventForTest(t, &got, existing, input)
			want := test.want
			if input.URL == nil {
				want = existing.SignupLink
			}
			if got.RegistrationRequired != (test.want != nil) {
				t.Fatalf("registration required after sync = %v, want %v", got.RegistrationRequired, test.want != nil)
			}
			if !stringPointersEqual(got.SignupLink, want) {
				t.Fatalf("signup link after preserving omitted fields = %v, want %v", got.SignupLink, want)
			}
		})
	}
}

func stringPtrForEventroTest(value string) *string {
	return &value
}

func TestEventroEventLocation(t *testing.T) {
	for _, test := range []struct {
		name, payload, want, wantExisting string
	}{
		{"top level wins", `{"location":" Main hall ","customFields":{"location":"Other hall"}}`, "Main hall", "Main hall"},
		{"top level only", `{"location":" Main hall "}`, "Main hall", "Main hall"},
		{"null top level falls back", `{"location":null,"customFields":{"location":" Other hall "}}`, "Other hall", "Other hall"},
		{"blank top level falls back", `{"location":"  ","customFields":{"location":" Other hall "}}`, "Other hall", "Other hall"},
		{"omitted top level falls back", `{"customFields":{"location":"Other hall"}}`, "Other hall", "Other hall"},
		{"both omitted", `{}`, "", "Existing hall"},
		{"null top level only", `{"location":null}`, "", "Existing hall"},
		{"blank top level only", `{"location":"  "}`, "", ""},
		{"explicitly cleared fallback", `{"location":null,"customFields":{"location":null}}`, "", ""},
	} {
		t.Run(test.name, func(t *testing.T) {
			var input eventroEventResponse
			if err := json.Unmarshal([]byte(test.payload), &input); err != nil {
				t.Fatal(err)
			}
			got := mapEventroToEvent(input)
			if got.Location != test.want {
				t.Fatalf("location = %q, want %q", got.Location, test.want)
			}
			mergeEventroEventForTest(t, &got, models.Event{Location: "Existing hall"}, input)
			if got.Location != test.wantExisting {
				t.Fatalf("location after preserving omitted fields = %q, want %q", got.Location, test.wantExisting)
			}
		})
	}
}

// Exercise the same field selection and three-way merge as the import path.
func mergeEventroEventForTest(t *testing.T, incoming *models.Event, existing models.Event, input eventroEventResponse) {
	t.Helper()
	current := eventroStructValues(&existing, eventroResourceSpecs["events"].fields)
	state := models.EventroSyncState{Source: current}
	updates := mergeEventroValues(current, eventroStructValues(incoming, eventroEventSyncFields(input)), &state, false)
	merged := existing
	for field, value := range updates {
		if err := applyEventroStructValue(&merged, field, value); err != nil {
			t.Fatal(err)
		}
	}
	*incoming = merged
}

func TestEventroEventRegistrationRequirementPreservedWithoutLink(t *testing.T) {
	for _, payload := range []string{`{}`, `{"url":null}`, `{"url":""}`, `{"url":"  "}`} {
		t.Run(payload, func(t *testing.T) {
			var input eventroEventResponse
			if err := json.Unmarshal([]byte(payload), &input); err != nil {
				t.Fatal(err)
			}
			got := mapEventroToEvent(input)
			mergeEventroEventForTest(t, &got, models.Event{RegistrationRequired: true}, input)
			if !got.RegistrationRequired {
				t.Fatal("missing signup link cleared registration requirement")
			}
		})
	}
}
