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

// GetHighlightCards returns a paginated list of highlight cards.
// @Summary List highlight cards
// @Tags highlight-cards
// @Produce json
// @Param range query string false "Pagination range, e.g. [0,24]"
// @Param sort query string false "Sort, e.g. [\"title\",\"ASC\"]"
// @Param filter query string false "Filter"
// @Success 200 {array} models.HighlightCard
// @Header 200 {string} Content-Range "highlightcards 0-24/5"
// @Security BearerAuth
// @Router /highlightcards [get]
func GetHighlightCards(w http.ResponseWriter, r *http.Request) {
	params, _ := utils.ParseListParams(r.URL.Query())

	var items []models.HighlightCard
	query := db.DB.Model(&models.HighlightCard{})

	for k, v := range params.Filter {
		query = query.Where(utils.ToSnakeCase(k)+" = ?", v)
	}
	if len(params.Sort) == 2 {
		query = query.Order(utils.ToSnakeCase(params.Sort[0]) + " " + params.Sort[1])
	}

	start, end := params.Range[0], params.Range[1]

	var total int64
	db.DB.Model(&models.HighlightCard{}).Count(&total)

	query = query.Offset(start).Limit(end - start + 1)
	query.Find(&items)

	w.Header().Set("Access-Control-Expose-Headers", "Content-Range")
	w.Header().Set("Content-Range", fmt.Sprintf("highlightcards %d-%d/%d", start, end, total))
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(items)
}

// GetHighlightCardByID returns a single highlight card by ID.
// @Summary Get highlight card by ID
// @Tags highlight-cards
// @Produce json
// @Param id path int true "HighlightCard ID"
// @Success 200 {object} models.HighlightCard
// @Failure 404 {string} string "highlight card not found"
// @Security BearerAuth
// @Router /highlightcards/{id} [get]
func GetHighlightCardByID(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]

	var item models.HighlightCard
	if err := db.DB.First(&item, id).Error; err != nil {
		http.Error(w, "highlight card not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(item)
}

// CreateHighlightCard creates a new highlight card.
// @Summary Create highlight card
// @Tags highlight-cards
// @Accept json
// @Produce json
// @Param body body models.HighlightCard true "Highlight card data"
// @Success 201 {object} models.HighlightCard
// @Failure 400 {string} string "Invalid body"
// @Failure 500 {string} string "Create failed"
// @Security BearerAuth
// @Router /highlightcards [post]
func CreateHighlightCard(w http.ResponseWriter, r *http.Request) {
	var item models.HighlightCard
	if err := json.NewDecoder(r.Body).Decode(&item); err != nil {
		log.Println(err)
		http.Error(w, "Invalid body", http.StatusBadRequest)
		return
	}

	NormalizeOptionalStringPointers(&item.Brand, &item.LinkText, &item.LinkUrl, &item.CtaEventName)

	if err := createWithAudit(r, "highlightcards", &item, func(tx *gorm.DB) error {
		return tx.Create(&item).Error
	}, nil, "highlight-cards"); err != nil {
		http.Error(w, "Create failed", http.StatusInternalServerError)
		return
	}
	writeCreatedJSONResponse(w, item)
}

// UpdateHighlightCard updates a highlight card by ID.
// @Summary Update highlight card
// @Tags highlight-cards
// @Accept json
// @Produce json
// @Param id path int true "HighlightCard ID"
// @Param body body models.HighlightCard true "Updated highlight card"
// @Success 200 {object} models.HighlightCard
// @Failure 400 {string} string "Invalid data"
// @Failure 404 {string} string "Not found"
// @Failure 500 {string} string "Update failed"
// @Security BearerAuth
// @Router /highlightcards/{id} [put]
func UpdateHighlightCard(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]

	var item models.HighlightCard
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
			"brand":        {},
			"linkText":     {},
			"linkUrl":      {},
			"ctaEventName": {},
		},
		map[string]struct{}{
			"id": {},
		},
	)

	before := item
	if err := updateWithAudit(r, "highlightcards", id, before, &item, func(tx *gorm.DB) error {
		return tx.Model(&item).Updates(updateMap).Error
	}, func(tx *gorm.DB) error {
		return tx.First(&item, id).Error
	}, "highlight-cards"); err != nil {
		http.Error(w, "Update failed", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(item)
}

// DeleteHighlightCard deletes a highlight card by ID.
// @Summary Delete highlight card
// @Tags highlight-cards
// @Produce json
// @Param id path int true "HighlightCard ID"
// @Success 200 {string} string "Deleted"
// @Failure 404 {string} string "highlight card not found"
// @Security BearerAuth
// @Router /highlightcards/{id} [delete]
func DeleteHighlightCard(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	writeDeleteResponseWithAudit[models.HighlightCard](w, r, "highlightcards", id, "highlight card not found", nil, "highlight-cards")
}
