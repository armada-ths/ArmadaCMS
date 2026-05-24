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

// GetIndustries returns a paginated list of industries.
// @Summary List industries
// @Tags industries
// @Produce json
// @Param range query string false "Pagination range, e.g. [0,24]"
// @Param sort query string false "Sort, e.g. [\"name\",\"ASC\"]"
// @Param filter query string false "Filter, e.g. {\"name\":\"Finance\"}"
// @Success 200 {array} models.Industry
// @Header 200 {string} Content-Range "industries 0-24/50"
// @Router /industries [get]
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

// GetIndustryByID returns a single industry by ID.
// @Summary Get industry by ID
// @Tags industries
// @Produce json
// @Param id path int true "Industry ID"
// @Success 200 {object} models.Industry
// @Failure 404 {string} string "industry not found"
// @Router /industries/{id} [get]
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

// CreateIndustry creates a new industry.
// @Summary Create industry
// @Tags industries
// @Accept json
// @Produce json
// @Param body body models.Industry true "Industry data"
// @Success 201 {object} models.Industry
// @Failure 400 {string} string "Invalid body"
// @Failure 500 {string} string "Create failed"
// @Security BearerAuth
// @Router /industries [post]
func CreateIndustry(w http.ResponseWriter, r *http.Request) {
	var item models.Industry
	if err := json.NewDecoder(r.Body).Decode(&item); err != nil {
		log.Println(err)
		http.Error(w, "Invalid body", http.StatusBadRequest)
		return
	}
	if err := createWithAudit(r, "industries", &item, func(tx *gorm.DB) error {
		return tx.Create(&item).Error
	}, nil, "industries"); err != nil {
		http.Error(w, "Create failed", http.StatusInternalServerError)
		return
	}
	writeCreatedJSONResponse(w, item)
}

// UpdateIndustry updates an industry by ID.
// @Summary Update industry
// @Tags industries
// @Accept json
// @Produce json
// @Param id path int true "Industry ID"
// @Param body body models.Industry true "Updated industry data"
// @Success 200 {object} models.Industry
// @Failure 400 {string} string "Invalid data"
// @Failure 404 {string} string "Not found"
// @Failure 500 {string} string "Update failed"
// @Security BearerAuth
// @Router /industries/{id} [put]
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
	}, "industries"); err != nil {
		http.Error(w, "Update failed", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(item)
}

// DeleteIndustry deletes an industry by ID.
// @Summary Delete industry
// @Tags industries
// @Produce json
// @Param id path int true "Industry ID"
// @Success 200 {string} string "Deleted"
// @Failure 404 {string} string "industry not found"
// @Security BearerAuth
// @Router /industries/{id} [delete]
func DeleteIndustry(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	writeDeleteResponseWithAudit[models.Industry](w, r, "industries", id, "industry not found", nil, nil, "industries")
}
