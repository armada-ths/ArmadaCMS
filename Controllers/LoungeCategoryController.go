package controllers

import (
	"ArmadaCMS/main/audit"
	"ArmadaCMS/main/db"
	"ArmadaCMS/main/models"
	"ArmadaCMS/main/utils"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/gorilla/mux"
	"gorm.io/gorm"
)

type loungeCatalogue struct {
	Revision   int64                   `json:"revision"`
	Categories []models.LoungeCategory `json:"categories"`
	Items      []models.LoungeItem     `json:"items"`
}

type loungeCategoryMutation struct {
	Name   string  `json:"name"`
	NameSv *string `json:"nameSv"`
}

func loungePathID(w http.ResponseWriter, r *http.Request) (uint64, bool) {
	id, err := strconv.ParseUint(mux.Vars(r)["id"], 10, 64)
	if err != nil || id == 0 || id > 9007199254740991 {
		http.Error(w, "Invalid resource ID", http.StatusBadRequest)
		return 0, false
	}
	return id, true
}

func lockLoungeCatalogue(tx *gorm.DB) (int64, error) {
	var revision int64
	if err := tx.Raw("SELECT revision FROM lounge_catalogue_state WHERE id = 1 FOR UPDATE").Scan(&revision).Error; err != nil {
		return 0, err
	}
	return revision, tx.Exec("UPDATE lounge_catalogue_state SET revision = revision + 1 WHERE id = 1").Error
}

func readLoungeCatalogue(tx *gorm.DB) (loungeCatalogue, error) {
	catalogue := loungeCatalogue{Categories: []models.LoungeCategory{}, Items: []models.LoungeItem{}}
	if err := tx.Raw("SELECT revision FROM lounge_catalogue_state WHERE id = 1").Scan(&catalogue.Revision).Error; err != nil {
		return catalogue, err
	}
	if err := tx.Order("sort_order, id").Find(&catalogue.Categories).Error; err != nil {
		return catalogue, err
	}
	err := tx.Order("sort_order, id").Find(&catalogue.Items).Error
	return catalogue, err
}

// GetLoungeCatalogue returns a consistent catalogue and revision.
// @Summary Get lounge catalogue
// @Tags lounge
// @Produce json
// @Success 200 {object} loungeCatalogue
// @Router /lounge-catalogue [get]
func GetLoungeCatalogue(w http.ResponseWriter, r *http.Request) {
	var catalogue loungeCatalogue
	err := db.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec("SET TRANSACTION ISOLATION LEVEL REPEATABLE READ").Error; err != nil {
			return err
		}
		var err error
		catalogue, err = readLoungeCatalogue(tx)
		return err
	})
	if err != nil {
		loungeError(w, err)
		return
	}
	writeJSONResponse(w, http.StatusOK, catalogue)
}

type loungeReorder struct {
	Revision   int64 `json:"revision"`
	Categories []struct {
		ID      uint   `json:"id"`
		ItemIDs []uint `json:"itemIds"`
	} `json:"categories"`
}

func validateLoungeReorder(input loungeReorder, current loungeCatalogue) error {
	if input.Revision != current.Revision {
		return errors.New("catalogue changed; reload before reordering")
	}
	categories, items := map[uint]bool{}, map[uint]bool{}
	for _, c := range current.Categories {
		categories[c.ID] = true
	}
	for _, item := range current.Items {
		items[item.ID] = true
	}
	for _, c := range input.Categories {
		if !categories[c.ID] {
			return errors.New("invalid or duplicate category")
		}
		delete(categories, c.ID)
		for _, item := range c.ItemIDs {
			if !items[item] {
				return errors.New("invalid or duplicate item")
			}
			delete(items, item)
		}
	}
	if len(categories) != 0 || len(items) != 0 {
		return errors.New("reorder must contain the complete catalogue")
	}
	return nil
}

