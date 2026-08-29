package controllers

import "testing"

func TestExhibitorSortColumn(t *testing.T) {
	tests := map[string]string{
		"id":           "id",
		"name":         "name",
		"type":         "type",
		"tier":         "tier",
		"fairLocation": "fair_location",
		"industries":   "id",
		"programs":     "id",
		"unknown":      "id",
	}

	for field, want := range tests {
		t.Run(field, func(t *testing.T) {
			if got := exhibitorSortColumn(field); got != want {
				t.Fatalf("exhibitorSortColumn(%q) = %q, want %q", field, got, want)
			}
		})
	}
}
