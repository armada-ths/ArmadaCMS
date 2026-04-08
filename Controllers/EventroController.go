package controllers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"ArmadaCMS/main/db"
	"ArmadaCMS/main/models"
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
// @Success 200 {array} models.Exhibitor
// @Failure 500 {string} string "Failed to fetch from Eventro"
// @Security BearerAuth
// @Router /eventroexhibitors [get]
func FetchExhibitorsEventro(w http.ResponseWriter, r *http.Request) {
	client := &http.Client{Timeout: 30 * time.Second}
	fairID := os.Getenv("EVENTRO_FAIR_ID")
	baseURL := fmt.Sprintf("https://app.eventro.se/api/v1/fairs/%s/exhibitors/", fairID)

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

	for _, e := range allExhibitors {
		ex := mapEventroToExhibitor(e)

		// Upsert base exhibitor fields
		result := db.DB.
			Session(&gorm.Session{
				FullSaveAssociations: false, // don't auto-save relationships
				SkipHooks:            true,  // prevent hooks that might trigger save
			}).
			Omit("Industries", "Industries.*",
				"Employments", "Employments.*",
				"Programs", "Programs.*").
			Clauses(clause.OnConflict{
				Columns: []clause.Column{{Name: "eventro_id"}},
				DoUpdates: clause.AssignmentColumns([]string{
					"name",
					"tier",
					"company_website",
					"about",
					"logo_freesize_url",
					"cities",
				}),
			}).Create(&ex)

		if result.Error != nil {
			log.Printf("❌ Failed upserting exhibitor %s (%s): %v", e.ID, e.Organization.Name, result.Error)
			continue
		}

		if result.RowsAffected == 1 {
			inserted++
		} else {
			updated++
		}

		for i := range ex.Industries {
			db.DB.Where("name = ?", ex.Industries[i].Name).FirstOrCreate(&ex.Industries[i])
		}
		for i := range ex.Employments {
			db.DB.Where("name = ?", ex.Employments[i].Name).FirstOrCreate(&ex.Employments[i])
		}
		for i := range ex.Programs {
			db.DB.Where("name = ?", ex.Programs[i].Name).FirstOrCreate(&ex.Programs[i])
		}

		// Now link
		db.DB.Model(&ex).Association("Industries").Replace(ex.Industries)
		db.DB.Model(&ex).Association("Employments").Replace(ex.Employments)
		db.DB.Model(&ex).Association("Programs").Replace(ex.Programs)
	}

	log.Printf("✅ Sync completed — inserted: %d, updated: %d", inserted, updated)
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(fmt.Sprintf("Sync completed — inserted: %d, updated: %d", inserted, updated)))
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
