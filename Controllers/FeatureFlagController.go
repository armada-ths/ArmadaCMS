package controllers

import (
	"ArmadaCMS/main/db"
	"ArmadaCMS/main/models"
	"ArmadaCMS/main/utils"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
	"gorm.io/gorm"
)

func GetFeatureFlags(w http.ResponseWriter, r *http.Request) {
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
	if err := createWithAudit(r, "featureflags", &item, func(tx *gorm.DB) error {
		return tx.Create(&item).Error
	}, nil); err != nil {
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

	before := item
	if err := updateWithAudit(r, "featureflags", id, before, &item, func(tx *gorm.DB) error {
		return tx.Model(&item).Updates(updatePayload).Error
	}, func(tx *gorm.DB) error {
		return tx.First(&item, id).Error
	}); err != nil {
		http.Error(w, "Update failed", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(item)
}

func DeleteFeatureFlag(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	writeDeleteResponseWithAudit[models.FeatureFlag](w, r, "featureflags", id, "feature flag not found", nil)
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
		if len(updates) > 0 {
			dbConn.Model(&existing).Updates(updates)
		}
	}
	return nil
}
