package controllers

import (
	"encoding/json"
	"net/http"

	"gorm.io/gorm"
)

func writeJSONResponse(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(payload)
}

func writeCreatedJSONResponse(w http.ResponseWriter, payload any) {
	writeJSONResponse(w, http.StatusCreated, payload)
}

func writeDeleteResponse(w http.ResponseWriter, result *gorm.DB, notFoundMessage string) {
	if result.Error != nil {
		http.Error(w, "Delete failed", http.StatusInternalServerError)
		return
	}

	if result.RowsAffected == 0 {
		http.Error(w, notFoundMessage, http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
