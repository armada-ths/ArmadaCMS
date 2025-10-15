package controllers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/shopspring/decimal"
	"gorm.io/gorm/clause"

	"ArmadaCMS/main/db"
	"ArmadaCMS/main/models"
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

func FetchEventsEventro(w http.ResponseWriter, r *http.Request) {
	client := &http.Client{Timeout: 30 * time.Second}
	fairID := os.Getenv("EVENTRO_FAIR_ID")
	url := fmt.Sprintf("https://app.eventro.se/api/v1/fairs/%s/events/", fairID)

	req, err := http.NewRequest(http.MethodGet, url, nil)
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

	for _, e := range result.Events {
		ev := mapEventroToEvent(e)

		result := db.DB.Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "eventro_id"}},
			DoUpdates: clause.AssignmentColumns([]string{
				"name", "description", "event_start", "event_end",
				"registration_end", "event_max_capacity",
			}),
		}).Create(&ev)

		if result.Error != nil {
			log.Printf("❌ Failed upserting event %s (%s): %v", e.ID, e.Name, result.Error)
			continue
		}

		if result.RowsAffected == 1 {
			inserted++
		} else {
			updated++
		}
	}

	log.Printf("✅ Event sync completed — inserted: %d, updated: %d", inserted, updated)
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(fmt.Sprintf("Sync completed — inserted: %d, updated: %d", inserted, updated)))
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

	var fee *decimal.Decimal = nil
	location := "TBA"

	return models.Event{
		EventroID:            e.ID,
		Name:                 e.Name,
		Description:          e.Description,
		Location:             location,
		Food:                 nil,
		EventStart:           derefOrNow(parseTime(e.EventStartsAt)),
		EventEnd:             derefOrNow(parseTime(e.EventEndsAt)),
		RegistrationEnd:      parseTime(e.ClosesForRegistrationAt),
		ImageURL:             nil,
		Fee:                  fee,
		RegistrationRequired: e.AllowWaitlist,
		SignupLink:           nil,
		EventMaxCapacity:     &e.MaxParticipants,
	}
}

// helper — ensures we always have a valid time.Time
func derefOrNow(t *time.Time) time.Time {
	if t == nil {
		return time.Now().UTC()
	}
	return *t
}
