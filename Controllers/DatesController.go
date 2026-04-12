package controllers

import (
	"ArmadaCMS/main/db"
	"ArmadaCMS/main/models"
	"encoding/json"
	"net/http"
)

// GetFairDates returns the fair dates in the nested JSON format expected by
// the public armada.nu frontend. It reads from the database (latest record)
// and falls back to hardcoded defaults if no record exists yet.
// @Summary Get public fair dates
// @Tags public
// @Produce json
// @Success 200 {object} models.FairDate
// @Router /dates [get]
func GetFairDates(w http.ResponseWriter, r *http.Request) {
	var config models.FairDateConfig
	result := db.DB.Order("id desc").First(&config)
	if result.Error != nil {
		http.Error(w, "No fair date configuration found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(config.ToFairDate())
}