// ReorderLoungeCatalogue atomically reorders and moves categories/items.
// @Summary Reorder lounge catalogue
// @Tags lounge
// @Accept json
// @Produce json
// @Param body body loungeReorder true "Complete ordering and revision"
// @Success 200 {object} loungeCatalogue
// @Failure 409 {string} string "Catalogue changed"
// @Security BearerAuth
// @Router /lounge-catalogue [put]
func ReorderLoungeCatalogue(w http.ResponseWriter, r *http.Request) {
	var input loungeReorder
	if !decodeLoungeJSON(w, r, &input) {
		return
	}
	var after loungeCatalogue
	conflict := false
	err := updateWithGroupedAudit(r, "lounge", 1, input.Revision, &after, func(tx *gorm.DB, child *http.Request) error {
		revision, err := lockLoungeCatalogue(tx)
		if err != nil {
			return err
		}
		current, err := readLoungeCatalogue(tx)
		if err != nil {
			return err
		}
		current.Revision = revision
		if err := validateLoungeReorder(input, current); err != nil {
			conflict = true
			return err
		}
		for position, c := range input.Categories {
			var old models.LoungeCategory
			for _, category := range current.Categories {
				if category.ID == c.ID {
					old = category
					break
				}
			}
			newCategory := old
			newCategory.SortOrder = int64(position)
			if err := tx.Model(&models.LoungeCategory{}).Where("id = ?", c.ID).Update("sort_order", position).Error; err != nil {
				return err
			}
			if err := audit.LogUpdate(tx, child, "lounge-categories", c.ID, old, newCategory); err != nil {
				return err
			}
			for index, itemID := range c.ItemIDs {
				var oldItem models.LoungeItem
				for _, item := range current.Items {
					if item.ID == itemID {
						oldItem = item
						break
					}
				}
				newItem := oldItem
				newItem.CategoryID, newItem.SortOrder = c.ID, int64(index)
				if err := tx.Model(&models.LoungeItem{}).Where("id = ?", itemID).Updates(map[string]any{"category_id": c.ID, "sort_order": index}).Error; err != nil {
					return err
				}
				if err := audit.LogUpdate(tx, child, "lounge-items", itemID, oldItem, newItem); err != nil {
					return err
				}
			}
		}
		after, err = readLoungeCatalogue(tx)
		return err
	}, nil)
	if conflict {
		http.Error(w, "Catalogue changed or invalid ordering; reload and retry", http.StatusConflict)
		return
	}
	if err != nil {
		loungeError(w, err)
		return
	}
	writeJSONResponse(w, http.StatusOK, after)
}

// GetLoungeCategories lists localized categories.
// @Summary List lounge categories
// @Tags lounge
// @Produce json
// @Success 200 {array} models.LoungeCategory
// @Router /lounge-categories [get]
func GetLoungeCategories(w http.ResponseWriter, r *http.Request) {
	items := []models.LoungeCategory{}
	query := db.DB.Order("sort_order, id")
	if raw := r.URL.Query().Get("filter"); raw != "" {
		var filter struct {
			IDs []uint `json:"id"`
		}
		if err := json.Unmarshal([]byte(raw), &filter); err != nil {
			http.Error(w, "Invalid category filter", 400)
			return
		}
		if filter.IDs != nil {
			query = query.Where("id IN ?", filter.IDs)
		}
	}
	params, _ := utils.ParseListParams(r.URL.Query())
	var total int64
	if err := query.Model(&models.LoungeCategory{}).Count(&total).Error; err != nil {
		loungeError(w, err)
		return
	}
	if r.URL.Query().Has("range") {
		query = query.Offset(params.Range[0]).Limit(params.Range[1] - params.Range[0] + 1)
	}
	if err := query.Find(&items).Error; err != nil {
		loungeError(w, err)
		return
	}
	start := 0
	if r.URL.Query().Has("range") {
		start = params.Range[0]
	}
	w.Header().Set("Access-Control-Expose-Headers", "Content-Range")
	w.Header().Set("Content-Range", fmt.Sprintf("lounge-categories %d-%d/%d", start, start+len(items)-1, total))
	writeJSONResponse(w, http.StatusOK, items)
}

// GetLoungeCategory returns one category.
// @Summary Get lounge category
// @Tags lounge
// @Param id path int true "Category ID"
// @Success 200 {object} models.LoungeCategory
// @Router /lounge-categories/{id} [get]
func GetLoungeCategory(w http.ResponseWriter, r *http.Request) {
	id, ok := loungePathID(w, r)
	if !ok {
		return
	}
	var category models.LoungeCategory
	if err := db.DB.First(&category, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			http.Error(w, "Category not found", 404)
		} else {
			loungeError(w, err)
		}
		return
	}
	writeJSONResponse(w, 200, category)
}

