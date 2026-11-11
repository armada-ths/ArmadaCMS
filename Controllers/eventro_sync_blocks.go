package controllers

import (
	"ArmadaCMS/main/audit"
	"ArmadaCMS/main/auth"
	"ArmadaCMS/main/db"
	"ArmadaCMS/main/models"
	"errors"
	"net/http"
	"slices"
	"strconv"
	"strings"

	"github.com/gorilla/mux"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var errEventroSyncSkipped = errors.New("skipped Eventro record")

// Acquire before reading the block or locking a CMS row. Recruitment imports and
// grouped deletes share a lock because deleting a period also removes its roles.
func lockEventroSyncDomain(tx *gorm.DB, resource string) error {
	switch {
	case resource == "recruitmentperiods", resource == "recruitmentroles":
		resource = "recruitment"
	case slices.Contains(eventroCategoryResources, resource):
		resource = "exhibitors"
	}
	return tx.Exec("SELECT pg_advisory_xact_lock(hashtextextended(?, 0))", "eventro:sync-domain:"+resource).Error
}

func beginEventroImport(tx *gorm.DB, resource, key string) error {
	if err := lockEventroSyncDomain(tx, resource); err != nil {
		return err
	}
	var id uint
	result := tx.Model(&models.EventroSyncBlock{}).Select("id").
		Where("resource = ? AND eventro_key = ?", resource, key).Limit(1).Scan(&id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 0 {
		return errEventroSyncSkipped
	}
	return nil
}

func blockDeletedEventroRecord(tx *gorm.DB, r *http.Request, resource string, entity any) error {
	var key *string
	var label string
	switch row := entity.(type) {
	case *models.Exhibitor:
		key, label = row.EventroID, row.Name
	case *models.Event:
		key, label = &row.EventroID, row.Name
	case *models.Profile:
		key, label = row.EventroKey, row.Name
	case *models.FairDateConfig:
		key, label = row.EventroID, row.Description
	case *models.RecruitmentPeriod:
		key, label = row.EventroID, row.Name
	case *models.RecruitmentRole:
		key, label = row.EventroRoleID, row.Name
	case *models.Industry:
		label = row.Name
	case *models.Employment:
		label = row.Name
	case *models.Program:
		label = row.Name
	}
	isCategory := slices.Contains(eventroCategoryResources, resource)
	if isCategory {
		key = &label
	}
	if key == nil || strings.TrimSpace(*key) == "" {
		return nil
	}
	block := models.EventroSyncBlock{Resource: resource, EventroKey: *key, Label: label}
	result := tx.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "resource"}, {Name: "eventro_key"}},
		DoNothing: true,
	}).Create(&block)
	if result.Error != nil || result.RowsAffected == 0 {
		return result.Error
	}
	if err := audit.LogCreate(tx, r, "eventro_sync_blocks", block.ID, &block); err != nil {
		return err
	}
	if isCategory {
		return forgetDeletedEventroCategory(tx, r, resource, *key)
	}
	return nil
}

// ListEventroSyncBlocks lists source records excluded from synchronization.
// @Summary List blocked Eventro records for a resource
// @Tags eventro
// @Produce json
// @Param resource query string true "CMS resource"
// @Success 200 {array} models.EventroSyncBlock
// @Failure 400 {string} string "Invalid resource"
// @Failure 403 {string} string "Insufficient resource permissions"
// @Security BearerAuth
// @Router /eventrosyncblocks [get]
func ListEventroSyncBlocks(w http.ResponseWriter, r *http.Request) {
	resource := r.URL.Query().Get("resource")
	if _, valid := eventroResourceSpecs[resource]; !valid && !slices.Contains(eventroCategoryResources, resource) {
		http.Error(w, "Invalid resource", http.StatusBadRequest)
		return
	}
	if !auth.HasPermission(auth.GetPermissionsFromContext(r), resource+".delete") {
		http.Error(w, "Insufficient resource permissions", http.StatusForbidden)
		return
	}
	blocks := []models.EventroSyncBlock{}
	if err := db.DB.Where("resource = ?", resource).Order("deleted_at DESC, id DESC").Find(&blocks).Error; err != nil {
		http.Error(w, "Failed to load blocked records", http.StatusInternalServerError)
		return
	}
	writeJSONResponse(w, http.StatusOK, blocks)
}

// DeleteEventroSyncBlock allows the next Eventro sync to import the record.
// @Summary Allow a blocked Eventro record to sync again
// @Tags eventro
// @Param id path int true "Block ID"
// @Success 204
// @Failure 400 {string} string "Invalid block ID"
// @Failure 403 {string} string "Insufficient resource permissions"
// @Failure 404 {string} string "Block not found"
// @Security BearerAuth
// @Router /eventrosyncblocks/{id} [delete]
func DeleteEventroSyncBlock(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	blockID, err := strconv.ParseInt(id, 10, 64)
	if err != nil || blockID <= 0 {
		http.Error(w, "Invalid block ID", http.StatusBadRequest)
		return
	}
	var block models.EventroSyncBlock
	if err := db.DB.First(&block, blockID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			http.Error(w, "Block not found", http.StatusNotFound)
		} else {
			http.Error(w, "Failed to load blocked record", http.StatusInternalServerError)
		}
		return
	}
	if !auth.HasPermission(auth.GetPermissionsFromContext(r), block.Resource+".delete") {
		http.Error(w, "Insufficient resource permissions", http.StatusForbidden)
		return
	}
	writeDeleteResponseWithAudit(w, r, "eventro_sync_blocks", id, "Block not found", nil,
		func(tx *gorm.DB, current *models.EventroSyncBlock) error {
			return lockEventroSyncDomain(tx, current.Resource)
		})
}
