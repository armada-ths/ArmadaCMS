package controllers

import (
	"ArmadaCMS/main/models"
	"testing"
)

func TestEventroProductMapping(t *testing.T) {
	mapping, err := parseEventroProductTiers(`{"gold-id":"Gold","silver-id":"Silver"}`)
	if err != nil {
		t.Fatal(err)
	}
	products := []eventroProduct{{ProductID: "silver-id", Name: "Misleading Gold"}, {ProductID: "gold-id"}, {ProductID: "addon"}}
	for _, list := range [][]eventroProduct{products, {products[2], products[1], products[0]}} {
		tier, unknown := mappedEventroTier(list, mapping)
		if tier == nil || *tier != models.TierGold || len(unknown) != 1 || unknown[0].ProductID != "addon" {
			t.Fatal("incorrect deterministic tier mapping")
		}
	}
	if tier, _ := mappedEventroTier(products, nil); tier != nil {
		t.Fatal("unknown tier defaulted")
	}
	for _, raw := range []string{`{`, `{"id":"gold"}`, `{"":"Gold"}`} {
		if _, err := parseEventroProductTiers(raw); err == nil {
			t.Fatal("invalid mapping accepted")
		}
	}
}

func TestEventroIgnoredProducts(t *testing.T) {
	mapping, err := parseEventroProductTiers(`{"addon":"Ignore","gold":"Gold","silver":"Silver","bronze":"Bronze","startup":"Standard"}`)
	if err != nil {
		t.Fatal(err)
	}
	for _, scenario := range []struct {
		name        string
		products    []eventroProduct
		wantTier    models.Tier
		wantUnknown int
	}{
		{"only ignored", []eventroProduct{{ProductID: "addon", Name: "Gold marketing"}}, "", 0},
		{"ignored before package", []eventroProduct{{ProductID: "addon"}, {ProductID: "silver"}}, models.TierSilver, 0},
		{"ignored after package", []eventroProduct{{ProductID: "bronze"}, {ProductID: "addon"}}, models.TierBronze, 0},
		{"standard remains classified", []eventroProduct{{ProductID: "addon"}, {ProductID: "startup"}}, models.TierStandard, 0},
		{"highest real tier wins", []eventroProduct{{ProductID: "silver"}, {ProductID: "addon"}, {ProductID: "gold"}}, models.TierGold, 0},
		{"unknown still reported", []eventroProduct{{ProductID: "addon"}, {ProductID: "unknown"}}, "", 1},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			tier, unknown := mappedEventroTier(scenario.products, mapping)
			if len(unknown) != scenario.wantUnknown {
				t.Fatalf("unmapped products = %#v", unknown)
			}
			if scenario.wantTier == "" {
				if tier != nil {
					t.Fatalf("ignored products assigned tier %q", *tier)
				}
			} else if tier == nil || *tier != scenario.wantTier {
				t.Fatalf("tier = %v, want %s", tier, scenario.wantTier)
			}
			for _, product := range unknown {
				if product.ProductID == "addon" {
					t.Fatal("ignored add-on still reported")
				}
			}
		})
	}
	for _, raw := range []string{`{"addon":"ignore"}`, `{"addon":"Ignored"}`, `{"addon":""}`} {
		if _, err := parseEventroProductTiers(raw); err == nil {
			t.Fatalf("invalid mapping accepted: %s", raw)
		}
	}
}
