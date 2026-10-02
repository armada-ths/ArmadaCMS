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

func normalizeTimelineEra(era *models.TimelineEra) bool {
	era.Title = strings.TrimSpace(era.Title)
	return era.Title != "" && era.SortOrder >= 0
}

// GetTimelineEras lists available timeline eras.
// @Summary List timeline eras
// @Tags timeline-eras
// @Produce json
// @Param range query string false "Pagination range, e.g. [0,24]"
// @Success 200 {array} models.TimelineEra
// @Router /timeline-eras [get]
func GetTimelineEras(w http.ResponseWriter, r *http.Request) {
	params, _ := utils.ParseListParams(r.URL.Query())
	direction := "ASC"
	if rawSort := r.URL.Query().Get("sort"); rawSort != "" {
		var sort []string
		if err := json.Unmarshal([]byte(rawSort), &sort); err != nil ||
			len(sort) != 2 || sort[0] != "sortOrder" ||
			(sort[1] != "ASC" && sort[1] != "DESC") {
			http.Error(w, "unsupported era sort", http.StatusBadRequest)
			return
		}
		direction = sort[1]
	}
	query := db.DB.Model(&models.TimelineEra{})
	// React Admin's getMany uses filter={"id":[...]}. Keep reference fields
	// scoped to the requested eras, including when the table grows.
	var filters map[string]json.RawMessage
	if err := json.Unmarshal([]byte(r.URL.Query().Get("filter")), &filters); err == nil {
		if rawIDs, ok := filters["id"]; ok {
			var ids []uint
			if err := json.Unmarshal(rawIDs, &ids); err == nil {
				query = query.Where("id IN ?", ids)
			}
		}
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		http.Error(w, "List failed", http.StatusInternalServerError)
		return
	}
	query = query.Order("sort_order " + direction).Order("id " + direction)
	start := 0
	if r.URL.Query().Has("range") {
		start = params.Range[0]
		query = query.Offset(start).Limit(params.Range[1] - start + 1)
	}
	eras := make([]models.TimelineEra, 0)
	if err := query.Find(&eras).Error; err != nil {
		http.Error(w, "List failed", http.StatusInternalServerError)
		return
	}
	end := start
	if len(eras) > 0 {
		end = start + len(eras) - 1
	}
	w.Header().Set("Access-Control-Expose-Headers", "Content-Range")
	w.Header().Set("Content-Range", fmt.Sprintf("timeline-eras %d-%d/%d", start, end, total))
	writeJSONResponse(w, http.StatusOK, eras)
}

// GetTimelineEraByID returns a timeline era.
// @Summary Get timeline era by ID
// @Tags timeline-eras
// @Produce json
// @Param id path int true "Timeline Era ID"
// @Success 200 {object} models.TimelineEra
// @Router /timeline-eras/{id} [get]
func GetTimelineEraByID(w http.ResponseWriter, r *http.Request) {
	var era models.TimelineEra
	if err := db.DB.First(&era, mux.Vars(r)["id"]).Error; err != nil {
		http.Error(w, "timeline era not found", http.StatusNotFound)
		return
	}

	writeJSONResponse(w, http.StatusOK, era)
}

// CreateTimelineEra creates a timeline era.
// @Summary Create timeline era
// @Tags timeline-eras
// @Accept json
// @Produce json
// @Param body body models.TimelineEra true "Timeline era data"
// @Success 201 {object} models.TimelineEra
// @Security BearerAuth
// @Router /timeline-eras [post]
func CreateTimelineEra(w http.ResponseWriter, r *http.Request) {
	var era models.TimelineEra
	if err := json.NewDecoder(r.Body).Decode(&era); err != nil {
		http.Error(w, "Invalid body", http.StatusBadRequest)
		return
	}
	if !normalizeTimelineEra(&era) {
		http.Error(w, "title and non-negative sortOrder are required", http.StatusBadRequest)
		return
	}
	if err := createWithAudit(r, "timeline-eras", &era, func(tx *gorm.DB) error {
		return tx.Create(&era).Error
	}, nil, "timeline-entries"); err != nil {
		http.Error(w, "Create failed", http.StatusInternalServerError)
		return
	}
	writeCreatedJSONResponse(w, era)
}

// UpdateTimelineEra updates a timeline era.
// @Summary Update timeline era
// @Tags timeline-eras
// @Accept json
// @Produce json
// @Param id path int true "Timeline Era ID"
// @Param body body models.TimelineEra true "Timeline era data"
// @Success 200 {object} models.TimelineEra
// @Security BearerAuth
// @Router /timeline-eras/{id} [put]
func UpdateTimelineEra(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	var era models.TimelineEra
	if err := db.DB.First(&era, id).Error; err != nil {
		http.Error(w, "timeline era not found", http.StatusNotFound)
		return
	}
	var updates models.TimelineEra
	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		http.Error(w, "Invalid body", http.StatusBadRequest)
		return
	}
	if !normalizeTimelineEra(&updates) {
		http.Error(w, "title and non-negative sortOrder are required", http.StatusBadRequest)
		return
	}
	before := era
	if err := updateWithAudit(r, "timeline-eras", id, before, &era, func(tx *gorm.DB) error {
		return tx.Model(&era).Updates(map[string]any{"title": updates.Title, "sort_order": updates.SortOrder}).Error
	}, func(tx *gorm.DB) error {
		return tx.First(&era, id).Error
	}, "timeline-entries"); err != nil {
		http.Error(w, "Update failed", http.StatusInternalServerError)
		return
	}
	writeJSONResponse(w, http.StatusOK, era)
}

// DeleteTimelineEra deletes an empty timeline era.
// @Summary Delete timeline era
// @Tags timeline-eras
// @Produce json
// @Param id path int true "Timeline Era ID"
// @Success 204 {string} string "Deleted"
// @Security BearerAuth
// @Router /timeline-eras/{id} [delete]
func DeleteTimelineEra(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	var count int64
	if err := db.DB.Model(&models.TimelineEntry{}).Where("era_id = ?", id).Count(&count).Error; err != nil {
		http.Error(w, "Delete failed", http.StatusInternalServerError)
		return
	}
	if count > 0 {
		http.Error(w, "Move or delete this era's entries first", http.StatusConflict)
		return
	}
	writeDeleteResponseWithAudit[models.TimelineEra](w, r, "timeline-eras", id, "timeline era not found", nil, nil, "timeline-entries")
}
