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

func GetEmployments(w http.ResponseWriter, r *http.Request) {
	params, _ := utils.ParseListParams(r.URL.Query())

	var items []models.Employment
	query := db.DB.Model(&models.Employment{})

	for k, v := range params.Filter {
		query = query.Where(k+" = ?", v)
	}
	if len(params.Sort) == 2 {
		query = query.Order(params.Sort[0] + " " + params.Sort[1])
	}

	start, end := params.Range[0], params.Range[1]

	var total int64
	db.DB.Model(&models.Employment{}).Count(&total)

	query = query.Offset(start).Limit(end - start + 1)
	query.Find(&items)

	w.Header().Set("Access-Control-Expose-Headers", "Content-Range")
	w.Header().Set("Content-Range", fmt.Sprintf("employments %d-%d/%d", start, end, total))
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(items)
}

func GetEmploymentByID(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]

	var item models.Employment
	if err := db.DB.First(&item, id).Error; err != nil {
		http.Error(w, "employment not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(item)
}

func CreateEmployment(w http.ResponseWriter, r *http.Request) {
	var item models.Employment
	if err := json.NewDecoder(r.Body).Decode(&item); err != nil {
		log.Println(err)
		http.Error(w, "Invalid body", http.StatusBadRequest)
		return
	}
	if err := db.DB.Create(&item).Error; err != nil {
		http.Error(w, "Create failed", http.StatusInternalServerError)
		return
	}
	// return with exhibitor preloaded
	db.DB.Preload("Exhibitor").First(&item, item.ID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(item)
}

func UpdateEmployment(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]

	var item models.Employment
	if err := db.DB.First(&item, id).Error; err != nil {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}

	var updates models.Employment
	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		http.Error(w, "Invalid data", http.StatusBadRequest)
		return
	}

	db.DB.Model(&item).Updates(updates)
	db.DB.Preload("Exhibitor").First(&item, id)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(item)
}

func DeleteEmployment(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	if err := db.DB.Delete(&models.Employment{}, id).Error; err != nil {
		http.Error(w, "Delete failed", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
