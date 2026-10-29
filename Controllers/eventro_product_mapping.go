package controllers

import (
	"ArmadaCMS/main/models"
	"encoding/json"
	"fmt"
	"strings"
)

// Ignore is a product-mapping action, never an exhibitor tier.
const eventroProductIgnore = "Ignore"

type eventroProduct struct {
	ProductID   string `json:"productId"`
	Name        string `json:"name"`
	DisplayName string `json:"displayName"`
}

func parseEventroProductTiers(raw string) (map[string]models.Tier, error) {
	mapping := map[string]models.Tier{}
	if strings.TrimSpace(raw) == "" {
		return mapping, nil
	}
	if err := json.Unmarshal([]byte(raw), &mapping); err != nil {
		return nil, fmt.Errorf("invalid EVENTRO_PRODUCT_TIERS JSON: %w", err)
	}
	for id, tier := range mapping {
		if strings.TrimSpace(id) == "" || (tier != models.TierGold && tier != models.TierSilver && tier != models.TierBronze && tier != models.TierStandard && tier != eventroProductIgnore) {
			return nil, fmt.Errorf("invalid EVENTRO_PRODUCT_TIERS entry %q: expected Standard, Bronze, Silver, Gold, or Ignore", id)
		}
	}
	return mapping, nil
}

func mappedEventroTier(products []eventroProduct, mapping map[string]models.Tier) (*models.Tier, []models.EventroOrderedProduct) {
	rank := map[models.Tier]int{models.TierStandard: 1, models.TierBronze: 2, models.TierSilver: 3, models.TierGold: 4}
	var selected *models.Tier
	unmapped := []models.EventroOrderedProduct{}
	for _, product := range products {
		tier, ok := mapping[product.ProductID]
		if !ok {
			unmapped = append(unmapped, models.EventroOrderedProduct{ProductID: product.ProductID, Name: product.Name, DisplayName: product.DisplayName})
			continue
		}
		if tier == eventroProductIgnore {
			continue
		}
		if selected == nil || rank[tier] > rank[*selected] {
			selected = &tier
		}
	}
	return selected, unmapped
}
