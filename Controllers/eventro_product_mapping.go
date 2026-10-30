package controllers

import "ArmadaCMS/main/models"

type eventroProduct struct {
	ProductID   string `json:"productId"`
	Name        string `json:"name"`
	DisplayName string `json:"displayName"`
}

func mappedEventroTier(products []eventroProduct) (*models.Tier, []models.EventroOrderedProduct) {
	mapping := map[string]models.Tier{
		"Startup Kit": models.TierStandard,
		"Bronze Kit":  models.TierBronze,
		"Silver Kit":  models.TierSilver,
		"Gold Kit":    models.TierGold,
	}
	rank := map[models.Tier]int{models.TierStandard: 1, models.TierBronze: 2, models.TierSilver: 3, models.TierGold: 4}
	var selected *models.Tier
	unmapped := []models.EventroOrderedProduct{}
	for _, product := range products {
		tier, ok := mapping[product.DisplayName]
		if !ok {
			unmapped = append(unmapped, models.EventroOrderedProduct{ProductID: product.ProductID, Name: product.Name, DisplayName: product.DisplayName})
			continue
		}
		if selected == nil || rank[tier] > rank[*selected] {
			selected = &tier
		}
	}
	return selected, unmapped
}
