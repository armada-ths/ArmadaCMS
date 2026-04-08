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

// GetFairDateConfigs returns a paginated list of fair date configurations.
// @Summary List fair date configs
// @Tags fair-dates
// @Produce json
// @Param range query string false "Pagination range, e.g. [0,24]"
// @Param sort query string false "Sort, e.g. [\"description\",\"ASC\"]"
// @Param filter query string false "Filter"
// @Success 200 {array} models.FairDateConfig
// @Header 200 {string} Content-Range "fairdates 0-24/5"
// @Security BearerAuth
// @Router /fairdates [get]
func GetFairDateConfigs(w http.ResponseWriter, r *http.Request) {
	params, _ := utils.ParseListParams(r.URL.Query())

	var items []models.FairDateConfig
	query := db.DB.Model(&models.FairDateConfig{})

	for k, v := range params.Filter {
		query = query.Where(utils.ToSnakeCase(k)+" = ?", v)
	}
	if len(params.Sort) == 2 {
		query = query.Order(utils.ToSnakeCase(params.Sort[0]) + " " + params.Sort[1])
	}

	start, end := params.Range[0], params.Range[1]

	var total int64
	db.DB.Model(&models.FairDateConfig{}).Count(&total)

	query = query.Offset(start).Limit(end - start + 1)
	query.Find(&items)

	w.Header().Set("Access-Control-Expose-Headers", "Content-Range")
	w.Header().Set("Content-Range", fmt.Sprintf("fairdates %d-%d/%d", start, end, total))
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(items)
}

// GetFairDateConfigByID returns a single fair date config by ID.
// @Summary Get fair date config by ID
// @Tags fair-dates
// @Produce json
// @Param id path int true "FairDateConfig ID"
// @Success 200 {object} models.FairDateConfig
// @Failure 404 {string} string "fair date config not found"
// @Security BearerAuth
// @Router /fairdates/{id} [get]
func GetFairDateConfigByID(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]

	var item models.FairDateConfig
	if err := db.DB.First(&item, id).Error; err != nil {
		http.Error(w, "fair date config not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(item)
}

// CreateFairDateConfig creates a new fair date configuration.
// @Summary Create fair date config
// @Tags fair-dates
// @Accept json
// @Produce json
// @Param body body models.FairDateConfig true "Fair date config data"
// @Success 201 {object} models.FairDateConfig
// @Failure 400 {string} string "Invalid body"
// @Failure 500 {string} string "Create failed"
// @Security BearerAuth
// @Router /fairdates [post]
func CreateFairDateConfig(w http.ResponseWriter, r *http.Request) {
	var item models.FairDateConfig
	if err := json.NewDecoder(r.Body).Decode(&item); err != nil {
		log.Println(err)
		http.Error(w, "Invalid body", http.StatusBadRequest)
		return
	}
	if err := createWithAudit(r, "fairdates", &item, func(tx *gorm.DB) error {
		return tx.Create(&item).Error
	}, nil); err != nil {
		http.Error(w, "Create failed", http.StatusInternalServerError)
		return
	}
	writeCreatedJSONResponse(w, item)
}

// UpdateFairDateConfig updates a fair date configuration by ID.
// @Summary Update fair date config
// @Tags fair-dates
// @Accept json
// @Produce json
// @Param id path int true "FairDateConfig ID"
// @Param body body models.FairDateConfig true "Updated fair date config"
// @Success 200 {object} models.FairDateConfig
// @Failure 400 {string} string "Invalid data"
// @Failure 404 {string} string "Not found"
// @Failure 500 {string} string "Update failed"
// @Security BearerAuth
// @Router /fairdates/{id} [put]
func UpdateFairDateConfig(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]

	var item models.FairDateConfig
	if err := db.DB.First(&item, id).Error; err != nil {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}

	var updates models.FairDateConfig
	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		http.Error(w, "Invalid data", http.StatusBadRequest)
		return
	}

	before := item
	if err := updateWithAudit(r, "fairdates", id, before, &item, func(tx *gorm.DB) error {
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

// DeleteFairDateConfig deletes a fair date configuration by ID.
// @Summary Delete fair date config
// @Tags fair-dates
// @Produce json
// @Param id path int true "FairDateConfig ID"
// @Success 200 {string} string "Deleted"
// @Failure 404 {string} string "fair date config not found"
// @Security BearerAuth
// @Router /fairdates/{id} [delete]
func DeleteFairDateConfig(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	writeDeleteResponseWithAudit[models.FairDateConfig](w, r, "fairdates", id, "fair date config not found", nil)
}
