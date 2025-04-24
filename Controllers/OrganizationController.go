package controllers

import (
	"ArmadaCMS/main/models"
	"encoding/json"
	"net/http"
)

// GetOrganization returns the organization structure with its members
func GetOrganization(w http.ResponseWriter, r *http.Request) {
    // Create a sample organization (in production, you would fetch from database)
    orgGroups := models.GetOrganizationGroups()

    // Set headers and return response
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(orgGroups)
}