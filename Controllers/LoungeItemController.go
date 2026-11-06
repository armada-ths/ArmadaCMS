package controllers

import (
	"ArmadaCMS/main/db"
	"ArmadaCMS/main/models"
	"ArmadaCMS/main/utils"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/gorilla/mux"
	"gorm.io/gorm"
)

var loungeSortColumns = map[string]string{
	"id": "id", "name": "name", "sortOrder": "sort_order", "soldOut": "sold_out",
}

// GetLoungeItems lists the complete catalogue unless a range is specified.
// @Summary List lounge items
// @Tags lounge-items
// @Produce json
// @Param range query string false "Inclusive pagination range"
// @Param sort query string false "Field and ASC/DESC direction"
// @Param filter query string false "Name, soldOut or ID filter"
// @Success 200 {array} models.LoungeItem
// @Router /lounge-items [get]
func GetLoungeItems(w http.ResponseWriter, r *http.Request) {
	params, _ := utils.ParseListParams(r.URL.Query())
	column, direction := "sort_order", "ASC"
	if raw := r.URL.Query().Get("sort"); raw != "" {
		var sort []string
		if json.Unmarshal([]byte(raw), &sort) != nil || len(sort) != 2 {
			http.Error(w, "Invalid sort", http.StatusBadRequest)
			return
		}
		var ok bool
		column, ok = loungeSortColumns[sort[0]]
		direction = sort[1]
		if !ok || (direction != "ASC" && direction != "DESC") {
			http.Error(w, "Invalid sort", http.StatusBadRequest)
			return
		}
	}
	query := db.DB.Model(&models.LoungeItem{})
	if raw := r.URL.Query().Get("filter"); raw != "" {
		var filters map[string]json.RawMessage
		if json.Unmarshal([]byte(raw), &filters) != nil {
			http.Error(w, "Invalid filter", http.StatusBadRequest)
			return
		}
		for key, value := range filters {
			switch key {
			case "name":
				var name string
				if json.Unmarshal(value, &name) != nil {
					http.Error(w, "Invalid name filter", http.StatusBadRequest)
					return
				}
				query = query.Where("name = ?", name)
			case "soldOut":
				var soldOut bool
				if json.Unmarshal(value, &soldOut) != nil {
					http.Error(w, "Invalid soldOut filter", http.StatusBadRequest)
					return
				}
				query = query.Where("sold_out = ?", soldOut)
			case "id":
				var ids []uint
				if json.Unmarshal(value, &ids) != nil {
					http.Error(w, "Invalid id filter", http.StatusBadRequest)
					return
				}
				query = query.Where("id IN ?", ids)
			default:
				http.Error(w, "Unsupported filter", http.StatusBadRequest)
				return
			}
		}
	}
	start := 0
	if r.URL.Query().Has("range") {
		var bounds []int
		if json.Unmarshal([]byte(r.URL.Query().Get("range")), &bounds) != nil ||
			len(bounds) != 2 || bounds[0] < 0 || bounds[1] < bounds[0] || bounds[1]-bounds[0] >= 1000 {
			http.Error(w, "Invalid range", http.StatusBadRequest)
			return
		}
		start = params.Range[0]
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		log.Printf("lounge-items count failed: %v", err)
		http.Error(w, "List failed", http.StatusInternalServerError)
		return
	}
	query = query.Order(column + " " + direction).Order("id ASC")
	if r.URL.Query().Has("range") {
		query = query.Offset(start).Limit(params.Range[1] - start + 1)
	}
	items := make([]models.LoungeItem, 0)
	if err := query.Find(&items).Error; err != nil {
		log.Printf("lounge-items list failed: %v", err)
		http.Error(w, "List failed", http.StatusInternalServerError)
		return
	}
	end := start
	if len(items) > 0 {
		end = start + len(items) - 1
	}
	w.Header().Set("Access-Control-Expose-Headers", "Content-Range")
	w.Header().Set("Content-Range", fmt.Sprintf("lounge-items %d-%d/%d", start, end, total))
	writeJSONResponse(w, http.StatusOK, items)
}

