package audit

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestBuildEntryIncludesParentID(t *testing.T) {
	request := httptest.NewRequest(http.MethodPost, "/api/v1/eventroevents", nil)
	request = WithParent(request, 42)

	entry := buildEntry(nil, request, "update", "events", 7, map[string]any{"name": "Before"}, map[string]any{"name": "After"})

	if entry.ParentID == nil || *entry.ParentID != 42 {
		t.Fatalf("expected parent ID 42, got %v", entry.ParentID)
	}
	if entry.Action != "update" || entry.ResourceType != "events" || entry.ResourceID != "7" {
		t.Fatalf("unexpected audit identity: %#v", entry)
	}
}

func TestBuildEntryLeavesStandaloneLogsUngrouped(t *testing.T) {
	request := httptest.NewRequest(http.MethodPost, "/api/v1/events", nil)

	entry := buildEntry(nil, request, "create", "events", 7, nil, map[string]any{"name": "Event"})

	if entry.ParentID != nil {
		t.Fatalf("expected no parent ID, got %d", *entry.ParentID)
	}
}
