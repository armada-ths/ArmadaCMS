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

// GetPrograms returns a paginated list of programs.
// @Summary List programs
// @Tags programs
// @Produce json
// @Param range query string false "Pagination range, e.g. [0,24]"
// @Param sort query string false "Sort, e.g. [\"name\",\"ASC\"]"
// @Param filter query string false "Filter, e.g. {\"name\":\"Computer Science\"}"
// @Success 200 {array} models.Program
// @Header 200 {string} Content-Range "programs 0-24/50"
// @Router /programs [get]
func GetPrograms(w http.ResponseWriter, r *http.Request) {
	params, _ := utils.ParseListParams(r.URL.Query())

	var items []models.Program
	query := db.DB.Model(&models.Program{})

	for k, v := range params.Filter {
		query = query.Where(k+" = ?", v)
	}
	if len(params.Sort) == 2 {
		query = query.Order(params.Sort[0] + " " + params.Sort[1])
	}

	start, end := params.Range[0], params.Range[1]

	var total int64
	db.DB.Model(&models.Program{}).Count(&total)

	query = query.Offset(start).Limit(end - start + 1)
	query.Find(&items)

	w.Header().Set("Access-Control-Expose-Headers", "Content-Range")
	w.Header().Set("Content-Range", fmt.Sprintf("programs %d-%d/%d", start, end, total))
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(items)
}

// GetProgramByID returns a single program by ID.
// @Summary Get program by ID
// @Tags programs
// @Produce json
// @Param id path int true "Program ID"
// @Success 200 {object} models.Program
// @Failure 404 {string} string "program not found"
// @Router /programs/{id} [get]
func GetProgramByID(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]

	var item models.Program
	if err := db.DB.First(&item, id).Error; err != nil {
		http.Error(w, "program not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(item)
}

// CreateProgram creates a new program.
// @Summary Create program
// @Tags programs
// @Accept json
// @Produce json
// @Param body body models.Program true "Program data"
// @Success 201 {object} models.Program
// @Failure 400 {string} string "Invalid body"
// @Failure 500 {string} string "Create failed"
// @Security BearerAuth
// @Router /programs [post]
func CreateProgram(w http.ResponseWriter, r *http.Request) {
	var item models.Program
	if err := json.NewDecoder(r.Body).Decode(&item); err != nil {
		log.Println(err)
		http.Error(w, "Invalid body", http.StatusBadRequest)
		return
	}
	if err := createWithAudit(r, "programs", &item, func(tx *gorm.DB) error {
		return tx.Create(&item).Error
	}, nil, "programs"); err != nil {
		http.Error(w, "Create failed", http.StatusInternalServerError)
		return
	}
	writeCreatedJSONResponse(w, item)
}

// UpdateProgram updates a program by ID.
// @Summary Update program
// @Tags programs
// @Accept json
// @Produce json
// @Param id path int true "Program ID"
// @Param body body models.Program true "Updated program data"
// @Success 200 {object} models.Program
// @Failure 400 {string} string "Invalid data"
// @Failure 404 {string} string "Not found"
// @Failure 500 {string} string "Update failed"
// @Security BearerAuth
// @Router /programs/{id} [put]
func UpdateProgram(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]

	var item models.Program
	if err := db.DB.First(&item, id).Error; err != nil {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}

	var updates models.Program
	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		http.Error(w, "Invalid data", http.StatusBadRequest)
		return
	}

	before := item
	if err := updateWithAudit(r, "programs", id, before, &item, func(tx *gorm.DB) error {
		return tx.Model(&item).Updates(updates).Error
	}, func(tx *gorm.DB) error {
		return tx.First(&item, id).Error
	}, "programs"); err != nil {
		http.Error(w, "Update failed", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(item)
}

// DeleteProgram deletes a program by ID.
// @Summary Delete program
// @Tags programs
// @Produce json
// @Param id path int true "Program ID"
// @Success 200 {string} string "Deleted"
// @Failure 404 {string} string "program not found"
// @Security BearerAuth
// @Router /programs/{id} [delete]
func DeleteProgram(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	writeDeleteResponseWithAudit[models.Program](w, r, "programs", id, "program not found", nil, "programs")
}
