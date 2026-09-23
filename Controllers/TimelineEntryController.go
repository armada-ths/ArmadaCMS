package controllers

import (
	"ArmadaCMS/main/db"
	"ArmadaCMS/main/models"
	"ArmadaCMS/main/utils"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
	"gorm.io/gorm"
)

// GetTimelineEntries returns the complete public timeline, or a page when range is set.
// @Summary List timeline entries
// @Tags timeline-entries
// @Produce json
// @Param range query string false "Pagination range, e.g. [0,24]"
// @Success 200 {array} models.TimelineEntry
// @Router /timeline-entries [get]
func GetTimelineEntries(w http.ResponseWriter, r *http.Request) {
	params, _ := utils.ParseListParams(r.URL.Query())
	// The only admin sort is the same era/entry order used by the public page.
	// Reject other sort requests instead of showing a misleading sorted header.
	if rawSort := r.URL.Query().Get("sort"); rawSort != "" {
		var sort []string
		if err := json.Unmarshal([]byte(rawSort), &sort); err != nil ||
			len(sort) != 2 || sort[0] != "timelineOrder" || sort[1] != "ASC" {
			http.Error(w, "unsupported timeline sort", http.StatusBadRequest)
			return
		}
	}
	query := db.DB.Model(&models.TimelineEntry{}).Joins("JOIN timeline_eras ON timeline_eras.id = timeline_entries.era_id")
	for key, value := range params.Filter {
		switch key {
		case "eraId":
			query = query.Where("timeline_entries.era_id = ?", value)
		case "title":
			query = query.Where("timeline_entries.title = ?", value)
		}
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		http.Error(w, "List failed", http.StatusInternalServerError)
		return
	}
	query = query.Preload("EraRecord").Order("timeline_eras.sort_order ASC").Order("timeline_eras.id ASC").Order("timeline_entries.sort_order ASC").Order("timeline_entries.id ASC")
	start := 0
	if r.URL.Query().Has("range") {
		start = params.Range[0]
		query = query.Offset(start).Limit(params.Range[1] - start + 1)
	}
	entries := make([]models.TimelineEntry, 0)
	if err := query.Find(&entries).Error; err != nil {
		http.Error(w, "List failed", http.StatusInternalServerError)
		return
	}
	for i := range entries {
		entries[i].SetEraPresentation()
	}
	end := start
	if len(entries) > 0 {
		end = start + len(entries) - 1
	}
	w.Header().Set("Access-Control-Expose-Headers", "Content-Range")
	w.Header().Set("Content-Range", fmt.Sprintf("timeline-entries %d-%d/%d", start, end, total))
	writeJSONResponse(w, http.StatusOK, entries)
}

// GetTimelineEntryByID returns a single timeline entry by ID.
// @Summary Get timeline entry by ID
// @Tags timeline-entries
// @Produce json
// @Param id path int true "Timeline Entry ID"
// @Success 200 {object} models.TimelineEntry
// @Router /timeline-entries/{id} [get]
func GetTimelineEntryByID(w http.ResponseWriter, r *http.Request) {
	var entry models.TimelineEntry
	if err := db.DB.Preload("EraRecord").First(&entry, mux.Vars(r)["id"]).Error; err != nil {
		http.Error(w, "timeline entry not found", http.StatusNotFound)
		return
	}
	entry.SetEraPresentation()
	writeJSONResponse(w, http.StatusOK, entry)
}

func normalizeTimelineEntry(entry *models.TimelineEntry) bool {
	entry.Title = strings.TrimSpace(entry.Title)
	entry.Body = strings.TrimSpace(entry.Body)
	return entry.Title != "" && entry.Body != "" && entry.EraID != 0 && entry.SortOrder >= 0
}

func validateTimelineEntry(w http.ResponseWriter, entry *models.TimelineEntry) bool {
	if !normalizeTimelineEntry(entry) {
		http.Error(w, "title, body, eraId and non-negative sortOrder are required", http.StatusBadRequest)
		return false
	}
	if err := db.DB.First(&models.TimelineEra{}, entry.EraID).Error; err != nil {
		http.Error(w, "era not found", http.StatusBadRequest)
		return false
	}
	return true
}

// CreateTimelineEntry creates a timeline entry.
// @Summary Create timeline entry
// @Tags timeline-entries
// @Accept json
// @Produce json
// @Param body body models.TimelineEntry true "Timeline entry data"
// @Success 201 {object} models.TimelineEntry
// @Security BearerAuth
// @Router /timeline-entries [post]
func CreateTimelineEntry(w http.ResponseWriter, r *http.Request) {
	var entry models.TimelineEntry
	if err := json.NewDecoder(r.Body).Decode(&entry); err != nil {
		http.Error(w, "Invalid body", http.StatusBadRequest)
		return
	}
	if !validateTimelineEntry(w, &entry) {
		return
	}
	if err := createWithAudit(r, "timeline-entries", &entry, func(tx *gorm.DB) error {
		return tx.Create(&entry).Error
	}, func(tx *gorm.DB) error {
		return tx.Preload("EraRecord").First(&entry, entry.ID).Error
	}, "timeline-entries"); err != nil {
		http.Error(w, "Create failed", http.StatusInternalServerError)
		return
	}
	entry.SetEraPresentation()
	writeCreatedJSONResponse(w, entry)
}

// UpdateTimelineEntry updates a timeline entry by ID.
// @Summary Update timeline entry
// @Tags timeline-entries
// @Accept json
// @Produce json
// @Param id path int true "Timeline Entry ID"
// @Param body body models.TimelineEntry true "Updated timeline entry data"
// @Success 200 {object} models.TimelineEntry
// @Security BearerAuth
// @Router /timeline-entries/{id} [put]
func UpdateTimelineEntry(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	var entry models.TimelineEntry
	if err := db.DB.Preload("EraRecord").First(&entry, id).Error; err != nil {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}
	entry.SetEraPresentation()
	var updates models.TimelineEntry
	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		http.Error(w, "Invalid body", http.StatusBadRequest)
		return
	}
	if !validateTimelineEntry(w, &updates) {
		return
	}
	before := entry
	if err := updateWithAudit(r, "timeline-entries", id, before, &entry, func(tx *gorm.DB) error {
		return tx.Model(&entry).Updates(map[string]any{
			"title": updates.Title, "body": updates.Body,
			"era_id": updates.EraID, "sort_order": updates.SortOrder,
		}).Error
	}, func(tx *gorm.DB) error {
		return tx.Preload("EraRecord").First(&entry, id).Error
	}, "timeline-entries"); err != nil {
		http.Error(w, "Update failed", http.StatusInternalServerError)
		return
	}
	entry.SetEraPresentation()
	writeJSONResponse(w, http.StatusOK, entry)
}

// DeleteTimelineEntry deletes a timeline entry by ID.
// @Summary Delete timeline entry
// @Tags timeline-entries
// @Produce json
// @Param id path int true "Timeline Entry ID"
// @Success 204 {string} string "Deleted"
// @Security BearerAuth
// @Router /timeline-entries/{id} [delete]
func DeleteTimelineEntry(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	writeDeleteResponseWithAudit[models.TimelineEntry](w, r, "timeline-entries", id, "timeline entry not found", nil, nil, "timeline-entries")
}
