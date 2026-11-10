package controllers

import (
	"ArmadaCMS/main/audit"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"reflect"
	"sort"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"ArmadaCMS/main/db"
	"ArmadaCMS/main/models"
	"ArmadaCMS/main/utils"
)

// ---------- Eventro API Models ----------

type eventroExhibitorResponse struct {
	ID              string           `json:"id"`
	Organization    eventroOrg       `json:"organization"`
	Catalogue       eventroCatalogue `json:"catalogue"`
	OrderedProducts []eventroProduct `json:"orderedProducts"`
}

type eventroOrg struct {
	Name          string `json:"name"`
	LogoThumbnail string `json:"logoThumbnail"`
	Website       string `json:"website"`
	Logo          string `json:"logo"`
}

type eventroCatalogue struct {
	Educations  []string `json:"educations"`
	Employments []string `json:"employments"`
	Locations   []string `json:"locations"`
	Industries  []string `json:"industries"`
	About       string   `json:"about"`
}

// ---------- Controller ----------

// FetchExhibitorsEventro syncs exhibitors from the Eventro API into the local database and returns the current list.
// @Summary Sync & list exhibitors from Eventro
// @Tags eventro
// @Produce json
// @Param fairId query string true "Eventro fair instance ID"
// @Success 200 {array} models.Exhibitor
// @Failure 400 {string} string "fairId query parameter is required"
// @Failure 500 {string} string "Failed to fetch from Eventro"
// @Security BearerAuth
// @Router /eventroexhibitors [get]
func FetchExhibitorsEventro(w http.ResponseWriter, r *http.Request) {
	client := &http.Client{Timeout: 30 * time.Second}
	fairID, ok := requireEventroFairID(w, r)
	if !ok {
		return
	}
	baseURL := fmt.Sprintf("https://app.eventro.se/api/v1/fairs/%s/exhibitors/", url.PathEscape(fairID))

	allExhibitors, err := fetchEventroPages[eventroExhibitorResponse](r.Context(), client, baseURL, "exhibitors", os.Getenv("EVENTRO_API"), os.Getenv("EVENTRO_ORG"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	for _, exhibitor := range allExhibitors {
		if strings.TrimSpace(exhibitor.Organization.Name) == "" {
			http.Error(w, "missing exhibitor name", http.StatusBadGateway)
			return
		}
	}

	inserted := 0
	updated := 0
	unchanged := 0
	skipped := 0
	failed := 0
	group, err := startAuditGroup(r, "sync", "exhibitors", fairID, map[string]any{"source": "eventro", "status": "running"})
	if err != nil {
		http.Error(w, "failed to start audit log", http.StatusInternalServerError)
		return
	}
	childRequest := audit.WithParent(r, group.ID)

	for _, e := range allExhibitors {
		created, changed, err := syncEventroExhibitorWithAudit(childRequest, e, fairID)
		if errors.Is(err, errEventroSyncSkipped) {
			skipped++
			continue
		}
		if err != nil {
			log.Printf("❌ Failed upserting exhibitor %s (%s): %v", e.ID, e.Organization.Name, err)
			failed++
			continue
		}
		if !changed {
			unchanged++
		} else if created {
			inserted++
		} else {
			updated++
		}
	}

	status := auditGroupStatus(inserted+updated+unchanged+skipped, failed)
	if err := finalizeAuditGroup(group.ID, status, map[string]any{"source": "eventro", "inserted": inserted, "updated": updated, "unchanged": unchanged, "skipped": skipped, "failed": failed}); err != nil {
		http.Error(w, "failed to finalize audit log", http.StatusInternalServerError)
		return
	}
	if inserted+updated > 0 {
		for _, tag := range []string{"exhibitors", "industries", "employments", "programs"} {
			go utils.RevalidateTag(tag)
		}
	}
	log.Printf("✅ Sync completed — inserted: %d, updated: %d, skipped: %d", inserted, updated, skipped)
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "Sync completed — inserted: %d, updated: %d, unchanged: %d, skipped: %d, failed: %d", inserted, updated, unchanged, skipped, failed)
}

func syncEventroExhibitorWithAudit(r *http.Request, e eventroExhibitorResponse, fairID string) (bool, bool, error) {
	ex := mapEventroToExhibitor(e)
	tier, unmapped := mappedEventroTier(e.OrderedProducts)
	ex.Tier = tier
	created := false
	changed := false
	err := db.DB.Transaction(func(tx *gorm.DB) error {
		if err := beginEventroImport(tx, "exhibitors", strings.TrimSpace(e.ID)); err != nil {
			return err
		}
		if err := filterBlockedEventroCategories(tx, &ex); err != nil {
			return err
		}
		var before models.Exhibitor
		findErr := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Preload("Industries").Preload("Employments").Preload("Programs").
			Where("eventro_id = ?", ex.EventroID).First(&before).Error
		if findErr != nil && findErr != gorm.ErrRecordNotFound {
			return findErr
		}
		state := models.EventroSyncState{Resource: "exhibitors", RecordID: before.ID, FairID: fairID}
		if before.ID != 0 {
			if err := tx.First(&state, "resource = ? AND record_id = ?", "exhibitors", before.ID).Error; err != nil && err != gorm.ErrRecordNotFound {
				return err
			}
		}
		stateBefore, _ := json.Marshal(state)
		state.FairID, state.UnmappedProducts = fairID, unmapped
		ex = mergeEventroExhibitor(before, ex, &state, findErr == gorm.ErrRecordNotFound)
		result := tx.Session(&gorm.Session{FullSaveAssociations: false, SkipHooks: true}).
			Omit("Industries", "Industries.*", "Employments", "Employments.*", "Programs", "Programs.*").
			Clauses(clause.OnConflict{
				Columns: []clause.Column{{Name: "eventro_id"}},
				DoUpdates: clause.AssignmentColumns([]string{
					"name", "tier", "company_website", "about", "logo_freesize_url", "logo_thumbnail", "cities",
				}),
			}).Create(&ex)
		if result.Error != nil {
			return result.Error
		}
		var persisted struct {
			ID uint
		}
		if err := tx.Model(&models.Exhibitor{}).
			Select("id").
			Where("eventro_id = ?", ex.EventroID).
			Scan(&persisted).Error; err != nil {
			return err
		}
		ex.ID = persisted.ID
		if err := persistEventroExhibitorCategories(tx, r, &ex, before); err != nil {
			return err
		}
		state.RecordID = ex.ID
		if err := tx.Save(&state).Error; err != nil {
			return err
		}
		stateAfter, _ := json.Marshal(state)
		if string(stateBefore) != string(stateAfter) {
			if err := audit.LogUpdate(tx, r, "eventro_sync_states", ex.ID, json.RawMessage(stateBefore), state); err != nil {
				return err
			}
		}
		if err := tx.Preload("Industries").Preload("Employments").Preload("Programs").
			Where("eventro_id = ?", ex.EventroID).First(&ex).Error; err != nil {
			return err
		}
		sortExhibitorAssociations(&before)
		sortExhibitorAssociations(&ex)
		created = findErr == gorm.ErrRecordNotFound
		changed = created || !reflect.DeepEqual(before, ex)
		if !changed {
			return nil
		}
		if created {
			return audit.LogCreate(tx, r, "exhibitors", ex.ID, ex)
		}
		return audit.LogUpdate(tx, r, "exhibitors", ex.ID, before, ex)
	})
	return created, changed, err
}

func sameIndustryIDs(left, right []models.Industry) bool {
	if len(left) != len(right) {
		return false
	}
	ids := make(map[uint]struct{}, len(left))
	for _, item := range left {
		ids[item.ID] = struct{}{}
	}
	for _, item := range right {
		if _, ok := ids[item.ID]; !ok {
			return false
		}
	}
	return true
}

func sameEmploymentIDs(left, right []models.Employment) bool {
	if len(left) != len(right) {
		return false
	}
	ids := make(map[uint]struct{}, len(left))
	for _, item := range left {
		ids[item.ID] = struct{}{}
	}
	for _, item := range right {
		if _, ok := ids[item.ID]; !ok {
			return false
		}
	}
	return true
}

func sameProgramIDs(left, right []models.Program) bool {
	if len(left) != len(right) {
		return false
	}
	ids := make(map[uint]struct{}, len(left))
	for _, item := range left {
		ids[item.ID] = struct{}{}
	}
	for _, item := range right {
		if _, ok := ids[item.ID]; !ok {
			return false
		}
	}
	return true
}

func sortExhibitorAssociations(exhibitor *models.Exhibitor) {
	sort.Slice(exhibitor.Industries, func(i, j int) bool { return exhibitor.Industries[i].ID < exhibitor.Industries[j].ID })
	sort.Slice(exhibitor.Employments, func(i, j int) bool { return exhibitor.Employments[i].ID < exhibitor.Employments[j].ID })
	sort.Slice(exhibitor.Programs, func(i, j int) bool { return exhibitor.Programs[i].ID < exhibitor.Programs[j].ID })
}

// ---------- Mapping Helpers ----------

func mapEventroToExhibitor(e eventroExhibitorResponse) models.Exhibitor {

	// Safely handle optional strings
	website := strings.TrimSpace(e.Organization.Website)
	logo := strings.TrimSpace(e.Organization.Logo)
	about := strings.TrimSpace(e.Catalogue.About)
	eventroId := strings.TrimSpace(e.ID)

	// Join all locations into a single cities string
	var cities *string
	if locations := normalizedEventroLocations(e.Catalogue.Locations); len(locations) > 0 {
		joined := strings.Join(locations, ", ")
		cities = &joined
	}

	industries := make([]models.Industry, 0, len(e.Catalogue.Industries))
	for _, name := range sortedEventroCategoryLabels(e.Catalogue.Industries) {
		industries = append(industries, models.Industry{Name: name})
	}

	employments := make([]models.Employment, 0, len(e.Catalogue.Employments))
	for _, name := range sortedEventroCategoryLabels(e.Catalogue.Employments) {
		employments = append(employments, models.Employment{Name: name})
	}

	programs := make([]models.Program, 0, len(e.Catalogue.Educations))
	for _, name := range sortedEventroCategoryLabels(e.Catalogue.Educations) {
		programs = append(programs, models.Program{Name: name})
	}

	return models.Exhibitor{
		EventroID: nullIfEmpty(eventroId),
		Name:      e.Organization.Name,
		Type:      "company",

		CompanyWebsite:      nullIfEmpty(website),
		About:               nullIfEmpty(about),
		LogoFreesizeUrl:     nullIfEmpty(logo),
		LogoThumbnail:       nullIfEmpty(strings.TrimSpace(e.Organization.LogoThumbnail)),
		Cities:              cities,
		Industries:          industries,
		Employments:         employments,
		Programs:            programs,
		ClimateCompensation: false,
		Flyer:               "",
	}
}

func nullIfEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
