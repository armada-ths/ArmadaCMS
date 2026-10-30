package controllers

import (
	"ArmadaCMS/main/audit"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"time"

	"gorm.io/gorm"

	"ArmadaCMS/main/db"
	"ArmadaCMS/main/models"
	"ArmadaCMS/main/utils"
)

// ---------- Eventro API Models ----------

type eventroEventResponse struct {
	CustomFields            map[string]json.RawMessage `json:"customFields"`
	ID                      string                     `json:"id"`
	Name                    string                     `json:"name"`
	URL                     *string                    `json:"url"`
	Location                *string                    `json:"location"`
	Description             *string                    `json:"description"`
	Participants            int                        `json:"participants"`
	AllowWaitlist           bool                       `json:"allowWaitlist"`
	MaxParticipants         int                        `json:"maxParticipants"`
	EventStartsAt           string                     `json:"eventStartsAt"`
	EventEndsAt             string                     `json:"eventEndsAt"`
	OpensForRegistrationAt  string                     `json:"opensForRegistrationAt"`
	ClosesForRegistrationAt string                     `json:"closesForRegistrationAt"`
	OpensForPublicationAt   string                     `json:"opensForPublicationAt"`
	ClosesForPublicationAt  string                     `json:"closesForPublicationAt"`
}

// ---------- Controller ----------

// FetchEventsEventro syncs events from the Eventro API into the local database and returns the current list.
// @Summary Sync & list events from Eventro
// @Tags eventro
// @Produce json
// @Param fairId query string true "Eventro fair instance ID"
// @Success 200 {array} models.Event
// @Failure 400 {string} string "fairId query parameter is required"
// @Failure 500 {string} string "Failed to fetch from Eventro"
// @Security BearerAuth
// @Router /eventroevents [get]
func FetchEventsEventro(w http.ResponseWriter, r *http.Request) {
	client := &http.Client{Timeout: 30 * time.Second}
	fairID, ok := requireEventroFairID(w, r)
	if !ok {
		return
	}
	endpoint := fmt.Sprintf("https://app.eventro.se/api/v1/fairs/%s/events/", url.PathEscape(fairID))

	items, err := fetchEventroPages[eventroEventResponse](r.Context(), client, endpoint, "events", os.Getenv("EVENTRO_API"), os.Getenv("EVENTRO_ORG"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}

	inserted := 0
	updated := 0
	unchanged := 0
	failed := 0
	group, err := startAuditGroup(r, "sync", "events", fairID, map[string]any{"source": "eventro", "status": "running"})
	if err != nil {
		http.Error(w, "failed to start audit log", http.StatusInternalServerError)
		return
	}
	childRequest := audit.WithParent(r, group.ID)

	for _, e := range items {
		if !validEventroTimestamp(e.EventStartsAt) || !validEventroTimestamp(e.EventEndsAt) {
			failed++
			continue
		}
		ev := mapEventroToEvent(e)
		created := false
		changed := false
		err := db.DB.Transaction(func(tx *gorm.DB) error {
			_, inserted, updated, err := syncEventroRecordWithAudit(tx, childRequest, "events", &ev, "eventro_id", e.ID, nil, eventroEventSyncFields(e))
			created, changed = inserted, inserted || updated
			return err
		})
		if err != nil {
			log.Printf("❌ Failed upserting event %s (%s): %v", e.ID, e.Name, err)
			failed++
			continue
		}
		if !changed {
			unchanged++
		} else if created {
			inserted++
		} else {
			updated++
		}
	}

	status := auditGroupStatus(inserted+updated+unchanged, failed)
	summary := map[string]any{"source": "eventro", "inserted": inserted, "updated": updated, "unchanged": unchanged, "failed": failed}
	if err := finalizeAuditGroup(group.ID, status, summary); err != nil {
		http.Error(w, "failed to finalize audit log", http.StatusInternalServerError)
		return
	}
	if inserted+updated > 0 {
		go utils.RevalidateTag("events")
	}
	log.Printf("✅ Event sync completed — inserted: %d, updated: %d", inserted, updated)
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "Sync completed — inserted: %d, updated: %d, unchanged: %d, failed: %d", inserted, updated, unchanged, failed)
}

func stringPointersEqual(left, right *string) bool {
	if left == nil || right == nil {
		return left == right
	}
	return *left == *right
}

// ---------- Mapper ----------

func mapEventroToEvent(e eventroEventResponse) models.Event {
	parseTime := func(s string) *time.Time {
		if s == "" {
			return nil
		}
		t, err := time.Parse(time.RFC3339, s)
		if err != nil {
			log.Printf("⚠️ Failed to parse time %q: %v", s, err)
			return nil
		}
		return &t
	}

	signupLink := e.URL
	NormalizeOptionalStringPointers(&signupLink)

	location, _ := eventroEventLocation(e)
	food, _ := eventroTextField(e.CustomFields, "food")
	openTo, _ := eventroTextField(e.CustomFields, "opento")
	mapURL, _ := eventroTextField(e.CustomFields, "mapurl")
	locationText := ""
	if location != nil {
		locationText = *location
	}
	return models.Event{
		EventroID:        e.ID,
		Name:             e.Name,
		Description:      utils.SanitizeEventDescription(e.Description),
		Location:         locationText,
		Food:             food,
		OpenTo:           openTo,
		MapURL:           mapURL,
		EventStart:       derefOrNow(parseTime(e.EventStartsAt)),
		EventEnd:         derefOrNow(parseTime(e.EventEndsAt)),
		RegistrationEnd:  parseTime(e.ClosesForRegistrationAt),
		ImageURL:         nil,
		SignupLink:       signupLink,
		EventMaxCapacity: &e.MaxParticipants,
	}
}

// helper — ensures we always have a valid time.Time
func derefOrNow(t *time.Time) time.Time {
	if t == nil {
		return time.Now().UTC()
	}
	return *t
}

// The API omits archived fields and returns null for active, unset fields.
// Ignore unexpected types without failing the entire event import.
func eventroTextField(fields map[string]json.RawMessage, key string) (*string, bool) {
	raw, present := fields[key]
	if !present {
		return nil, false
	}
	var value *string
	if err := json.Unmarshal(raw, &value); err != nil {
		return nil, false
	}
	NormalizeOptionalStringPointers(&value)
	return value, true
}

func eventroEventLocation(input eventroEventResponse) (*string, bool) {
	location := input.Location
	NormalizeOptionalStringPointers(&location)
	if location != nil {
		return location, true
	}
	location, present := eventroTextField(input.CustomFields, "location")
	return location, present || input.Location != nil
}

func eventroEventSyncFields(e eventroEventResponse) []string {
	fields := append([]string{}, eventroResourceSpecs["events"].fields...)
	omitted := map[string]bool{}
	if e.URL == nil {
		omitted["signupLink"] = true
	}
	if _, present := eventroEventLocation(e); !present {
		omitted["location"] = true
	}
	for field, key := range map[string]string{"food": "food", "openTo": "opento", "mapUrl": "mapurl"} {
		if _, present := eventroTextField(e.CustomFields, key); !present {
			omitted[field] = true
		}
	}
	selected := []string{}
	for _, field := range fields {
		if !omitted[field] {
			selected = append(selected, field)
		}
	}
	return selected
}
