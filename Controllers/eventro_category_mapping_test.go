package controllers

import (
	"ArmadaCMS/main/models"
	"reflect"
	"testing"
)

func TestEventroCategoryMappingBeforeMerge(t *testing.T) {
	source := eventroExhibitorResponse{Catalogue: eventroCatalogue{
		Industries:  []string{"advisory", "consulting"},
		Employments: []string{"Full Time", "full time", "Masters Thesis"},
		Educations:  []string{"Computer Engineering", "Computer Science", "electricity"},
	}}
	incoming := mapEventroToExhibitor(source)
	values := exhibitorSyncValues(incoming)
	for field, expected := range map[string]string{
		"industries":  `["Consulting & Advisory"]`,
		"employments": `["Full-time employment","Master's thesis"]`,
		"programs":    `["Computer Science","Electrical Engineering"]`,
	} {
		if !sameEventroValue(values[field], []byte(expected)) {
			t.Fatalf("%s: got %s, want %s", field, values[field], expected)
		}
	}
	legacy := incoming
	legacy.Industries = []models.Industry{{Name: "advisory"}, {Name: "consulting"}}
	legacyState := models.EventroSyncState{Source: exhibitorSyncValues(legacy)}
	normalized := mergeEventroExhibitor(legacy, incoming, &legacyState, false)
	if !reflect.DeepEqual(normalized.Industries, incoming.Industries) || legacyState.Overrides["industries"] {
		t.Fatal("normalization was not applied to an unchanged Eventro-owned field")
	}
	before := incoming
	state := models.EventroSyncState{Source: exhibitorSyncValues(incoming)}
	current := before
	current.Industries = []models.Industry{{Name: "CMS correction"}}
	merged := mergeEventroExhibitor(current, incoming, &state, false)
	if !reflect.DeepEqual(merged.Industries, current.Industries) || !state.Overrides["industries"] {
		t.Fatal("normalization overwrote a manual CMS correction")
	}
}
