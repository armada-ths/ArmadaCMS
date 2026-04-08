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

// GetEmployments returns a paginated list of employment types.
// @Summary List employment types
// @Tags employments
// @Produce json
// @Param range query string false "Pagination range, e.g. [0,24]"
// @Param sort query string false "Sort, e.g. [\"name\",\"ASC\"]"
// @Param filter query string false "Filter, e.g. {\"name\":\"Full-time\"}"
// @Success 200 {array} models.Employment
// @Header 200 {string} Content-Range "employments 0-24/10"
// @Router /employments [get]
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

// GetEmploymentByID returns a single employment type by ID.
// @Summary Get employment by ID
// @Tags employments
// @Produce json
// @Param id path int true "Employment ID"
// @Success 200 {object} models.Employment
// @Failure 404 {string} string "employment not found"
// @Router /employments/{id} [get]
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

// CreateEmployment creates a new employment type.
// @Summary Create employment
// @Tags employments
// @Accept json
// @Produce json
// @Param body body models.Employment true "Employment data"
// @Success 201 {object} models.Employment
// @Failure 400 {string} string "Invalid body"
// @Failure 500 {string} string "Create failed"
// @Security BearerAuth
// @Router /employments [post]
func CreateEmployment(w http.ResponseWriter, r *http.Request) {
	var item models.Employment
	if err := json.NewDecoder(r.Body).Decode(&item); err != nil {
		log.Println(err)
		http.Error(w, "Invalid body", http.StatusBadRequest)
		return
	}
	if err := createWithAudit(r, "employments", &item, func(tx *gorm.DB) error {
		return tx.Create(&item).Error
	}, func(tx *gorm.DB) error {
		return tx.Preload("Exhibitor").First(&item, item.ID).Error
	}); err != nil {
		http.Error(w, "Create failed", http.StatusInternalServerError)
		return
	}

	writeCreatedJSONResponse(w, item)
}

// UpdateEmployment updates an employment type by ID.
// @Summary Update employment
// @Tags employments
// @Accept json
// @Produce json
// @Param id path int true "Employment ID"
// @Param body body models.Employment true "Updated employment data"
// @Success 200 {object} models.Employment
// @Failure 400 {string} string "Invalid data"
// @Failure 404 {string} string "Not found"
// @Failure 500 {string} string "Update failed"
// @Security BearerAuth
// @Router /employments/{id} [put]
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

	before := item
	if err := updateWithAudit(r, "employments", id, before, &item, func(tx *gorm.DB) error {
		return tx.Model(&item).Updates(updates).Error
	}, func(tx *gorm.DB) error {
		return tx.Preload("Exhibitor").First(&item, id).Error
	}); err != nil {
		http.Error(w, "Update failed", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(item)
}

// DeleteEmployment deletes an employment type by ID.
// @Summary Delete employment
// @Tags employments
// @Produce json
// @Param id path int true "Employment ID"
// @Success 200 {string} string "Deleted"
// @Failure 404 {string} string "employment not found"
// @Security BearerAuth
// @Router /employments/{id} [delete]
func DeleteEmployment(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	writeDeleteResponseWithAudit[models.Employment](w, r, "employments", id, "employment not found", func(tx *gorm.DB) *gorm.DB {
		return tx.Preload("Exhibitor")
	})
}
