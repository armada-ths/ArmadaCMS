package controllers

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"
	"unicode"

	"ArmadaCMS/main/models"

	"gorm.io/gorm"
)

type fairDateMilestone string

const (
	milestoneFairDay      fairDateMilestone = "fair day"
	milestoneIRStart      fairDateMilestone = "IR start"
	milestoneIREnd        fairDateMilestone = "IR end"
	milestoneIRAcceptance fairDateMilestone = "IR acceptance"
	milestoneFRStart      fairDateMilestone = "FR start"
	milestoneFREnd        fairDateMilestone = "FR end"
	milestoneEventsStart  fairDateMilestone = "events start"
)

// FetchFairDatesEventro replaces all fair date configurations with the selected Eventro fair timeline.
// @Summary Replace fair dates from Eventro
// @Tags eventro
// @Produce plain
// @Param fairId query string true "Eventro fair instance ID"
// @Success 200 {string} string "Sync completed"
// @Failure 400 {string} string "fairId query parameter is required"
// @Failure 422 {string} string "Eventro timeline could not be mapped"
// @Failure 500 {string} string "Sync failed"
// @Failure 502 {string} string "Failed to fetch from Eventro"
// @Security BearerAuth
// @Router /eventrofairdates [get]
func FetchFairDatesEventro(w http.ResponseWriter, r *http.Request) {
	fairID, ok := requireEventroFairID(w, r)
	if !ok {
		return
	}

	fair, err := fetchEventroFairByID(
		r.Context(),
		&http.Client{Timeout: 30 * time.Second},
		eventroFairsURL,
		os.Getenv("EVENTRO_API"),
		os.Getenv("EVENTRO_ORG"),
		fairID,
	)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}

	item, err := mapEventroFairToFairDate(fair)
	if err != nil {
		log.Printf("cannot map Eventro fair timeline for fair %s: %v", fair.ID, err)
		http.Error(w, err.Error(), http.StatusUnprocessableEntity)
		return
	}

	err = replaceAllWithAudit(
		r,
		"fairdates",
		&item,
		func(tx *gorm.DB) error {
			return tx.Exec("LOCK TABLE fair_date_configs IN ACCESS EXCLUSIVE MODE").Error
		},
		func(tx *gorm.DB) error {
			if err := tx.Session(&gorm.Session{AllowGlobalUpdate: true}).
				Delete(&models.FairDateConfig{}).Error; err != nil {
				return err
			}
			return tx.Create(&item).Error
		},
		"dates",
	)
	if err != nil {
		log.Printf("failed to replace fair dates from Eventro fair %s: %v", fair.ID, err)
		http.Error(w, "failed to sync Eventro fair dates", http.StatusInternalServerError)
		return
	}

	_, _ = fmt.Fprint(w, "Sync completed - replaced all fair dates")
}

func mapEventroFairToFairDate(fair eventroFairResponse) (models.FairDateConfig, error) {
	eventroID := strings.TrimSpace(fair.ID)
	if eventroID == "" {
		return models.FairDateConfig{}, errors.New("eventro fair has no ID")
	}

	item := models.FairDateConfig{
		EventroID:   &eventroID,
		Description: strings.TrimSpace(fair.Name),
	}
	fairDays := make(map[string]struct{})

	for _, entry := range fair.TimelineEntries {
		milestone := classifyFairDateMilestone(entry.Title)
		if milestone == "" {
			continue
		}

		date, err := timelineDate(entry.Date)
		if err != nil {
			return models.FairDateConfig{}, fmt.Errorf("invalid date for timeline entry %q: %w", entry.Title, err)
		}

		switch milestone {
		case milestoneFairDay:
			fairDays[date] = struct{}{}
		case milestoneIRStart:
			assignEarlierDate(&item.IRStart, date)
		case milestoneIREnd:
			assignLaterDate(&item.IREnd, date)
		case milestoneIRAcceptance:
			assignEarlierDate(&item.IRAcceptance, date)
		case milestoneFRStart:
			assignEarlierDate(&item.FRStart, date)
		case milestoneFREnd:
			assignLaterDate(&item.FREnd, date)
		case milestoneEventsStart:
			assignEarlierDate(&item.EventsStart, date)
		}
	}

	days := make([]string, 0, len(fairDays))
	for day := range fairDays {
		days = append(days, day)
	}
	sort.Strings(days)
	item.FairDays = strings.Join(days, ",")

	missing := missingRequiredMilestones(item)
	if len(missing) > 0 {
		return models.FairDateConfig{}, fmt.Errorf(
			"eventro timeline is missing recognizable entries for: %s",
			strings.Join(missing, ", "),
		)
	}

	return item, nil
}

