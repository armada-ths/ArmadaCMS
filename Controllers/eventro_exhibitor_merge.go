package controllers

import (
	"ArmadaCMS/main/models"
	"encoding/json"
	"fmt"
	"reflect"
	"slices"
	"sort"
	"strings"
)

// A logo and its thumbnail are one field so a manual logo never uses an old thumbnail.
type eventroLogoValue struct {
	Original  *string `json:"original"`
	Thumbnail *string `json:"thumbnail"`
}

func normalizedEventroLocations(values []string) []string {
	result := []string{}
	seen := map[string]bool{}
	for _, value := range values {
		value = strings.Join(strings.Fields(value), " ")
		key := strings.ToLower(value)
		if value != "" && !seen[key] {
			seen[key] = true
			result = append(result, value)
		}
	}
	sort.Strings(result)
	return result
}

// Sort and deduplicate exact labels for order-independent field comparisons.
func sortedEventroCategoryLabels(values []string) []string {
	labels := make([]string, 0, len(values))
	for _, label := range values {
		if strings.TrimSpace(label) != "" {
			labels = append(labels, label)
		}
	}
	slices.Sort(labels)
	return slices.Compact(labels)
}

func exhibitorSyncValues(e models.Exhibitor) map[string]json.RawMessage {
	industries, employments, programs := []string{}, []string{}, []string{}
	for _, v := range e.Industries {
		industries = append(industries, v.Name)
	}
	for _, v := range e.Employments {
		employments = append(employments, v.Name)
	}
	for _, v := range e.Programs {
		programs = append(programs, v.Name)
	}
	values := map[string]any{
		"name": e.Name, "tier": e.Tier, "companyWebsite": e.CompanyWebsite,
		"about": e.About, "logo": eventroLogoValue{e.LogoFreesizeUrl, e.LogoThumbnail},
		"cities": e.Cities, "industries": sortedEventroCategoryLabels(industries),
		"employments": sortedEventroCategoryLabels(employments), "programs": sortedEventroCategoryLabels(programs),
	}
	result := map[string]json.RawMessage{}
	for key, value := range values {
		result[key], _ = json.Marshal(value)
	}
	return result
}

func sameEventroValue(a, b json.RawMessage) bool {
	var x, y any
	if json.Unmarshal(a, &x) != nil || json.Unmarshal(b, &y) != nil {
		return false
	}
	return reflect.DeepEqual(x, y)
}

func emptyEventroValue(value json.RawMessage) bool {
	return string(value) == "null" || string(value) == `""` || string(value) == "[]" || string(value) == `{"original":null,"thumbnail":null}`
}

// mergeEventroExhibitor advances the source baseline while retaining explicit CMS ownership.
// Unknown historical differences are treated as CMS corrections on first import.
func mergeEventroExhibitor(current, incoming models.Exhibitor, state *models.EventroSyncState, isNew, syncCategories bool) models.Exhibitor {
	merged := current
	// Adding a thumbnail for the same original does not replace a CMS correction.
	if current.LogoThumbnail == nil && current.LogoFreesizeUrl != nil && incoming.LogoFreesizeUrl != nil && *current.LogoFreesizeUrl == *incoming.LogoFreesizeUrl {
		current.LogoThumbnail = incoming.LogoThumbnail
		merged.LogoThumbnail = incoming.LogoThumbnail
	}
	if isNew {
		merged = incoming
	}
	cms, source := exhibitorSyncValues(current), exhibitorSyncValues(incoming)
	if incoming.Tier == nil {
		delete(source, "tier")
	}
	var preserveOmitted []string
	if !syncCategories {
		preserveOmitted = []string{"industries", "employments", "programs"}
		for _, field := range preserveOmitted {
			delete(source, field)
			copyEventroExhibitorField(&merged, current, field)
		}
	}
	for field := range mergeEventroValues(cms, source, state, isNew, preserveOmitted...) {
		copyEventroExhibitorField(&merged, incoming, field)
	}
	return merged
}

func copyEventroExhibitorField(to *models.Exhibitor, from models.Exhibitor, field string) {
	switch field {
	case "name":
		to.Name = from.Name
	case "tier":
		to.Tier = from.Tier
	case "companyWebsite":
		to.CompanyWebsite = from.CompanyWebsite
	case "about":
		to.About = from.About
	case "logo":
		to.LogoFreesizeUrl, to.LogoThumbnail = from.LogoFreesizeUrl, from.LogoThumbnail
	case "cities":
		to.Cities = from.Cities
	case "industries":
		to.Industries = from.Industries
	case "employments":
		to.Employments = from.Employments
	case "programs":
		to.Programs = from.Programs
	}
}

func applyEventroReviewValue(e *models.Exhibitor, field string, value json.RawMessage) error {
	switch field {
	case "name":
		return json.Unmarshal(value, &e.Name)
	case "tier":
		return json.Unmarshal(value, &e.Tier)
	case "companyWebsite":
		return json.Unmarshal(value, &e.CompanyWebsite)
	case "about":
		return json.Unmarshal(value, &e.About)
	case "cities":
		return json.Unmarshal(value, &e.Cities)
	case "logo":
		var logo eventroLogoValue
		if err := json.Unmarshal(value, &logo); err != nil {
			return err
		}
		e.LogoFreesizeUrl, e.LogoThumbnail = logo.Original, logo.Thumbnail
		return nil
	case "industries", "employments", "programs":
		var labels []string
		if err := json.Unmarshal(value, &labels); err != nil {
			return err
		}
		switch field {
		case "industries":
			e.Industries = []models.Industry{}
			for _, label := range labels {
				e.Industries = append(e.Industries, models.Industry{Name: label})
			}
		case "employments":
			e.Employments = []models.Employment{}
			for _, label := range labels {
				e.Employments = append(e.Employments, models.Employment{Name: label})
			}
		case "programs":
			e.Programs = []models.Program{}
			for _, label := range labels {
				e.Programs = append(e.Programs, models.Program{Name: label})
			}
		}
		return nil
	}
	return fmt.Errorf("unsupported field %q", field)
}
