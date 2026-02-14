package models

import "strings"

// Person represents an individual in the organization
type Person struct {
	ID          int     `json:"id"`
	Name        string  `json:"name"`
	Rank        *string `json:"rank"`
	Email       *string `json:"email"`
	Picture     *string `json:"picture"`
	LinkedInURL *string `json:"linkedin_url"`
	Role        string  `json:"role"`
}

// OrganizationGroup represents a single organization group with its members
type OrganizationGroup struct {
	Name   string   `json:"name"`
	People []Person `json:"people"`
}

// profile is backend goadmin struct, person is frontend interface
func ConvertProfileToPerson(profile Profile) Person {
	optional := func(value string) *string {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			return nil
		}
		return &trimmed
	}

	return Person{
		ID:          int(profile.ID),
		Name:        profile.Name,
		Rank:        optional(profile.Rank),
		Email:       optional(profile.Email),
		Picture:     optional(profile.Photo),
		LinkedInURL: optional(profile.Linkedin),
		Role:        profile.Title,
	}
}
