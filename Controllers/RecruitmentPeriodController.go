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
	"time"

	"github.com/gorilla/mux"
	"gorm.io/gorm"
)

type recruitmentPeriodPayload struct {
	EventroID *string `json:"eventroId"`
	Name      string  `json:"name"`
	Link      string  `json:"link"`
	StartDate *string `json:"startDate"`
	EndDate   *string `json:"endDate"`
}

// GetRecruitmentPeriods returns a paginated list of recruitment periods.
// @Summary List recruitment periods
// @Tags recruitment
// @Produce json
// @Param range query string false "Pagination range, e.g. [0,24]"
// @Param sort query string false "Sort, e.g. [\"name\",\"ASC\"]"
// @Param filter query string false "Filter"
// @Success 200 {array} models.RecruitmentPeriod
// @Header 200 {string} Content-Range "recruitmentperiods 0-24/10"
// @Security BearerAuth
// @Router /recruitmentperiods [get]
func GetRecruitmentPeriods(w http.ResponseWriter, r *http.Request) {
	params, _ := utils.ParseListParams(r.URL.Query())

	var items []models.RecruitmentPeriod
	query := db.DB.Model(&models.RecruitmentPeriod{})

	for k, v := range params.Filter {
		query = query.Where(k+" = ?", v)
	}
	if len(params.Sort) == 2 {
		query = query.Order(params.Sort[0] + " " + params.Sort[1])
	}

	start, end := params.Range[0], params.Range[1]
	limit := end - start + 1

	var total int64
	db.DB.Model(&models.RecruitmentPeriod{}).Count(&total)

	query = query.Offset(start).Limit(limit)
	query.Preload("Roles").Preload("Roles.Team").Find(&items)

	w.Header().Set("Access-Control-Expose-Headers", "Content-Range")
	w.Header().Set("Content-Range", fmt.Sprintf("recruitmentperiods %d-%d/%d", start, end, total))
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(items)
}

// GetRecruitmentPeriodByID returns a single recruitment period by ID.
// @Summary Get recruitment period by ID
// @Tags recruitment
// @Produce json
// @Param id path int true "RecruitmentPeriod ID"
// @Success 200 {object} models.RecruitmentPeriod
// @Failure 404 {string} string "recruitment period not found"
// @Security BearerAuth
// @Router /recruitmentperiods/{id} [get]
func GetRecruitmentPeriodByID(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]

	var item models.RecruitmentPeriod
	if err := db.DB.Preload("Roles").Preload("Roles.Team").First(&item, id).Error; err != nil {
		http.Error(w, "recruitment period not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(item)
}

// CreateRecruitmentPeriod creates a new recruitment period.
// @Summary Create recruitment period
// @Tags recruitment
// @Accept json
// @Produce json
// @Param body body controllers.recruitmentPeriodPayload true "Recruitment period data"
// @Success 201 {object} models.RecruitmentPeriod
// @Failure 400 {string} string "Invalid body"
// @Failure 500 {string} string "Create failed"
// @Security BearerAuth
// @Router /recruitmentperiods [post]
func CreateRecruitmentPeriod(w http.ResponseWriter, r *http.Request) {
	var payload recruitmentPeriodPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		log.Println(err)
		http.Error(w, "Invalid body", http.StatusBadRequest)
		return
	}

	startDate, err := parseFlexibleDateTime(payload.StartDate)
	if err != nil {
		http.Error(w, "Invalid startDate", http.StatusBadRequest)
		return
	}

	endDate, err := parseFlexibleDateTime(payload.EndDate)
	if err != nil {
		http.Error(w, "Invalid endDate", http.StatusBadRequest)
		return
	}

	item := models.RecruitmentPeriod{
		EventroID: trimStringPointer(payload.EventroID),
		Name:      strings.TrimSpace(payload.Name),
		Link:      strings.TrimSpace(payload.Link),
		StartDate: startDate,
		EndDate:   endDate,
	}

	if err := createWithAudit(r, "recruitmentperiods", &item, func(tx *gorm.DB) error {
		return tx.Create(&item).Error
	}, func(tx *gorm.DB) error {
		return tx.Preload("Roles").Preload("Roles.Team").First(&item, item.ID).Error
	}); err != nil {
		http.Error(w, "Create failed", http.StatusInternalServerError)
		return
	}

	writeCreatedJSONResponse(w, item)
}

// UpdateRecruitmentPeriod updates a recruitment period by ID.
// @Summary Update recruitment period
// @Tags recruitment
// @Accept json
// @Produce json
// @Param id path int true "RecruitmentPeriod ID"
// @Param body body controllers.recruitmentPeriodPayload true "Updated recruitment period"
// @Success 200 {object} models.RecruitmentPeriod
// @Failure 400 {string} string "Invalid body"
// @Failure 404 {string} string "Not found"
// @Failure 500 {string} string "Update failed"
// @Security BearerAuth
// @Router /recruitmentperiods/{id} [put]
func UpdateRecruitmentPeriod(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]

	var item models.RecruitmentPeriod
	if err := db.DB.First(&item, id).Error; err != nil {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}

	var payload recruitmentPeriodPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "Invalid data", http.StatusBadRequest)
		return
	}

	startDate, err := parseFlexibleDateTime(payload.StartDate)
	if err != nil {
		http.Error(w, "Invalid startDate", http.StatusBadRequest)
		return
	}

	endDate, err := parseFlexibleDateTime(payload.EndDate)
	if err != nil {
		http.Error(w, "Invalid endDate", http.StatusBadRequest)
		return
	}

	updates := map[string]interface{}{
		"eventro_id": trimStringPointer(payload.EventroID),
		"name":       strings.TrimSpace(payload.Name),
		"link":       strings.TrimSpace(payload.Link),
		"start_date": startDate,
		"end_date":   endDate,
	}

	before := item
	if err := updateWithAudit(r, "recruitmentperiods", id, before, &item, func(tx *gorm.DB) error {
		return tx.Model(&item).Updates(updates).Error
	}, func(tx *gorm.DB) error {
		return tx.Preload("Roles").Preload("Roles.Team").First(&item, id).Error
	}); err != nil {
		http.Error(w, "Update failed", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(item)
}

// DeleteRecruitmentPeriod deletes a recruitment period by ID.
// @Summary Delete recruitment period
// @Tags recruitment
// @Produce json
// @Param id path int true "RecruitmentPeriod ID"
// @Success 200 {string} string "Deleted"
// @Failure 404 {string} string "recruitment period not found"
// @Security BearerAuth
// @Router /recruitmentperiods/{id} [delete]
func DeleteRecruitmentPeriod(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	writeDeleteResponseWithAudit[models.RecruitmentPeriod](w, r, "recruitmentperiods", id, "recruitment period not found", func(tx *gorm.DB) *gorm.DB {
		return tx.Preload("Roles").Preload("Roles.Team")
	})
}

func parseFlexibleDateTime(value *string) (*time.Time, error) {
	if value == nil {
		return nil, nil
	}

	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return nil, nil
	}

	layouts := []string{
		time.RFC3339,
		"2006-01-02T15:04",
		"2006-01-02",
	}

	location, err := time.LoadLocation("Europe/Stockholm")
	if err != nil {
		location = time.Local
	}

	for _, layout := range layouts {
		var parsed time.Time
		if layout == time.RFC3339 {
			parsed, err = time.Parse(layout, trimmed)
		} else {
			parsed, err = time.ParseInLocation(layout, trimmed, location)
		}

		if err != nil {
			continue
		}

		utc := parsed.UTC()
		return &utc, nil
	}

	return nil, fmt.Errorf("invalid datetime format")
}