// CreateLoungeCategory creates a category at the end of the catalogue.
// @Summary Create lounge category
// @Tags lounge
// @Accept json
// @Param body body loungeCategoryMutation true "Localized category"
// @Success 201 {object} models.LoungeCategory
// @Security BearerAuth
// @Router /lounge-categories [post]
func CreateLoungeCategory(w http.ResponseWriter, r *http.Request) { saveLoungeCategory(w, r) }

// UpdateLoungeCategory updates a category name without changing its position.
// @Summary Update lounge category
// @Tags lounge
// @Accept json
// @Param id path int true "Category ID"
// @Param body body loungeCategoryMutation true "Localized category"
// @Success 200 {object} models.LoungeCategory
// @Security BearerAuth
// @Router /lounge-categories/{id} [put]
func UpdateLoungeCategory(w http.ResponseWriter, r *http.Request) { saveLoungeCategory(w, r) }

func saveLoungeCategory(w http.ResponseWriter, r *http.Request) {
	var input loungeCategoryMutation
	if !decodeLoungeJSON(w, r, &input) {
		return
	}
	input.Name = strings.TrimSpace(input.Name)
	NormalizeOptionalStringPointers(&input.NameSv)
	if input.Name == "" {
		http.Error(w, "Name is required", 400)
		return
	}
	category := models.LoungeCategory{Name: input.Name, NameSv: input.NameSv}
	persist := func(tx *gorm.DB) error {
		if _, err := lockLoungeCatalogue(tx); err != nil {
			return err
		}
		if category.ID == 0 {
			if err := tx.Model(&models.LoungeCategory{}).Select("COALESCE(MAX(sort_order), -1) + 1").Scan(&category.SortOrder).Error; err != nil {
				return err
			}
			return tx.Create(&category).Error
		}
		return tx.Model(&category).Updates(map[string]any{"name": input.Name, "name_sv": input.NameSv}).Error
	}
	if r.Method == "POST" {
		if err := createWithAudit(r, "lounge-categories", &category, persist, nil); err != nil {
			loungeError(w, err)
			return
		}
		writeCreatedJSONResponse(w, category)
	} else {
		id, ok := loungePathID(w, r)
		if !ok {
			return
		}
		var before models.LoungeCategory
		if err := db.DB.First(&before, id).Error; err != nil {
			loungeError(w, err)
			return
		}
		category.ID, category.SortOrder = before.ID, before.SortOrder
		if err := updateWithAudit(r, "lounge-categories", category.ID, before, &category, persist, nil); err != nil {
			loungeError(w, err)
			return
		}
		writeJSONResponse(w, 200, category)
	}
}

// DeleteLoungeCategory deletes a category and its catalogue items, preserving orders.
// @Summary Delete category and items
// @Tags lounge
// @Param id path int true "Category ID"
// @Success 204 {string} string "Deleted"
// @Security BearerAuth
// @Router /lounge-categories/{id} [delete]
func DeleteLoungeCategory(w http.ResponseWriter, r *http.Request) {
	id, ok := loungePathID(w, r)
	if !ok {
		return
	}
	writeGroupedDeleteResponseWithAudit[models.LoungeCategory](w, r, "lounge-categories", strconv.FormatUint(id, 10), "Category not found", nil,
		func(tx *gorm.DB, category *models.LoungeCategory, child *http.Request) error {
			if _, err := lockLoungeCatalogue(tx); err != nil {
				return err
			}
			var items []models.LoungeItem
			if err := tx.Where("category_id = ?", category.ID).Find(&items).Error; err != nil {
				return err
			}
			for _, item := range items {
				if err := audit.LogDelete(tx, child, "lounge-items", item.ID, item); err != nil {
					return err
				}
			}
			return tx.Where("category_id = ?", category.ID).Delete(&models.LoungeItem{}).Error
		}, nil)
}

func decodeLoungeJSON(w http.ResponseWriter, r *http.Request, target any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 256<<10)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		http.Error(w, "Invalid JSON", 400)
		return false
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		http.Error(w, "Unexpected trailing JSON", http.StatusBadRequest)
		return false
	}
	return true
}

func loungeError(w http.ResponseWriter, err error) {
	log.Printf("lounge operation failed: %v", err)
	http.Error(w, "Lounge operation failed", http.StatusInternalServerError)
}
