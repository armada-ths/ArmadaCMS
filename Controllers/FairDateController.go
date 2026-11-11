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
// @Param filter query string false "Filter object; q searches text, other public scalar field names or database column names match exact values"
// @Success 200 {array} models.FairDateConfig
// @Header 200 {string} Content-Range "fairdates 0-24/5"
// @Security BearerAuth
// @Router /fairdates [get]
func GetFairDateConfigs(w http.ResponseWriter, r *http.Request) {
	params, _ := utils.ParseListParams(r.URL.Query())

	var items []models.FairDateConfig
	query := db.DB.Model(&models.FairDateConfig{})

	query = utils.ApplySearch(query, params.Filter["q"], "id", "description", "fair_days", "description_sv")
	var filterErr error
	query, filterErr = utils.ApplyColumnFilters(query, params.Filter)
	if filterErr != nil {
		http.Error(w, filterErr.Error(), http.StatusBadRequest)
		return
	}

	if len(params.Sort) == 2 {
		query = query.Order(utils.ToSnakeCase(params.Sort[0]) + " " + params.Sort[1])
	}

	start, end := params.Range[0], params.Range[1]

	var total int64
	if err := query.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		http.Error(w, "List failed", http.StatusInternalServerError)
		return
	}

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

	if minutes := item.LoungeClosingLeadMinutes(); minutes < 0 || minutes > models.MaxLoungeOrderCloseMinutes {
		http.Error(w, "Lounge order closing lead must be between 0 and 1440 minutes", http.StatusBadRequest)
		return
	}

	if item.DayOneOpensAt == "" {
		item.DayOneOpensAt = "10:00"
	}
	if item.DayOneClosesAt == "" {
		item.DayOneClosesAt = "16:00"
	}
	if item.DayTwoOpensAt == "" {
		item.DayTwoOpensAt = "10:00"
	}
	if item.DayTwoClosesAt == "" {
		item.DayTwoClosesAt = "15:00"
	}
	if !validFairOpeningHours(w, item) {
		return
	}

	NormalizeOptionalStringPointers(&item.DescriptionSv)
	item.EventroID = nil

	if err := createWithAudit(r, "fairdates", &item, func(tx *gorm.DB) error {
		return tx.Create(&item).Error
	}, nil, "dates"); err != nil {
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

	var rawUpdates map[string]any
	if err := json.NewDecoder(r.Body).Decode(&rawUpdates); err != nil {
		http.Error(w, "Invalid data", http.StatusBadRequest)
		return
	}

	updateMap := BuildNormalizedSnakeCaseUpdateMap(
		rawUpdates,
		map[string]struct{}{
			"descriptionSv": {},
		},
		map[string]struct{}{
			"id":         {},
			"eventroId":  {},
			"eventro_id": {},
		},
	)

	if raw, exists := updateMap["lounge_order_close_minutes"]; exists {
		minutes, ok := raw.(float64)
		if !ok || minutes < 0 || minutes > models.MaxLoungeOrderCloseMinutes || minutes != float64(int(minutes)) {
			http.Error(w, "Lounge order closing lead must be a whole number between 0 and 1440 minutes", http.StatusBadRequest)
			return
		}
		updateMap["lounge_order_close_minutes"] = int(minutes)
	}

	next := item
	for column, target := range map[string]*string{
		"day_one_opens_at":  &next.DayOneOpensAt,
		"day_one_closes_at": &next.DayOneClosesAt,
		"day_two_opens_at":  &next.DayTwoOpensAt,
		"day_two_closes_at": &next.DayTwoClosesAt,
	} {
		if raw, exists := updateMap[column]; exists {
			value, _ := raw.(string)
			*target = value
		}
	}
	if !validFairOpeningHours(w, next) {
		return
	}

	before := item
	if err := updateWithAudit(r, "fairdates", id, before, &item, func(tx *gorm.DB) error {
		return tx.Model(&item).Updates(updateMap).Error
	}, func(tx *gorm.DB) error {
		return tx.First(&item, id).Error
	}, "dates"); err != nil {
		http.Error(w, "Update failed", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(item)
}

func validFairOpeningHours(w http.ResponseWriter, item models.FairDateConfig) bool {
	for _, hours := range item.OpeningHours() {
		if !hours.Valid() {
			http.Error(w, "Enter opening and closing times as HH:MM with closing after opening", http.StatusBadRequest)
			return false
		}
	}
	return true
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
	writeDeleteResponseWithAudit[models.FairDateConfig](w, r, "fairdates", id, "fair date config not found", nil, nil, "dates")
}
