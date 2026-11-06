package controllers

import (
	"ArmadaCMS/main/models"
	"net/url"
	"testing"
)

func TestLoungeStatusOnlyMutation(t *testing.T) {
	description := "Description"
	max := int64(5)
	item := models.LoungeItem{Name: "Coffee", CategoryID: 1, Description: &description, SortOrder: 2, MaxPerOrder: &max}
	updates, err := applyLoungeFields(url.Values{"soldOut": {"true"}}, &item)
	if err != nil || !item.SoldOut || len(updates) != 1 {
		t.Fatalf("status-only mutation failed: %v, %v", updates, err)
	}
	if *item.Description != description || *item.MaxPerOrder != max || item.SortOrder != 2 {
		t.Fatal("status change modified unrelated fields")
	}
	updates, err = applyLoungeFields(url.Values{"soldOut": {"false"}}, &item)
	if err != nil || item.SoldOut || updates["sold_out"] != false {
		t.Fatal("mark available did not persist false")
	}
}

func TestLoungeOptionalFields(t *testing.T) {
	text := "Previous"
	max := int64(3)
	item := models.LoungeItem{Name: "Coffee", NameSv: &text, Description: &text, MaxPerOrder: &max}
	_, err := applyLoungeFields(url.Values{"nameSv": {""}, "description": {""}, "maxPerOrder": {""}}, &item)
	if err != nil || item.NameSv != nil || item.Description != nil || item.MaxPerOrder != nil {
		t.Fatal("optional fields were not cleared")
	}
	_, err = applyLoungeFields(url.Values{"name": {" Coffee "}, "nameSv": {" Kaffe "}, "sortOrder": {"0"}}, &item)
	if err != nil || item.Name != "Coffee" || *item.NameSv != "Kaffe" || item.SortOrder != 0 {
		t.Fatal("fields were not normalized")
	}
}

func TestLoungeInvalidFields(t *testing.T) {
	for _, values := range []url.Values{
		{"name": {" "}}, {"sortOrder": {"-1"}}, {"sortOrder": {"1.5"}},
		{"maxPerOrder": {"0"}}, {"maxPerOrder": {"9007199254740992"}},
		{"soldOut": {"yes"}}, {"categoryId": {"0"}},
	} {
		item := models.LoungeItem{Name: "Coffee"}
		if _, err := applyLoungeFields(values, &item); err == nil {
			t.Fatalf("accepted invalid fields %v", values)
		}
	}
}
