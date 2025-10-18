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

func GetExhibitors(w http.ResponseWriter, r *http.Request) {
	params, _ := utils.ParseListParams(r.URL.Query())
	log.Print(params)
	var exhibitors []models.Exhibitor
	query := db.DB.Model(&models.Exhibitor{})

	for k, v := range params.Filter {
		query = query.Where(k+" = ?", v)
	}

	if len(params.Sort) == 2 {
		query = query.Order(params.Sort[0] + " " + params.Sort[1])
	}

	all := r.URL.Query().Get("limit") == "all" || r.URL.Query().Get("all") == "true"

	var total int64
	db.DB.Model(&models.Exhibitor{}).Count(&total)

	if all {
		// No limit — return all exhibitors
		query.Preload("Industries").Preload("Programs").Preload("Employments").Find(&exhibitors)

		// Set Content-Range header to full range
		w.Header().Set("Access-Control-Expose-Headers", "Content-Range")
		w.Header().Set("Content-Range", fmt.Sprintf("exhibitors 0-%d/%d", total-1, total))
	} else {
		// Default paginated behavior
		start, end := params.Range[0], params.Range[1]
		limit := end - start + 1
		query = query.Offset(start).Limit(limit)
		query.Preload("Industries").Preload("Programs").Preload("Employments").Find(&exhibitors)

		w.Header().Set("Access-Control-Expose-Headers", "Content-Range")
		w.Header().Set("Content-Range", fmt.Sprintf("exhibitors %d-%d/%d", start, end, total))
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(exhibitors)
}

func GetExhibitorByID(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	var exhibitor models.Exhibitor
	if err := db.DB.Preload("Industries").Preload("Programs").Preload("Employments").First(&exhibitor, id).Error; err != nil {
		http.Error(w, "Exhibitor not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(exhibitor)
}

func CreateExhibitor(w http.ResponseWriter, r *http.Request) {
	var exhibitor models.Exhibitor
	if err := json.NewDecoder(r.Body).Decode(&exhibitor); err != nil {
		http.Error(w, "Invalid JSON body", http.StatusBadRequest)
		return
	}

	if err := db.DB.Create(&exhibitor).Error; err != nil {
		http.Error(w, "Create failed", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(exhibitor)
}

func UpdateExhibitor(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]

	var exhibitor models.Exhibitor
	if err := db.DB.Preload("Programs").Preload("Industries").First(&exhibitor, id).Error; err != nil {
		http.Error(w, "Exhibitor not found", http.StatusNotFound)
		return
	}

	var updates models.Exhibitor
	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		log.Println(err)
		http.Error(w, "Invalid JSON body", http.StatusBadRequest)
		return
	}

	// Basic field updates
	if err := db.DB.Model(&exhibitor).Updates(updates).Error; err != nil {
		http.Error(w, "Failed to update exhibitor", http.StatusInternalServerError)
		return
	}

	// Handle many-to-many manually
	log.Println(updates)
	if updates.Programs != nil {
		log.Println(updates.Programs)
		if err := db.DB.Model(&exhibitor).Association("Programs").Replace(updates.Programs); err != nil {
			http.Error(w, "Failed to update programs", http.StatusInternalServerError)
			return
		}
	}

	if updates.Industries != nil {
		if err := db.DB.Model(&exhibitor).Association("Industries").Replace(updates.Industries); err != nil {
			http.Error(w, "Failed to update industries", http.StatusInternalServerError)
			return
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(exhibitor)
}
func DeleteExhibitor(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	if err := db.DB.Delete(&models.Exhibitor{}, id).Error; err != nil {
		http.Error(w, "Delete failed", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
