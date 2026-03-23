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
)

type recruitmentRolePayload struct {
	EventroRoleID *string `json:"eventroRoleId"`
	RecruitmentID *uint   `json:"recruitmentId"`
	TeamID        *uint   `json:"team_id"`
	Name          string  `json:"name"`
	Description   string  `json:"description"`
}

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
	query.Preload("Team").Find(&items)

	w.Header().Set("Access-Control-Expose-Headers", "Content-Range")
	w.Header().Set("Content-Range", fmt.Sprintf("recruitmentroles %d-%d/%d", start, end, total))
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(items)
}

func GetRecruitmentRoleByID(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]

	var item models.RecruitmentRole
	if err := db.DB.Preload("Team").First(&item, id).Error; err != nil {
		http.Error(w, "recruitment role not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(item)
}

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

	if err := db.DB.Create(&item).Error; err != nil {
		http.Error(w, "Create failed", http.StatusInternalServerError)
		return
	}

	db.DB.Preload("Team").First(&item, item.ID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(item)
}

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

	db.DB.Model(&item).Updates(updates)
	db.DB.Preload("Team").First(&item, id)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(item)
}

func DeleteRecruitmentRole(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	if err := db.DB.Delete(&models.RecruitmentRole{}, id).Error; err != nil {
		http.Error(w, "Delete failed", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
