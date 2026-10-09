package controllers

import (
	"ArmadaCMS/main/models"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestTimelineEntryDoesNotRequireYear(t *testing.T) {
	entry := models.TimelineEntry{Title: "Event", Body: "Description", EraID: 1, SortOrder: 0}
	if !normalizeTimelineEntry(&entry) {
		t.Fatal("entry without a separate year was rejected")
	}
}

func TestTimelineEraNormalizesTitle(t *testing.T) {
	era := models.TimelineEra{Title: "  The 1980s  ", SortOrder: 0}
	if !normalizeTimelineEra(&era) || era.Title != "The 1980s" {
		t.Fatal("era title was rejected or not normalized")
	}
}

func TestTimelineListRejectsUnsupportedSort(t *testing.T) {
	for _, test := range []struct {
		name    string
		handler http.HandlerFunc
		path    string
	}{
		{"entries", GetTimelineEntries, "/timeline-entries"},
		{"eras", GetTimelineEras, "/timeline-eras"},
	} {
		t.Run(test.name, func(t *testing.T) {
			query := url.Values{"sort": {`["title","ASC"]`}}
			request := httptest.NewRequest(http.MethodGet, test.path+"?"+query.Encode(), nil)
			response := httptest.NewRecorder()
			test.handler(response, request)
			if response.Code != http.StatusBadRequest {
				t.Fatalf("unsupported sort returned %d, want %d", response.Code, http.StatusBadRequest)
			}
		})
	}
}
