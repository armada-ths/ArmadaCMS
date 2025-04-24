package controllers

import (
	"ArmadaCMS/main/models"
	"encoding/json"
	"net/http"
)


func GetFairDates(w http.ResponseWriter, r *http.Request) {
    fairDate := models.NewFairDate()
    
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(fairDate)
}