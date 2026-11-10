package controllers

import (
	"ArmadaCMS/main/models"
	"encoding/json"
	"maps"
	"net/http"
	"slices"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var eventroCategoryResources = []string{"industries", "employments", "programs"}

func filterBlockedEventroCategories(tx *gorm.DB, ex *models.Exhibitor) error {
	var blocks []models.EventroSyncBlock
	if err := tx.Select("resource", "eventro_key").Where("resource IN ?", eventroCategoryResources).Find(&blocks).Error; err != nil {
		return err
	}
	keys := map[string]map[string]bool{}
	for _, block := range blocks {
		if keys[block.Resource] == nil {
			keys[block.Resource] = map[string]bool{}
		}
		keys[block.Resource][block.EventroKey] = true
	}
	ex.Industries = slices.DeleteFunc(ex.Industries, func(item models.Industry) bool {
		return keys["industries"][item.Name]
	})
	ex.Employments = slices.DeleteFunc(ex.Employments, func(item models.Employment) bool {
		return keys["employments"][item.Name]
	})
	ex.Programs = slices.DeleteFunc(ex.Programs, func(item models.Program) bool {
		return keys["programs"][item.Name]
	})
	return nil
}

// Remove the deleted label from the last usable source value so unlinking the
// category does not give the CMS ownership of unrelated categories.
func forgetDeletedEventroCategory(tx *gorm.DB, r *http.Request, resource, key string) error {
	var states []models.EventroSyncState
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("resource = ? AND jsonb_exists(source, ?)", "exhibitors", resource).Find(&states).Error; err != nil {
		return err
	}
	for _, state := range states {
		var labels []string
		if err := json.Unmarshal(state.Source[resource], &labels); err != nil {
			return err
		}
		originalCount := len(labels)
		labels = slices.DeleteFunc(labels, func(label string) bool {
			return label == key
		})
		if len(labels) == originalCount {
			continue
		}
		before := state
		state.Source = maps.Clone(state.Source)
		state.Source[resource], _ = json.Marshal(labels)
		if err := saveEventroSyncStateWithAudit(tx, r, before, &state); err != nil {
			return err
		}
	}
	return nil
}
