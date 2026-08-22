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
	"gorm.io/gorm/clause"

	"ArmadaCMS/main/db"
	"ArmadaCMS/main/models"
	"ArmadaCMS/main/utils"
)

// ---------- Eventro API Models ----------

type eventroEventsResponse struct {
	Events []eventroEventResponse `json:"events"`
	Count  int                    `json:"count"`
}

type eventroEventResponse struct {
	ID                      string  `json:"id"`
	Name                    string  `json:"name"`
	Description             *string `json:"description"`
	Participants            int     `json:"participants"`
	AllowWaitlist           bool    `json:"allowWaitlist"`
	MaxParticipants         int     `json:"maxParticipants"`
	EventStartsAt           string  `json:"eventStartsAt"`
	EventEndsAt             string  `json:"eventEndsAt"`
	OpensForRegistrationAt  string  `json:"opensForRegistrationAt"`
	ClosesForRegistrationAt string  `json:"closesForRegistrationAt"`
	OpensForPublicationAt   string  `json:"opensForPublicationAt"`
	ClosesForPublicationAt  string  `json:"closesForPublicationAt"`
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

	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		http.Error(w, "failed to create request", http.StatusInternalServerError)
		return
	}
	req.Header.Set("Authorization", "Bearer "+os.Getenv("EVENTRO_API"))
	req.Header.Set("organization", os.Getenv("EVENTRO_ORG"))

	resp, err := client.Do(req)
	if err != nil {
		http.Error(w, "failed to contact Eventro API", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		http.Error(w, "unexpected Eventro status: "+resp.Status, http.StatusBadGateway)
		return
	}

	var result eventroEventsResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		log.Printf("❌ Failed to decode Eventro response: %v", err)
		http.Error(w, "failed to decode Eventro response", http.StatusInternalServerError)
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

	for _, e := range result.Events {
		ev := mapEventroToEvent(e)
		created := false
		changed := false
		err := db.DB.Transaction(func(tx *gorm.DB) error {
			var existing models.Event
			findErr := tx.Where("eventro_id = ?", ev.EventroID).First(&existing).Error
			if findErr != nil && findErr != gorm.ErrRecordNotFound {
				return findErr
			}
			if findErr == nil &&
				existing.Name == ev.Name &&
				stringPointersEqual(existing.Description, ev.Description) &&
				existing.EventStart.Equal(ev.EventStart) &&
				existing.EventEnd.Equal(ev.EventEnd) &&
				timesEqual(existing.RegistrationEnd, ev.RegistrationEnd) &&
				intPointersEqual(existing.EventMaxCapacity, ev.EventMaxCapacity) {
				return nil
			}

			result := tx.Clauses(clause.OnConflict{
				Columns: []clause.Column{{Name: "eventro_id"}},
				DoUpdates: clause.AssignmentColumns([]string{
					"name", "description", "event_start", "event_end",
					"registration_end", "event_max_capacity",
				}),
			}).Create(&ev)
			if result.Error != nil {
				return result.Error
			}
			if err := tx.Where("eventro_id = ?", ev.EventroID).First(&ev).Error; err != nil {
				return err
			}
			changed = true
			created = findErr == gorm.ErrRecordNotFound
			if created {
				return audit.LogCreate(tx, childRequest, "events", ev.ID, ev)
			}
			return audit.LogUpdate(tx, childRequest, "events", ev.ID, existing, ev)
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
	fmt.Fprintf(w, "Sync completed — inserted: %d, updated: %d", inserted, updated)
}

func timesEqual(left, right *time.Time) bool {
	if left == nil || right == nil {
		return left == right
	}
	return left.Equal(*right)
}

func intPointersEqual(left, right *int) bool {
	if left == nil || right == nil {
		return left == right
	}
	return *left == *right
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

	return models.Event{
		EventroID:        e.ID,
		Name:             e.Name,
		Description:      e.Description,
		Food:             nil,
		EventStart:       derefOrNow(parseTime(e.EventStartsAt)),
		EventEnd:         derefOrNow(parseTime(e.EventEndsAt)),
		RegistrationEnd:  parseTime(e.ClosesForRegistrationAt),
		ImageURL:         nil,
		SignupLink:       nil,
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
