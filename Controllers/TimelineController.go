package controllers

import (
	"ArmadaCMS/main/db"
	"ArmadaCMS/main/models"
	"ArmadaCMS/main/utils"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/mux"
)

// GetTimelineDates - list with filter/sort/range
func GetTimelineDates(w http.ResponseWriter, r *http.Request) {
	params, _ := utils.ParseListParams(r.URL.Query())

	var timelines []models.TimelineDate
	query := db.DB.Model(&models.TimelineDate{})

	for k, v := range params.Filter {
		query = query.Where(k+" = ?", v)
	}

	if len(params.Sort) == 2 {
		query = query.Order(params.Sort[0] + " " + params.Sort[1])
	}

	start, end := params.Range[0], params.Range[1]
	limit := end - start + 1

	var total int64
	db.DB.Model(&models.TimelineDate{}).Count(&total)

	query = query.Offset(start).Limit(limit)
	query.Find(&timelines)

	w.Header().Set("Access-Control-Expose-Headers", "Content-Range")
	w.Header().Set("Content-Range", fmt.Sprintf("timelines %d-%d/%d", start, end, total))
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(timelines)
}

// GetTimelineDateByID - single record fetch
func GetTimelineDateByID(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]

	var timeline models.TimelineDate
	if err := db.DB.First(&timeline, id).Error; err != nil {
		http.Error(w, "timeline not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(timeline)
}

// CreateTimelineDate - insert
func CreateTimelineDate(w http.ResponseWriter, r *http.Request) {
	var timeline models.TimelineDate
	if err := json.NewDecoder(r.Body).Decode(&timeline); err != nil {
		log.Println(err)
		http.Error(w, "Invalid body", http.StatusBadRequest)
		return
	}
	if err := db.DB.Create(&timeline).Error; err != nil {
		http.Error(w, "Create failed", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(timeline)
}

// UpdateTimelineDate - update
func UpdateTimelineDate(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]

	var timeline models.TimelineDate
	if err := db.DB.First(&timeline, id).Error; err != nil {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}

	var updates models.TimelineDate
	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		http.Error(w, "Invalid data", http.StatusBadRequest)
		return
	}

	db.DB.Model(&timeline).Updates(updates)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(timeline)
}

// DeleteTimelineDate - remove
func DeleteTimelineDate(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	if err := db.DB.Delete(&models.TimelineDate{}, id).Error; err != nil {
		http.Error(w, "Delete failed", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
