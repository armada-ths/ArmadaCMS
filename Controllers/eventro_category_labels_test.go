package controllers

import (
	"ArmadaCMS/main/models"
	"reflect"
	"testing"
)

func TestEventroCategoriesKeepSourceLabels(t *testing.T) {
	incoming := mapEventroToExhibitor(eventroExhibitorResponse{Catalogue: eventroCatalogue{
		Industries:  []string{"advisory", " Consulting ", "consulting", "consulting", "  "},
		Employments: []string{"Full Time", "full time"},
		Educations:  []string{"Graduate programme", "Computer Engineering"},
	}})
	values := exhibitorSyncValues(incoming)
	for field, expected := range map[string]string{
		"industries":  `[" Consulting ","advisory","consulting"]`,
		"employments": `["Full Time","full time"]`,
		"programs":    `["Computer Engineering","Graduate programme"]`,
	} {
		if !sameEventroValue(values[field], []byte(expected)) {
			t.Fatalf("%s: got %s, want %s", field, values[field], expected)
		}
	}

	current := incoming
	current.Industries = []models.Industry{{Name: "Consulting & Advisory"}}
	state := models.EventroSyncState{Source: exhibitorSyncValues(current)}
	merged := mergeEventroExhibitor(current, incoming, &state, false)
	if !reflect.DeepEqual(merged.Industries, incoming.Industries) || state.Overrides["industries"] {
		t.Fatal("source labels were not imported into an Eventro-owned field")
	}

	current.Industries = []models.Industry{{Name: "CMS correction"}}
	merged = mergeEventroExhibitor(current, incoming, &state, false)
	if !reflect.DeepEqual(merged.Industries, current.Industries) || !state.Overrides["industries"] {
		t.Fatal("source labels overwrote a manual CMS correction")
	}
}