func loadLoungeItem(w http.ResponseWriter, r *http.Request) (*models.LoungeItem, bool) {
	id, err := strconv.ParseUint(mux.Vars(r)["id"], 10, 64)
	if err != nil || id == 0 {
		http.Error(w, "Invalid item ID", http.StatusBadRequest)
		return nil, false
	}
	var item models.LoungeItem
	if err := db.DB.First(&item, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			http.Error(w, "Lounge item not found", http.StatusNotFound)
		} else {
			log.Printf("lounge item lookup failed: %v", err)
			http.Error(w, "Lookup failed", http.StatusInternalServerError)
		}
		return nil, false
	}
	return &item, true
}

// GetLoungeItemByID returns a lounge item.
// @Summary Get lounge item
// @Tags lounge-items
// @Produce json
// @Param id path int true "Item ID"
// @Success 200 {object} models.LoungeItem
// @Router /lounge-items/{id} [get]
func GetLoungeItemByID(w http.ResponseWriter, r *http.Request) {
	if item, ok := loadLoungeItem(w, r); ok {
		writeJSONResponse(w, http.StatusOK, item)
	}
}

func applyLoungeFields(values url.Values, item *models.LoungeItem) (map[string]any, error) {
	updates := map[string]any{}
	if values.Has("name") {
		item.Name = strings.TrimSpace(values.Get("name"))
		updates["name"] = item.Name
	}
	for _, field := range []struct {
		key, column string
		target      **string
	}{
		{"nameSv", "name_sv", &item.NameSv},
		{"description", "description", &item.Description},
		{"descriptionSv", "description_sv", &item.DescriptionSv},
	} {
		if values.Has(field.key) {
			value := values.Get(field.key)
			*field.target = &value
			NormalizeOptionalStringPointers(field.target)
			updates[field.column] = *field.target
		}
		if values.Has("categoryId") {
			value, err := strconv.ParseUint(values.Get("categoryId"), 10, 64)
			if err != nil || value == 0 || value > 9007199254740991 {
				return nil, errors.New("categoryId must be a positive safe integer")
			}
			item.CategoryID = uint(value)
			updates["category_id"] = item.CategoryID
		}
	}
	if values.Has("sortOrder") {
		value, err := strconv.ParseInt(values.Get("sortOrder"), 10, 64)
		if err != nil || value < 0 || value > 9007199254740991 {
			return nil, errors.New("sortOrder must be a non-negative safe integer")
		}
		item.SortOrder = value
		updates["sort_order"] = value
	}
	if values.Has("maxPerOrder") {
		if strings.TrimSpace(values.Get("maxPerOrder")) == "" {
			item.MaxPerOrder = nil
		} else {
			value, err := strconv.ParseInt(values.Get("maxPerOrder"), 10, 64)
			if err != nil || value <= 0 || value > 9007199254740991 {
				return nil, errors.New("maxPerOrder must be a positive safe integer")
			}
			item.MaxPerOrder = &value
		}
		updates["max_per_order"] = item.MaxPerOrder
	}
	if values.Has("soldOut") {
		value := values.Get("soldOut")
		if value != "true" && value != "false" {
			return nil, errors.New("soldOut must be true or false")
		}
		item.SoldOut = value == "true"
		updates["sold_out"] = item.SoldOut
	}
	if item.Name == "" {
		return nil, errors.New("name is required")
	}
	return updates, nil
}

func parseLoungeMutation(w http.ResponseWriter, r *http.Request, item *models.LoungeItem) (map[string]any, bool) {
	r.Body = http.MaxBytesReader(w, r.Body, 64<<10)
	var fields map[string]json.RawMessage
	if err := json.NewDecoder(r.Body).Decode(&fields); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return nil, false
	}
	values := url.Values{}
	for key, raw := range fields {
		switch key {
		case "id", "sortOrder":
			continue
		case "name", "nameSv", "description", "descriptionSv":
			var text *string
			if json.Unmarshal(raw, &text) != nil {
				http.Error(w, "Invalid text field", http.StatusBadRequest)
				return nil, false
			}
			value := ""
			if text != nil {
				value = *text
			}
			values.Set(key, value)
		case "categoryId", "maxPerOrder", "soldOut":
			if string(raw) == "null" {
				values.Set(key, "")
			} else {
				values.Set(key, string(raw))
			}
		default:
			http.Error(w, "Unsupported field", http.StatusBadRequest)
			return nil, false
		}
	}
	updates, err := applyLoungeFields(values, item)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return nil, false
	}
	if item.CategoryID == 0 {
		http.Error(w, "Category is required", http.StatusBadRequest)
		return nil, false
	}
	var count int64
	if err := db.DB.Model(&models.LoungeCategory{}).Where("id = ?", item.CategoryID).Count(&count).Error; err != nil {
		log.Printf("lounge category lookup failed: %v", err)
		http.Error(w, "Category lookup failed", http.StatusInternalServerError)
		return nil, false
	}
	if count != 1 {
		http.Error(w, "Category not found", http.StatusBadRequest)
		return nil, false
	}
	return updates, true
}

