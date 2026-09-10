package controllers

import (
	"ArmadaCMS/main/db"
	"ArmadaCMS/main/models"
	"ArmadaCMS/main/utils"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
	"gorm.io/gorm"
)

// GetTimelineEntries returns a paginated list of timeline entries.
// @Summary List timeline entries
// @Tags timeline-entries
// @Produce json
// @Param range query string false "Pagination range, e.g. [0,24]"
// @Param sort query string false "Sort, e.g. [\"sort_order\",\"ASC\"]"
// @Param filter query string false "Filter, e.g. {\"era\":\"1980s\"}"
// @Success 200 {array} models.TimelineEntry
// @Header 200 {string} Content-Range "timeline-entries 0-24/10"
// @Router /timeline-entries [get]
func GetTimelineEntries(w http.ResponseWriter, r *http.Request) {
	params, _ := utils.ParseListParams(r.URL.Query())

	var entries []models.TimelineEntry
	query := db.DB.Model(&models.TimelineEntry{})

	for k, v := range params.Filter {
		query = query.Where(k+" = ?", v)
	}

	if len(params.Sort) == 2 {
		query = query.Order(params.Sort[0] + " " + params.Sort[1])
	}

	start, end := params.Range[0], params.Range[1]
	limit := end - start + 1

	var total int64
	db.DB.Model(&models.TimelineEntry{}).Count(&total)

	query = query.Offset(start).Limit(limit)
	query.Find(&entries)

	w.Header().Set("Access-Control-Expose-Headers", "Content-Range")
	w.Header().Set("Content-Range", fmt.Sprintf("timeline-entries %d-%d/%d", start, end, total))
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(entries)
}

// GetTimelineEntryByID returns a single timeline entry by ID.
// @Summary Get timeline entry by ID
// @Tags timeline-entries
// @Produce json
// @Param id path int true "Timeline Entry ID"
// @Success 200 {object} models.TimelineEntry
// @Failure 404 {string} string "timeline entry not found"
// @Router /timeline-entries/{id} [get]
func GetTimelineEntryByID(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	var entry models.TimelineEntry
	if err := db.DB.First(&entry, id).Error; err != nil {
		http.Error(w, "timeline entry not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(entry)
}

// CreateTimelineEntry creates a new timeline entry.
// @Summary Create timeline entry
// @Tags timeline-entries
// @Accept json
// @Produce json
// @Param body body models.TimelineEntry true "Timeline entry data"
// @Success 201 {object} models.TimelineEntry
// @Failure 400 {string} string "Invalid body"
// @Failure 500 {string} string "Create failed"
// @Security BearerAuth
// @Router /timeline-entries [post]
func CreateTimelineEntry(w http.ResponseWriter, r *http.Request) {
	var entry models.TimelineEntry
	if err := json.NewDecoder(r.Body).Decode(&entry); err != nil {
		http.Error(w, "Invalid body", http.StatusBadRequest)
		return
	}
	if err := createWithAudit(r, "timeline-entries", &entry, func(tx *gorm.DB) error {
		return tx.Create(&entry).Error
	}, nil, "timeline"); err != nil {
		http.Error(w, "Create failed", http.StatusInternalServerError)
		return
	}

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
// @Failure 400 {string} string "Invalid data"
// @Failure 404 {string} string "Not found"
// @Failure 500 {string} string "Update failed"
// @Security BearerAuth
// @Router /timeline-entries/{id} [put]
func UpdateTimelineEntry(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	var entry models.TimelineEntry
	if err := db.DB.First(&entry, id).Error; err != nil {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}

	var updates models.TimelineEntry
	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		http.Error(w, "Invalid data", http.StatusBadRequest)
		return
	}

	before := entry
	if err := updateWithAudit(r, "timeline-entries", id, before, &entry, func(tx *gorm.DB) error {
		return tx.Model(&entry).Updates(updates).Error
	}, func(tx *gorm.DB) error {
		return tx.First(&entry, id).Error
	}, "timeline"); err != nil {
		http.Error(w, "Update failed", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(entry)
}

// DeleteTimelineEntry deletes a timeline entry by ID.
// @Summary Delete timeline entry
// @Tags timeline-entries
// @Produce json
// @Param id path int true "Timeline Entry ID"
// @Success 204 {string} string "Deleted"
// @Failure 404 {string} string "timeline entry not found"
// @Security BearerAuth
// @Router /timeline-entries/{id} [delete]
func DeleteTimelineEntry(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	writeDeleteResponseWithAudit[models.TimelineEntry](w, r, "timeline-entries", id, "timeline entry not found", nil, nil, "timeline")
}
