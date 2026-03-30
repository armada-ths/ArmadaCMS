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
	"gorm.io/gorm"
)

func GetIndustries(w http.ResponseWriter, r *http.Request) {
	params, _ := utils.ParseListParams(r.URL.Query())

	var items []models.Industry
	query := db.DB.Model(&models.Industry{})

	for k, v := range params.Filter {
		query = query.Where(k+" = ?", v)
	}
	if len(params.Sort) == 2 {
		query = query.Order(params.Sort[0] + " " + params.Sort[1])
	}

	start, end := params.Range[0], params.Range[1]

	var total int64
	db.DB.Model(&models.Industry{}).Count(&total)

	query = query.Offset(start).Limit(end - start + 1)
	query.Find(&items)

	w.Header().Set("Access-Control-Expose-Headers", "Content-Range")
	w.Header().Set("Content-Range", fmt.Sprintf("industries %d-%d/%d", start, end, total))
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(items)
}

func GetIndustryByID(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]

	var item models.Industry
	if err := db.DB.First(&item, id).Error; err != nil {
		http.Error(w, "industry not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(item)
}

func CreateIndustry(w http.ResponseWriter, r *http.Request) {
	var item models.Industry
	if err := json.NewDecoder(r.Body).Decode(&item); err != nil {
		log.Println(err)
		http.Error(w, "Invalid body", http.StatusBadRequest)
		return
	}
	if err := createWithAudit(r, "industries", &item, func(tx *gorm.DB) error {
		return tx.Create(&item).Error
	}, nil); err != nil {
		http.Error(w, "Create failed", http.StatusInternalServerError)
		return
	}
	writeCreatedJSONResponse(w, item)
}

func UpdateIndustry(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]

	var item models.Industry
	if err := db.DB.First(&item, id).Error; err != nil {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}

	var updates models.Industry
	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		http.Error(w, "Invalid data", http.StatusBadRequest)
		return
	}

	before := item
	if err := updateWithAudit(r, "industries", id, before, &item, func(tx *gorm.DB) error {
		return tx.Model(&item).Updates(updates).Error
	}, func(tx *gorm.DB) error {
		return tx.First(&item, id).Error
	}); err != nil {
		http.Error(w, "Update failed", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(item)
}

func DeleteIndustry(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	writeDeleteResponseWithAudit[models.Industry](w, r, "industries", id, "industry not found", nil)
}
