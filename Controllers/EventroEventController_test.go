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
			preserveOmittedEventroFields(&got, existing, input.CustomFields)
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
