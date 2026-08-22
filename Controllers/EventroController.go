package controllers

import (
	"ArmadaCMS/main/audit"
	"encoding/json"
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

type eventroExhibitorsResponse struct {
	Exhibitors []eventroExhibitorResponse `json:"exhibitors"`
	Count      int                        `json:"count"`
}

type eventroExhibitorResponse struct {
	ID              string           `json:"id"`
	Organization    eventroOrg       `json:"organization"`
	Catalogue       eventroCatalogue `json:"catalogue"`
	OrderedProducts []eventroProduct `json:"orderedProducts"`
}

type eventroOrg struct {
	Name    string `json:"name"`
	Website string `json:"website"`
	Logo    string `json:"logo"`
}

type eventroCatalogue struct {
	Educations  []string `json:"educations"`
	Employments []string `json:"employments"`
	Locations   []string `json:"locations"`
	Industries  []string `json:"industries"`
	About       string   `json:"about"`
}

type eventroProduct struct {
	ProductID   string `json:"productId"`
	Name        string `json:"name"`
	DisplayName string `json:"displayName"`
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

	allExhibitors := []eventroExhibitorResponse{}
	page := 0

	for {
		url := fmt.Sprintf("%s?pageIndex=%d", baseURL, page)
		req, err := http.NewRequest(http.MethodGet, url, nil)
		if err != nil {
			http.Error(w, "failed to create request", http.StatusInternalServerError)
			return
		}
		req.Header.Set("Authorization", "Bearer "+os.Getenv("EVENTRO_API"))
		req.Header.Set("organization", os.Getenv("EVENTRO_ORG"))

		resp, err := client.Do(req)
		if err != nil {
			http.Error(w, "failed to contact Eventro API", http.StatusBadGateway)
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			http.Error(w, "unexpected Eventro status: "+resp.Status, http.StatusBadGateway)
			return
		}

		var result eventroExhibitorsResponse
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			log.Printf("❌ Failed to decode Eventro response (page %d): %v", page, err)
			http.Error(w, "failed to decode Eventro response", http.StatusInternalServerError)
			return
		}

		if len(result.Exhibitors) == 0 {
			break
		}

		allExhibitors = append(allExhibitors, result.Exhibitors...)

		if len(allExhibitors) >= result.Count {
			break
		}
		page++
	}

	inserted := 0
	updated := 0
	unchanged := 0
	failed := 0
	group, err := startAuditGroup(r, "sync", "exhibitors", fairID, map[string]any{"source": "eventro", "status": "running"})
	if err != nil {
		http.Error(w, "failed to start audit log", http.StatusInternalServerError)
		return
	}
	childRequest := audit.WithParent(r, group.ID)

	for _, e := range allExhibitors {
		ex := mapEventroToExhibitor(e)
		created := false
		changed := false
		err := db.DB.Transaction(func(tx *gorm.DB) error {
			var before models.Exhibitor
			findErr := tx.Preload("Industries").Preload("Employments").Preload("Programs").
				Where("eventro_id = ?", ex.EventroID).First(&before).Error
			if findErr != nil && findErr != gorm.ErrRecordNotFound {
				return findErr
			}
			result := tx.Session(&gorm.Session{FullSaveAssociations: false, SkipHooks: true}).
				Omit("Industries", "Industries.*", "Employments", "Employments.*", "Programs", "Programs.*").
				Clauses(clause.OnConflict{
					Columns: []clause.Column{{Name: "eventro_id"}},
					DoUpdates: clause.AssignmentColumns([]string{
						"name", "tier", "company_website", "about", "logo_freesize_url", "cities",
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
			for i := range ex.Industries {
				result := tx.Where("name = ?", ex.Industries[i].Name).FirstOrCreate(&ex.Industries[i])
				if result.Error != nil {
					return result.Error
				}
				if result.RowsAffected > 0 {
					if err := audit.LogCreate(tx, childRequest, "industries", ex.Industries[i].ID, ex.Industries[i]); err != nil {
						return err
					}
				}
			}
			for i := range ex.Employments {
				result := tx.Where("name = ?", ex.Employments[i].Name).FirstOrCreate(&ex.Employments[i])
				if result.Error != nil {
					return result.Error
				}
				if result.RowsAffected > 0 {
					if err := audit.LogCreate(tx, childRequest, "employments", ex.Employments[i].ID, ex.Employments[i]); err != nil {
						return err
					}
				}
			}
			for i := range ex.Programs {
				result := tx.Where("name = ?", ex.Programs[i].Name).FirstOrCreate(&ex.Programs[i])
				if result.Error != nil {
					return result.Error
				}
				if result.RowsAffected > 0 {
					if err := audit.LogCreate(tx, childRequest, "programs", ex.Programs[i].ID, ex.Programs[i]); err != nil {
						return err
					}
				}
			}
			if !sameIndustryIDs(before.Industries, ex.Industries) {
				if err := tx.Model(&ex).Association("Industries").Replace(ex.Industries); err != nil {
					return err
				}
			}
			if !sameEmploymentIDs(before.Employments, ex.Employments) {
				if err := tx.Model(&ex).Association("Employments").Replace(ex.Employments); err != nil {
					return err
				}
			}
			if !sameProgramIDs(before.Programs, ex.Programs) {
				if err := tx.Model(&ex).Association("Programs").Replace(ex.Programs); err != nil {
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
				return audit.LogCreate(tx, childRequest, "exhibitors", ex.ID, ex)
			}
			return audit.LogUpdate(tx, childRequest, "exhibitors", ex.ID, before, ex)
		})
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

	status := auditGroupStatus(inserted+updated+unchanged, failed)
	if err := finalizeAuditGroup(group.ID, status, map[string]any{"source": "eventro", "inserted": inserted, "updated": updated, "unchanged": unchanged, "failed": failed}); err != nil {
		http.Error(w, "failed to finalize audit log", http.StatusInternalServerError)
		return
	}
	if inserted+updated > 0 {
		for _, tag := range []string{"exhibitors", "industries", "employments", "programs"} {
			go utils.RevalidateTag(tag)
		}
	}
	log.Printf("✅ Sync completed — inserted: %d, updated: %d", inserted, updated)
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "Sync completed — inserted: %d, updated: %d", inserted, updated)
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
	tierStr := deriveTier(e.OrderedProducts)
	tier := models.Tier(tierStr)

	// Safely handle optional strings
	website := strings.TrimSpace(e.Organization.Website)
	logo := strings.TrimSpace(e.Organization.Logo)
	about := strings.TrimSpace(e.Catalogue.About)
	eventroId := strings.TrimSpace(e.ID)

	// Join all locations into a single cities string
	var cities *string
	if len(e.Catalogue.Locations) > 0 {
		joined := strings.Join(e.Catalogue.Locations, ", ")
		cities = &joined
	}

	// Map related entities
	industries := make([]models.Industry, 0, len(e.Catalogue.Industries))
	for _, i := range e.Catalogue.Industries {
		name := strings.TrimSpace(i)
		if name == "" {
			continue
		}
		industries = append(industries, models.Industry{Name: name})
	}

	employments := make([]models.Employment, 0, len(e.Catalogue.Employments))
	for _, emp := range e.Catalogue.Employments {
		name := strings.TrimSpace(emp)
		if name == "" {
			continue
		}
		employments = append(employments, models.Employment{Name: name})
	}

	programs := make([]models.Program, 0, len(e.Catalogue.Educations))
	for _, edu := range e.Catalogue.Educations {
		name := strings.TrimSpace(edu)
		if name == "" {
			continue
		}
		programs = append(programs, models.Program{Name: name})
	}

	return models.Exhibitor{
		EventroID:           nullIfEmpty(eventroId),
		Name:                e.Organization.Name,
		Type:                "company",
		Tier:                &tier,
		CompanyWebsite:      nullIfEmpty(website),
		About:               nullIfEmpty(about),
		LogoFreesizeUrl:     nullIfEmpty(logo),
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

// ---------- Tier Derivation ----------

func deriveTier(products []eventroProduct) string {
	for _, p := range products {
		n := strings.ToLower(p.Name)
		d := strings.ToLower(p.DisplayName)

		switch {
		case strings.Contains(n, "gold") || strings.Contains(d, "gold"):
			return "Gold"
		case strings.Contains(n, "silver") || strings.Contains(d, "silver"):
			return "Silver"
		case strings.Contains(n, "bronze") || strings.Contains(d, "bronze"):
			return "Bronze"
		}
	}
	return "Bronze"
}
