package controllers

import (
	"ArmadaCMS/main/models"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestFetchEventroFairByIDIncludesTimeline(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("organization"); got != "organization-id" {
			t.Errorf("organization header = %q, want %q", got, "organization-id")
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"fairInstances": [
				{
					"id": "armada",
					"name": "THS Armada 2026",
					"startDate": "2026-03-29T00:00:00Z",
					"endDate": "2027-01-01T00:00:00Z",
					"timelineEntries": [
						{"id":"entry-1","title":"Fair day 1","date":"2026-11-17T00:00:00Z"}
					]
				}
			],
			"count": "1"
		}`))
	}))
	defer server.Close()

	got, err := fetchEventroFairByID(
		context.Background(),
		server.Client(),
		server.URL,
		"api-key",
		"organization-id",
		"armada",
	)
	if err != nil {
		t.Fatalf("fetchEventroFairByID() error = %v", err)
	}
	if len(got.TimelineEntries) != 1 || got.TimelineEntries[0].Title != "Fair day 1" {
		t.Errorf("fetchEventroFairByID() timeline = %#v", got.TimelineEntries)
	}
}

func TestFetchEventroFairByIDRejectsMissingSelectedFair(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"fairInstances":[],"count":0}`))
	}))
	defer server.Close()

	_, err := fetchEventroFairByID(
		context.Background(),
		server.Client(),
		server.URL,
		"api-key",
		"organization-id",
		"armada",
	)
	if err == nil {
		t.Fatal("fetchEventroFairByID() error = nil, want missing fair error")
	}
}

func TestRequireEventroFairID(t *testing.T) {
	t.Parallel()

	request := httptest.NewRequest(http.MethodGet, "/sync?fairId=armada", nil)
	recorder := httptest.NewRecorder()

	got, ok := requireEventroFairID(recorder, request)
	if !ok {
		t.Fatal("requireEventroFairID() ok = false, want true")
	}
	if got != "armada" {
		t.Errorf("requireEventroFairID() = %q, want %q", got, "armada")
	}
}

func TestRequireEventroFairIDRejectsMissingID(t *testing.T) {
	t.Parallel()

	request := httptest.NewRequest(http.MethodGet, "/sync", nil)
	recorder := httptest.NewRecorder()

	if _, ok := requireEventroFairID(recorder, request); ok {
		t.Fatal("requireEventroFairID() ok = true, want false")
	}
	if recorder.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", recorder.Code, http.StatusBadRequest)
	}
}

func TestMapEventroFairToFairDate(t *testing.T) {
	t.Parallel()

	fair := eventroFairResponse{
		ID:   "armada-2026",
		Name: "Armada 2026",
		TimelineEntries: []eventroTimelineEntry{
			{Title: "Fair day 2", Date: "2026-11-18T00:00:00+01:00"},
			{Title: "Priority registration opens", Date: "2026-03-30T00:00:00Z"},
			{Title: "Priority registration closes", Date: "2026-05-22T23:59:59Z"},
			{Title: "Acceptance date", Date: "2026-06-22T00:00:00Z"},
			{Title: "Standard Registration is open", Date: "2026-08-17T00:00:00Z"},
			{Title: "Standard Registration closes", Date: "2026-10-02T23:59:59Z"},
			{Title: "Event weeks start", Date: "2026-10-05T00:00:00Z"},
			{Title: "Fair day 1", Date: "2026-11-17T00:00:00+01:00"},
		},
	}

	got, err := mapEventroFairToFairDate(fair)
	if err != nil {
		t.Fatalf("mapEventroFairToFairDate() error = %v", err)
	}

	if got.EventroID == nil || *got.EventroID != "armada-2026" {
		t.Errorf("EventroID = %v, want armada-2026", got.EventroID)
	}
	if got.Description != "Armada 2026" {
		t.Errorf("Description = %q", got.Description)
	}
	if got.FairDays != "2026-11-17,2026-11-18" {
		t.Errorf("FairDays = %q", got.FairDays)
	}
	if got.IRStart != "2026-03-30" || got.IREnd != "2026-05-22" || got.IRAcceptance != "2026-06-22" {
		t.Errorf("IR dates = %q, %q, %q", got.IRStart, got.IREnd, got.IRAcceptance)
	}
	if got.FRStart != "2026-08-17" || got.FREnd != "2026-10-02" {
		t.Errorf("FR dates = %q, %q", got.FRStart, got.FREnd)
	}
	if got.EventsStart != "2026-10-05" {
		t.Errorf("EventsStart = %q", got.EventsStart)
	}
}

func TestMapEventroFairToFairDateSupportsAliases(t *testing.T) {
	t.Parallel()

	fair := eventroFairResponse{
		ID:   "armada",
		Name: "Armada",
		TimelineEntries: []eventroTimelineEntry{
			{Title: "IR starts", Date: "2026-03-30T00:00:00Z"},
			{Title: "IR deadline", Date: "2026-05-22T00:00:00Z"},
			{Title: "Acceptance notification", Date: "2026-06-22T00:00:00Z"},
			{Title: "Final registration begins", Date: "2026-08-17T00:00:00Z"},
			{Title: "FR ends", Date: "2026-10-02T00:00:00Z"},
			{Title: "Pre-fair events begin", Date: "2026-10-05T00:00:00Z"},
			{Title: "Career fair day 1", Date: "2026-11-17T00:00:00Z"},
		},
	}

	got, err := mapEventroFairToFairDate(fair)
	if err != nil {
		t.Fatalf("mapEventroFairToFairDate() error = %v", err)
	}
	if got.IRStart == "" || got.IREnd == "" || got.FRStart == "" || got.FREnd == "" || got.EventsStart == "" {
		t.Errorf("aliases were not mapped: %#v", got)
	}
}