func classifyFairDateMilestone(title string) fairDateMilestone {
	normalized := normalizeTimelineTitle(title)
	words := strings.Fields(normalized)

	if containsAnyPhrase(normalized, "fair day", "fair date", "career fair day") {
		return milestoneFairDay
	}
	if strings.Contains(normalized, "acceptance") {
		return milestoneIRAcceptance
	}

	isOpening := containsAnyWord(words, "open", "opens", "opened", "opening", "start", "starts", "begin", "begins")
	isClosing := containsAnyWord(words, "close", "closes", "closed", "closing", "end", "ends", "deadline")

	isEventPeriod := containsAnyPhrase(
		normalized,
		"event week",
		"events week",
		"event period",
		"event start",
		"events start",
		"pre fair event",
		"pre fair events",
	)
	if isEventPeriod && isOpening {
		return milestoneEventsStart
	}

	isPriorityRegistration := strings.Contains(normalized, "priority") ||
		strings.Contains(normalized, "initial registration") ||
		containsAnyWord(words, "ir")
	if isPriorityRegistration {
		switch {
		case isOpening:
			return milestoneIRStart
		case isClosing:
			return milestoneIREnd
		}
	}

	isStandardRegistration := strings.Contains(normalized, "standard") ||
		strings.Contains(normalized, "final registration") ||
		containsAnyWord(words, "fr")
	if isStandardRegistration {
		switch {
		case isOpening:
			return milestoneFRStart
		case isClosing:
			return milestoneFREnd
		}
	}

	return ""
}

func normalizeTimelineTitle(value string) string {
	var normalized strings.Builder
	for _, r := range strings.ToLower(strings.TrimSpace(value)) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			normalized.WriteRune(r)
		} else {
			normalized.WriteByte(' ')
		}
	}
	return strings.Join(strings.Fields(normalized.String()), " ")
}

func containsAnyPhrase(value string, phrases ...string) bool {
	for _, phrase := range phrases {
		if strings.Contains(value, phrase) {
			return true
		}
	}
	return false
}

func containsAnyWord(words []string, candidates ...string) bool {
	for _, word := range words {
		for _, candidate := range candidates {
			if word == candidate {
				return true
			}
		}
	}
	return false
}

func timelineDate(value string) (string, error) {
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return "", err
	}
	return parsed.Format(time.DateOnly), nil
}

func assignEarlierDate(target *string, candidate string) {
	if *target == "" || candidate < *target {
		*target = candidate
	}
}

func assignLaterDate(target *string, candidate string) {
	if *target == "" || candidate > *target {
		*target = candidate
	}
}

func missingRequiredMilestones(item models.FairDateConfig) []string {
	required := []struct {
		name  string
		value string
	}{
		{name: "fair days", value: item.FairDays},
		{name: "priority registration opens", value: item.IRStart},
		{name: "priority registration closes", value: item.IREnd},
		{name: "acceptance date", value: item.IRAcceptance},
		{name: "standard registration opens", value: item.FRStart},
		{name: "standard registration closes", value: item.FREnd},
		{name: "event weeks start", value: item.EventsStart},
	}

	missing := make([]string, 0)
	for _, field := range required {
		if field.value == "" {
			missing = append(missing, field.name)
		}
	}
	return missing
}
