package controllers

import (
	"ArmadaCMS/main/models"
	"encoding/json"
	"testing"
)

func TestLoungeStatusOnlyMutation(t *testing.T) {
	description := "Description"
	max := int64(5)
	item := models.LoungeItem{Name: "Coffee", CategoryID: 1, Description: &description, SortOrder: 2, MaxPerOrder: &max}
	updates, err := applyLoungeFields(map[string]json.RawMessage{"soldOut": json.RawMessage("true")}, &item)
	if err != nil || !item.SoldOut || len(updates) != 1 {
		t.Fatalf("status-only mutation failed: %v, %v", updates, err)
	}
	if *item.Description != description || *item.MaxPerOrder != max || item.SortOrder != 2 {
		t.Fatal("status change modified unrelated fields")
	}
	updates, err = applyLoungeFields(map[string]json.RawMessage{"soldOut": json.RawMessage("false")}, &item)
	if err != nil || item.SoldOut || updates["sold_out"] != false {
		t.Fatal("mark available did not persist false")
	}
}

func TestLoungeOptionalFields(t *testing.T) {
	text := "Previous"
	max := int64(3)
	item := models.LoungeItem{Name: "Coffee", NameSv: &text, Description: &text, MaxPerOrder: &max}
	_, err := applyLoungeFields(map[string]json.RawMessage{"nameSv": json.RawMessage("null"), "description": json.RawMessage(`""`), "maxPerOrder": json.RawMessage("null")}, &item)
	if err != nil || item.NameSv != nil || item.Description != nil || item.MaxPerOrder != nil {
		t.Fatal("optional fields were not cleared")
	}
	_, err = applyLoungeFields(map[string]json.RawMessage{"name": json.RawMessage(`" Coffee "`), "nameSv": json.RawMessage(`" Kaffe "`)}, &item)
	if err != nil || item.Name != "Coffee" || *item.NameSv != "Kaffe" || item.SortOrder != 0 {
		t.Fatal("fields were not normalized")
	}
}

func TestLoungeInvalidFields(t *testing.T) {
	for _, raw := range []string{
		`{"name":" "}`, `{"name":null}`, `{"maxPerOrder":0}`, `{"maxPerOrder":9007199254740992}`,
		`{"soldOut":"yes"}`, `{"soldOut":null}`, `{"categoryId":0}`, `{"categoryId":1.5}`,
	} {
		var fields map[string]json.RawMessage
		if err := json.Unmarshal([]byte(raw), &fields); err != nil {
			t.Fatal(err)
		}
		item := models.LoungeItem{Name: "Coffee"}
		if _, err := applyLoungeFields(fields, &item); err == nil {
			t.Fatalf("accepted invalid fields %s", raw)
		}
	}
}