func TestMapEventroFairToFairDateSupportsDirectEventStartTitle(t *testing.T) {
	t.Parallel()

	fair := eventroFairResponse{
		ID:   "armada",
		Name: "Armada",
		TimelineEntries: []eventroTimelineEntry{
			{Title: "Priority registration opens", Date: "2026-03-30T00:00:00Z"},
			{Title: "Priority registration closes", Date: "2026-05-22T00:00:00Z"},
			{Title: "Acceptance date", Date: "2026-06-22T00:00:00Z"},
			{Title: "Standard registration opens", Date: "2026-08-17T00:00:00Z"},
			{Title: "Standard registration closes", Date: "2026-10-02T00:00:00Z"},
			{Title: "Events start", Date: "2026-10-05T00:00:00Z"},
			{Title: "Fair day 1", Date: "2026-11-17T00:00:00Z"},
		},
	}

	got, err := mapEventroFairToFairDate(fair)
	if err != nil {
		t.Fatalf("mapEventroFairToFairDate() error = %v", err)
	}
	if got.EventsStart != "2026-10-05" {
		t.Errorf("EventsStart = %q", got.EventsStart)
	}
}

func TestMapEventroFairToFairDateRejectsMissingRequiredMilestones(t *testing.T) {
	t.Parallel()

	_, err := mapEventroFairToFairDate(eventroFairResponse{
		ID:   "armada",
		Name: "Armada",
		TimelineEntries: []eventroTimelineEntry{
			{Title: "Fair day 1", Date: "2026-11-17T00:00:00Z"},
		},
	})
	if err == nil {
		t.Fatal("mapEventroFairToFairDate() error = nil, want missing milestones error")
	}
	if !strings.Contains(err.Error(), "priority registration opens") {
		t.Errorf("error = %q, want missing milestone details", err)
	}
}

func TestMapEventroFairToFairDateRejectsInvalidRecognizedDate(t *testing.T) {
	t.Parallel()

	_, err := mapEventroFairToFairDate(eventroFairResponse{
		ID: "armada",
		TimelineEntries: []eventroTimelineEntry{
			{Title: "Priority registration opens", Date: "not-a-date"},
		},
	})
	if err == nil {
		t.Fatal("mapEventroFairToFairDate() error = nil, want invalid date error")
	}
}

func TestFairDateConfigsEqual(t *testing.T) {
	t.Parallel()

	eventroID := "armada-2026"
	base := models.FairDateConfig{
		EventroID:    &eventroID,
		Description:  "Armada 2026",
		FairDays:     "2026-11-17,2026-11-18",
		IRStart:      "2026-03-30",
		IREnd:        "2026-05-22",
		IRAcceptance: "2026-06-22",
		FRStart:      "2026-08-17",
		FREnd:        "2026-10-02",
		EventsStart:  "2026-10-05",
	}

	tests := []struct {
		name  string
		left  models.FairDateConfig
		right models.FairDateConfig
		want  bool
	}{
		{
			name:  "equal",
			left:  base,
			right: base,
			want:  true,
		},
		{
			name: "different eventro id",
			left: base,
			right: func() models.FairDateConfig {
				other := base
				value := "armada-2027"
				other.EventroID = &value
				return other
			}(),
			want: false,
		},
		{
			name: "different description",
			left: base,
			right: func() models.FairDateConfig {
				other := base
				other.Description = "Armada 2027"
				return other
			}(),
			want: false,
		},
		{
			name: "different fair days",
			left: base,
			right: func() models.FairDateConfig {
				other := base
				other.FairDays = "2026-11-17"
				return other
			}(),
			want: false,
		},
		{
			name: "different ir start",
			left: base,
			right: func() models.FairDateConfig {
				other := base
				other.IRStart = "2026-03-31"
				return other
			}(),
			want: false,
		},
		{
			name: "different ir end",
			left: base,
			right: func() models.FairDateConfig {
				other := base
				other.IREnd = "2026-05-23"
				return other
			}(),
			want: false,
		},
		{
			name: "different ir acceptance",
			left: base,
			right: func() models.FairDateConfig {
				other := base
				other.IRAcceptance = "2026-06-23"
				return other
			}(),
			want: false,
		},
		{
			name: "different fr start",
			left: base,
			right: func() models.FairDateConfig {
				other := base
				other.FRStart = "2026-08-18"
				return other
			}(),
			want: false,
		},
		{
			name: "different fr end",
			left: base,
			right: func() models.FairDateConfig {
				other := base
				other.FREnd = "2026-10-03"
				return other
			}(),
			want: false,
		},
		{
			name: "different events start",
			left: base,
			right: func() models.FairDateConfig {
				other := base
				other.EventsStart = "2026-10-06"
				return other
			}(),
			want: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := fairDateConfigsEqual(test.left, test.right); got != test.want {
				t.Fatalf("fairDateConfigsEqual() = %v, want %v", got, test.want)
			}
		})
	}
}
