package controllers

import (
	"testing"
)

func TestParseOptionalTeamIDAcceptsInt32Bounds(t *testing.T) {
	for _, value := range []string{"-2147483648", "2147483647"} {
		teamID, err := parseOptionalTeamID(value)
		if err != nil {
			t.Fatalf("parse %q: %v", value, err)
		}
		if teamID == nil {
			t.Fatalf("parse %q returned nil", value)
		}
	}
}

func TestParseOptionalTeamIDRejectsValuesOutsideInt32(t *testing.T) {
	for _, value := range []string{"2147483648", "-2147483649"} {
		if teamID, err := parseOptionalTeamID(value); err == nil || teamID != nil {
			t.Fatalf("parse %q = %v, %v; expected range error", value, teamID, err)
		}
	}
}

func TestParseOptionalTeamIDPreservesValidValue(t *testing.T) {
	teamID, err := parseOptionalTeamID(" 42 ")
	if err != nil {
		t.Fatalf("parse team ID: %v", err)
	}
	if teamID == nil || *teamID != 42 {
		t.Fatalf("unexpected team ID: %v", teamID)
	}
}
