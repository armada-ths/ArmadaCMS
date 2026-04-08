package controllers

import (
	"ArmadaCMS/main/db"
	"ArmadaCMS/main/models"
	"ArmadaCMS/main/utils"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
	"gorm.io/gorm"
)

type recruitmentRolePayload struct {
	EventroRoleID *string `json:"eventroRoleId"`
	RecruitmentID *uint   `json:"recruitmentId"`
	TeamID        *uint   `json:"team_id"`
	Name          string  `json:"name"`
	Description   string  `json:"description"`
}

// GetRecruitmentRoles returns a paginated list of recruitment roles.
// @Summary List recruitment roles
// @Tags recruitment
// @Produce json
// @Param range query string false "Pagination range, e.g. [0,24]"
// @Param sort query string false "Sort, e.g. [\"name\",\"ASC\"]"
// @Param filter query string false "Filter"
// @Success 200 {array} models.RecruitmentRole
// @Header 200 {string} Content-Range "recruitmentroles 0-24/50"
// @Security BearerAuth
// @Router /recruitmentroles [get]
func GetRecruitmentRoles(w http.ResponseWriter, r *http.Request) {
	params, _ := utils.ParseListParams(r.URL.Query())

	var items []models.RecruitmentRole
	query := db.DB.Model(&models.RecruitmentRole{})

	for k, v := range params.Filter {
		query = query.Where(k+" = ?", v)
	}
	if len(params.Sort) == 2 {
		query = query.Order(params.Sort[0] + " " + params.Sort[1])
	}

	start, end := params.Range[0], params.Range[1]
	limit := end - start + 1

	var total int64
	db.DB.Model(&models.RecruitmentRole{}).Count(&total)

	query = query.Offset(start).Limit(limit)
	query.Preload("Team").Preload("Recruitment").Find(&items)

	w.Header().Set("Access-Control-Expose-Headers", "Content-Range")
	w.Header().Set("Content-Range", fmt.Sprintf("recruitmentroles %d-%d/%d", start, end, total))
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(items)
}

// GetRecruitmentRoleByID returns a single recruitment role by ID.
// @Summary Get recruitment role by ID
// @Tags recruitment
// @Produce json
// @Param id path int true "RecruitmentRole ID"
// @Success 200 {object} models.RecruitmentRole
// @Failure 404 {string} string "recruitment role not found"
// @Security BearerAuth
// @Router /recruitmentroles/{id} [get]
func GetRecruitmentRoleByID(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]

	var item models.RecruitmentRole
	if err := db.DB.Preload("Team").Preload("Recruitment").First(&item, id).Error; err != nil {
		http.Error(w, "recruitment role not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(item)
}

// CreateRecruitmentRole creates a new recruitment role.
// @Summary Create recruitment role
// @Tags recruitment
// @Accept json
// @Produce json
// @Param body body controllers.recruitmentRolePayload true "Recruitment role data"
// @Success 201 {object} models.RecruitmentRole
// @Failure 400 {string} string "Invalid body"
// @Failure 500 {string} string "Create failed"
// @Security BearerAuth
// @Router /recruitmentroles [post]
func CreateRecruitmentRole(w http.ResponseWriter, r *http.Request) {
	var payload recruitmentRolePayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		log.Println(err)
		http.Error(w, "Invalid body", http.StatusBadRequest)
		return
	}

	recruitmentID := uint(0)
	if payload.RecruitmentID != nil {
		recruitmentID = *payload.RecruitmentID
	}

	item := models.RecruitmentRole{
		EventroRoleID: trimStringPointer(payload.EventroRoleID),
		RecruitmentID: recruitmentID,
		TeamID:        payload.TeamID,
		Name:          strings.TrimSpace(payload.Name),
		Description:   strings.TrimSpace(payload.Description),
	}

	if err := createWithAudit(r, "recruitmentroles", &item, func(tx *gorm.DB) error {
		return tx.Create(&item).Error
	}, func(tx *gorm.DB) error {
		return tx.Preload("Team").Preload("Recruitment").First(&item, item.ID).Error
	}); err != nil {
		http.Error(w, "Create failed", http.StatusInternalServerError)
		return
	}

	writeCreatedJSONResponse(w, item)
}

// UpdateRecruitmentRole updates a recruitment role by ID.
// @Summary Update recruitment role
// @Tags recruitment
// @Accept json
// @Produce json
// @Param id path int true "RecruitmentRole ID"
// @Param body body controllers.recruitmentRolePayload true "Updated recruitment role"
// @Success 200 {object} models.RecruitmentRole
// @Failure 400 {string} string "Invalid body"
// @Failure 404 {string} string "Not found"
// @Failure 500 {string} string "Update failed"
// @Security BearerAuth
// @Router /recruitmentroles/{id} [put]
func UpdateRecruitmentRole(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]

	var item models.RecruitmentRole
	if err := db.DB.First(&item, id).Error; err != nil {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}

	var payload recruitmentRolePayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "Invalid data", http.StatusBadRequest)
		return
	}

	updates := map[string]interface{}{
		"eventro_role_id": trimStringPointer(payload.EventroRoleID),
		"team_id":         payload.TeamID,
		"name":            strings.TrimSpace(payload.Name),
		"description":     strings.TrimSpace(payload.Description),
	}
	if payload.RecruitmentID != nil {
		updates["recruitment_id"] = *payload.RecruitmentID
	}

	before := item
	if err := updateWithAudit(r, "recruitmentroles", id, before, &item, func(tx *gorm.DB) error {
		return tx.Model(&item).Updates(updates).Error
	}, func(tx *gorm.DB) error {
		return tx.Preload("Team").Preload("Recruitment").First(&item, id).Error
	}); err != nil {
		http.Error(w, "Update failed", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(item)
}

// DeleteRecruitmentRole deletes a recruitment role by ID.
// @Summary Delete recruitment role
// @Tags recruitment
// @Produce json
// @Param id path int true "RecruitmentRole ID"
// @Success 200 {string} string "Deleted"
// @Failure 404 {string} string "recruitment role not found"
// @Security BearerAuth
// @Router /recruitmentroles/{id} [delete]
func DeleteRecruitmentRole(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	writeDeleteResponseWithAudit[models.RecruitmentRole](w, r, "recruitmentroles", id, "recruitment role not found", func(tx *gorm.DB) *gorm.DB {
		return tx.Preload("Team").Preload("Recruitment")
	})
}
