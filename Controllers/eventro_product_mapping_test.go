package controllers

import (
	"ArmadaCMS/main/models"
	"encoding/json"
	"testing"
)

func TestEventroProductMapping(t *testing.T) {
	products := []eventroProduct{{ProductID: "silver-id", Name: "Misleading Gold", DisplayName: "Silver Kit"}, {ProductID: "gold-id", DisplayName: "Gold Kit"}, {ProductID: "addon", Name: "Add-on", DisplayName: "Extra power"}}
	for _, list := range [][]eventroProduct{products, {products[2], products[1], products[0]}} {
		tier, unknown := mappedEventroTier(list)
		if tier == nil || *tier != models.TierGold || len(unknown) != 1 || unknown[0].ProductID != "addon" || unknown[0].DisplayName != "Extra power" || unknown[0].Name != "Add-on" {
			t.Fatal("incorrect deterministic tier mapping")
		}
	}
}

func TestEventroPackageTiers(t *testing.T) {
	for displayName, want := range map[string]models.Tier{
		"Startup Kit": models.TierStandard,
		"Bronze Kit":  models.TierBronze,
		"Silver Kit":  models.TierSilver,
		"Gold Kit":    models.TierGold,
	} {
		tier, unknown := mappedEventroTier([]eventroProduct{{ProductID: "unrelated-id", DisplayName: displayName}})
		if tier == nil || *tier != want || len(unknown) != 0 {
			t.Fatalf("tier = %v, unknown = %v, want %s", tier, unknown, want)
		}
	}
	if tier, unknown := mappedEventroTier(nil); tier != nil || len(unknown) != 0 {
		t.Fatal("empty export assigned a tier or reported products")
	}
}

func TestEventroDisplayNameMapping(t *testing.T) {
	for _, id := range []string{"first-exhibitor-product", "second-exhibitor-product"} {
		tier, unknown := mappedEventroTier([]eventroProduct{{ProductID: id, DisplayName: "Gold Kit"}})
		if tier == nil || *tier != models.TierGold || len(unknown) != 0 {
			t.Fatal("the same displayName must map independently of product ID")
		}
	}
	for _, displayName := range []string{"", "gold kit", "Gold Kit Extra", " Gold Kit ", "Ignore"} {
		product := eventroProduct{ProductID: "Gold Kit", Name: "Gold Kit", DisplayName: displayName}
		tier, unknown := mappedEventroTier([]eventroProduct{product})
		if tier != nil || len(unknown) != 1 || unknown[0].DisplayName != displayName {
			t.Fatalf("non-matching displayName %q must be reported without name or ID fallback", displayName)
		}
	}
	var product eventroProduct
	if err := json.Unmarshal([]byte(`{"productId":"Gold Kit","name":"Gold Kit","displayName":null}`), &product); err != nil {
		t.Fatal(err)
	}
	if tier, unknown := mappedEventroTier([]eventroProduct{product}); tier != nil || len(unknown) != 1 {
		t.Fatal("null displayName must remain unmapped")
	}
}
