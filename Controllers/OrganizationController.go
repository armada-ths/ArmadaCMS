package controllers

import (
	"ArmadaCMS/main/db"
	"ArmadaCMS/main/models"
	"encoding/json"
	"net/http"
	"sort"
)

// GetOrganizationEndpoint returns the organization structure with its members
func GetOrganizationEndpoint(w http.ResponseWriter, r *http.Request) {
	// Create a sample organization (in production, you would fetch from database)
	// orgGroups := models.GetOrganizationGroups()

	var profiles []models.Profile
	query := db.DB.Model(&models.Profile{})
	query.Preload("Team").Find(&profiles)
	teams := make(map[string][]models.Person)
	for _, profile := range profiles {
		members := teams[profile.Team.TeamName]
		members = append(members, models.ConvertProfileToPerson(profile))
		teams[profile.Team.TeamName] = members
	}
	var groups []models.OrganizationGroup
	for teamName, people := range teams {
		groups = append(groups, models.OrganizationGroup{
			Name:   teamName,
			People: people,
		})
	}
	sort.Slice(groups, func(i, j int) bool {
		return groups[i].Name < groups[j].Name
	})
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(groups)
}
