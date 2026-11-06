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

func TestExhibitorTierChoices(t *testing.T) {
	for _, value := range []string{"Standard", "Bronze", "Silver", "Gold"} {
		tier, err := parseExhibitorTier(value)
		if err != nil || tier == nil || string(*tier) != value {
			t.Fatalf("tier %q: %v, %v", value, tier, err)
		}
	}
	if tier, err := parseExhibitorTier(""); err != nil || tier != nil {
		t.Fatal("empty selection must clear tier")
	}
	for _, value := range []string{"Ignore", "gold", "Premium"} {
		if _, err := parseExhibitorTier(value); err == nil {
			t.Fatalf("unsupported tier accepted: %q", value)
		}
	}
}