// CreateLoungeItem creates a lounge item.
// @Summary Create lounge item
// @Tags lounge-items
// @Accept json
// @Produce json
// @Param body body models.LoungeItem true "Categorized item"
// @Success 201 {object} models.LoungeItem
// @Security BearerAuth
// @Router /lounge-items [post]
func CreateLoungeItem(w http.ResponseWriter, r *http.Request) {
	var item models.LoungeItem
	if _, ok := parseLoungeMutation(w, r, &item); !ok {
		return
	}
	if err := createWithAudit(r, "lounge-items", &item, func(tx *gorm.DB) error {
		if _, err := lockLoungeCatalogue(tx); err != nil {
			return err
		}
		if err := tx.Model(&models.LoungeItem{}).Select("COALESCE(MAX(sort_order), -1) + 1").Where("category_id = ?", item.CategoryID).Scan(&item.SortOrder).Error; err != nil {
			return err
		}
		return tx.Create(&item).Error
	}, nil); err != nil {
		log.Printf("lounge item create failed: %v", err)
		http.Error(w, "Create failed", http.StatusInternalServerError)
		return
	}
	writeCreatedJSONResponse(w, item)
}

// UpdateLoungeItem applies only supplied fields; empty optional fields clear values.
// @Summary Update lounge item
// @Tags lounge-items
// @Accept json
// @Produce json
// @Param id path int true "Item ID"
// @Param body body models.LoungeItem true "Updated fields"
// @Success 200 {object} models.LoungeItem
// @Security BearerAuth
// @Router /lounge-items/{id} [put]
func UpdateLoungeItem(w http.ResponseWriter, r *http.Request) {
	item, ok := loadLoungeItem(w, r)
	if !ok {
		return
	}
	before := *item
	updates, ok := parseLoungeMutation(w, r, item)
	if !ok {
		return
	}
	if len(updates) == 0 {
		http.Error(w, "No fields supplied", http.StatusBadRequest)
		return
	}
	if err := updateWithAudit(r, "lounge-items", item.ID, before, item, func(tx *gorm.DB) error {
		if _, err := lockLoungeCatalogue(tx); err != nil {
			return err
		}
		if item.CategoryID != before.CategoryID {
			var position int64
			if err := tx.Model(&models.LoungeItem{}).Select("COALESCE(MAX(sort_order), -1) + 1").Where("category_id = ?", item.CategoryID).Scan(&position).Error; err != nil {
				return err
			}
			updates["sort_order"] = position
		}
		return tx.Model(item).Updates(updates).Error
	}, func(tx *gorm.DB) error {
		return tx.First(item, item.ID).Error
	}); err != nil {
		log.Printf("lounge item update failed: %v", err)
		http.Error(w, "Update failed", http.StatusInternalServerError)
		return
	}
	writeJSONResponse(w, http.StatusOK, item)
}

// DeleteLoungeItem deletes a lounge item.
// @Summary Delete lounge item
// @Tags lounge-items
// @Param id path int true "Item ID"
// @Success 204 {string} string "Deleted"
// @Security BearerAuth
// @Router /lounge-items/{id} [delete]
func DeleteLoungeItem(w http.ResponseWriter, r *http.Request) {
	item, ok := loadLoungeItem(w, r)
	if !ok {
		return
	}
	writeDeleteResponseWithAudit[models.LoungeItem](w, r, "lounge-items", strconv.FormatUint(uint64(item.ID), 10), "Lounge item not found", nil, func(tx *gorm.DB, _ *models.LoungeItem) error {
		_, err := lockLoungeCatalogue(tx)
		return err
	})
}
