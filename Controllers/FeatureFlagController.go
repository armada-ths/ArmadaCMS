package controllers

import (
	"ArmadaCMS/main/db"
	"ArmadaCMS/main/models"
	"ArmadaCMS/main/utils"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"gorm.io/gorm"
)

func GetFeatureFlags(w http.ResponseWriter, r *http.Request) {
	if err := applyExhibitorSignupAuto(db.DB); err != nil {
		http.Error(w, "Failed to apply exhibitor signup rule", http.StatusInternalServerError)
		return
	}

	query := db.DB.Model(&models.FeatureFlag{})
	params, _ := utils.ParseListParams(r.URL.Query())
	usePagination := r.URL.Query().Get("range") != "" || r.URL.Query().Get("sort") != "" || r.URL.Query().Get("filter") != ""

	for k, v := range params.Filter {
		query = query.Where(utils.ToSnakeCase(k)+" = ?", v)
	}
	if len(params.Sort) == 2 {
		query = query.Order(utils.ToSnakeCase(params.Sort[0]) + " " + params.Sort[1])
	}

	var total int64
	db.DB.Model(&models.FeatureFlag{}).Count(&total)

	var items []models.FeatureFlag
	if usePagination {
		start, end := params.Range[0], params.Range[1]
		query = query.Offset(start).Limit(end - start + 1)
		query.Find(&items)

		w.Header().Set("Access-Control-Expose-Headers", "Content-Range")
		w.Header().Set("Content-Range", fmt.Sprintf("featureflags %d-%d/%d", start, end, total))
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(items)
		return
	}

	query.Find(&items)

	w.Header().Set("Access-Control-Expose-Headers", "Content-Range")
	if total == 0 {
		w.Header().Set("Content-Range", "featureflags 0-0/0")
	} else {
		w.Header().Set("Content-Range", fmt.Sprintf("featureflags %d-%d/%d", 0, total-1, total))
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(items)
}

func GetFeatureFlagByID(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	if err := applyExhibitorSignupAuto(db.DB); err != nil {
		http.Error(w, "Failed to apply exhibitor signup rule", http.StatusInternalServerError)
		return
	}

	var item models.FeatureFlag
	if err := db.DB.First(&item, id).Error; err != nil {
		http.Error(w, "feature flag not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(item)
}

func CreateFeatureFlag(w http.ResponseWriter, r *http.Request) {
	var item models.FeatureFlag
	if err := json.NewDecoder(r.Body).Decode(&item); err != nil {
		http.Error(w, "Invalid body", http.StatusBadRequest)
		return
	}
	if item.AutoValue == nil {
		item.AutoValue = boolPtr(item.Enabled)
	}
	if err := db.DB.Create(&item).Error; err != nil {
		http.Error(w, "Create failed", http.StatusInternalServerError)
		return
	}
	writeCreatedJSONResponse(w, item)
}

func UpdateFeatureFlag(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]

	var item models.FeatureFlag
	if err := db.DB.First(&item, id).Error; err != nil {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}

	var updates models.FeatureFlag
	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		http.Error(w, "Invalid data", http.StatusBadRequest)
		return
	}

	updatePayload := map[string]any{
		"description": updates.Description,
		"enabled":     updates.Enabled,
	}

	db.DB.Model(&item).Updates(updatePayload)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(item)
}

func DeleteFeatureFlag(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	writeDeleteResponse(w, db.DB.Delete(&models.FeatureFlag{}, id), "feature flag not found")
}

func SeedFeatureFlags(dbConn *gorm.DB) error {
	for _, flag := range models.DefaultFeatureFlags {
		var existing models.FeatureFlag
		err := dbConn.Where("key = ?", flag.Key).First(&existing).Error
		if err != nil {
			if err == gorm.ErrRecordNotFound {
				if err := dbConn.Create(&flag).Error; err != nil {
					return err
				}
				continue
			}
			return err
		}

		updates := map[string]any{}
		if existing.Description != flag.Description {
			updates["description"] = flag.Description
		}
		if existing.AutoValue == nil {
			updates["auto_value"] = flag.AutoValue
		}
		if len(updates) > 0 {
			dbConn.Model(&existing).Updates(updates)
		}
	}
	return nil
}

func boolPtr(value bool) *bool {
	return &value
}

func applyExhibitorSignupAuto(dbConn *gorm.DB) error {
	var flag models.FeatureFlag
	if err := dbConn.Where("key = ?", "EXHIBITOR_SIGNUP").First(&flag).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil
		}
		return err
	}

	open, err := computeExhibitorSignupOpen(dbConn)
	if err != nil {
		return err
	}

	if flag.AutoValue == nil || open != *flag.AutoValue {
		now := time.Now().UTC()
		updates := map[string]any{
			"enabled":         open,
			"auto_value":      open,
			"auto_updated_at": &now,
		}
		if err := dbConn.Model(&flag).Updates(updates).Error; err != nil {
			return err
		}
		flag.Enabled = open
		flag.AutoValue = &open
		flag.AutoUpdatedAt = &now
	}

	return nil
}

func computeExhibitorSignupOpen(dbConn *gorm.DB) (bool, error) {
	loc, err := time.LoadLocation("Europe/Stockholm")
	if err != nil {
		return false, err
	}

	var config models.FairDateConfig
	result := dbConn.Order("id desc").First(&config)

	var fairDate models.FairDate
	if result.Error != nil {
		fairDate = defaultFairDate()
	} else {
		fairDate = config.ToFairDate()
	}

	if len(fairDate.Fair.Days) == 0 {
		return false, nil
	}

	irStart, err := time.ParseInLocation("2006-01-02", fairDate.IR.Start, loc)
	if err != nil {
		return false, nil
	}
	firstFairDay, err := time.ParseInLocation("2006-01-02", fairDate.Fair.Days[0], loc)
	if err != nil {
		return false, nil
	}

	now := time.Now().In(loc)
	return !now.Before(irStart) && now.Before(firstFairDay), nil
}
