package controllers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

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

func FetchExhibitorsEventro(w http.ResponseWriter, r *http.Request) {
	client := &http.Client{Timeout: 30 * time.Second}
	fairID := os.Getenv("EVENTRO_FAIR_ID")
	baseURL := fmt.Sprintf("https://app.eventro.se/api/v1/fairs/%s/exhibitors/", fairID)

	allExhibitors := []eventroExhibitorResponse{}
	page := 1

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
			break // no more pages
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

		result := db.DB.Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "eventro_id"}},
			DoUpdates: clause.AssignmentColumns([]string{
				"name", "tier", "company_website", "about",
				"logo_freesize_url", "fair_location",
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
	}

	log.Printf("✅ Sync completed — inserted: %d, updated: %d", inserted, updated)
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(fmt.Sprintf("Sync completed — inserted: %d, updated: %d", inserted, updated)))
}

// ---------- Mapping Helpers ----------

func mapEventroToExhibitor(e eventroExhibitorResponse) models.Exhibitor {
	tierStr := deriveTier(e.OrderedProducts)
	tier := models.Tier(tierStr)

	location := ""
	if len(e.Catalogue.Locations) > 0 {
		location = e.Catalogue.Locations[0]
	}

	return models.Exhibitor{
		EventroID:           e.ID,
		Name:                e.Organization.Name,
		CompanyWebsite:      &e.Organization.Website,
		About:               &e.Catalogue.About,
		Tier:                &tier,
		LogoFreesizeUrl:     &e.Organization.Logo,
		FairLocation:        location,
		Type:                "company",
		ClimateCompensation: false,
		Flyer:               "",
	}
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
	return "Standard"
}
