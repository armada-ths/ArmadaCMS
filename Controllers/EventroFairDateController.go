package controllers

import (
	"ArmadaCMS/main/audit"
	"ArmadaCMS/main/db"
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
	"ArmadaCMS/main/utils"

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

	group, err := startAuditGroup(r, "sync", "fairdates", fairID, map[string]any{"source": "eventro", "status": "running"})
	if err != nil {
		http.Error(w, "failed to start audit log", http.StatusInternalServerError)
		return
	}
	stats, err := syncFairDateConfigWithAudit(audit.WithParent(r, group.ID), item)
	if err != nil {
		_ = finalizeAuditGroup(group.ID, "failed", map[string]any{"source": "eventro", "error": err.Error()})
		log.Printf("failed to sync fair dates from Eventro fair %s: %v", fair.ID, err)
		http.Error(w, "failed to sync Eventro fair dates", http.StatusInternalServerError)
		return
	}
	if err := finalizeAuditGroup(group.ID, "completed", map[string]any{
		"source":    "eventro",
		"inserted":  stats.inserted,
		"updated":   stats.updated,
		"unchanged": stats.unchanged,
		"deleted":   stats.deleted,
		"failed":    0,
	}); err != nil {
		http.Error(w, "failed to finalize audit log", http.StatusInternalServerError)
		return
	}
	if stats.inserted+stats.updated+stats.deleted > 0 {
		go utils.RevalidateTag("dates")
	}

	log.Printf(
		"✅ Fair date sync completed — inserted: %d, updated: %d, deleted: %d",
		stats.inserted,
		stats.updated,
		stats.deleted,
	)
	_, _ = fmt.Fprintf(w, "Sync completed — inserted: %d, updated: %d", stats.inserted, stats.updated)
}

type fairDateSyncStats struct {
	inserted  int
	updated   int
	unchanged int
	deleted   int
}

func syncFairDateConfigWithAudit(childRequest *http.Request, item models.FairDateConfig) (fairDateSyncStats, error) {
	stats := fairDateSyncStats{}
	err := db.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec("LOCK TABLE fair_date_configs IN ACCESS EXCLUSIVE MODE").Error; err != nil {
			return err
		}

		keepID, err := upsertFairDateConfigWithAudit(tx, childRequest, &item, &stats)
		if err != nil {
			return err
		}

		return deleteStaleFairDateConfigsWithAudit(tx, childRequest, keepID, &stats)
	})
	return stats, err
}

func upsertFairDateConfigWithAudit(tx *gorm.DB, childRequest *http.Request, item *models.FairDateConfig, stats *fairDateSyncStats) (uint, error) {
	if item.EventroID == nil {
		return 0, errors.New("eventro fair has no ID")
	}

	var existing models.FairDateConfig
	findErr := tx.Where("eventro_id = ?", *item.EventroID).First(&existing).Error
	if findErr != nil && findErr != gorm.ErrRecordNotFound {
		return 0, findErr
	}
	if findErr == gorm.ErrRecordNotFound {
		if err := tx.Create(item).Error; err != nil {
			return 0, err
		}
		stats.inserted = 1
		return item.ID, audit.LogCreate(tx, childRequest, "fairdates", item.ID, *item)
	}

	item.ID = existing.ID
	if fairDateConfigsEqual(existing, *item) {
		stats.unchanged = 1
		return existing.ID, nil
	}

	before := existing
	existing.EventroID = item.EventroID
	existing.Description = item.Description
	existing.FairDays = item.FairDays
	existing.IRStart = item.IRStart
	existing.IREnd = item.IREnd
	existing.IRAcceptance = item.IRAcceptance
	existing.FRStart = item.FRStart
	existing.FREnd = item.FREnd
	existing.EventsStart = item.EventsStart

	if err := tx.Save(&existing).Error; err != nil {
		return 0, err
	}

	stats.updated = 1
	return existing.ID, audit.LogUpdate(tx, childRequest, "fairdates", existing.ID, before, existing)
}

func deleteStaleFairDateConfigsWithAudit(tx *gorm.DB, childRequest *http.Request, keepID uint, stats *fairDateSyncStats) error {
	var stale []models.FairDateConfig
	if err := tx.Where("id <> ?", keepID).Find(&stale).Error; err != nil {
		return err
	}

	for i := range stale {
		if err := tx.Delete(&stale[i]).Error; err != nil {
			return err
		}
		if err := audit.LogDelete(tx, childRequest, "fairdates", stale[i].ID, stale[i]); err != nil {
			return err
		}
	}

	stats.deleted = len(stale)
	return nil
}

func fairDateConfigsEqual(left, right models.FairDateConfig) bool {
	return stringPointersEqual(left.EventroID, right.EventroID) &&
		left.Description == right.Description &&
		left.FairDays == right.FairDays &&
		left.IRStart == right.IRStart &&
		left.IREnd == right.IREnd &&
		left.IRAcceptance == right.IRAcceptance &&
		left.FRStart == right.FRStart &&
		left.FREnd == right.FREnd &&
		left.EventsStart == right.EventsStart
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
