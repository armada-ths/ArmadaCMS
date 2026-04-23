package controllers

import (
	"encoding/json"
	"net/http"
)

func writeJSONResponse(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(payload)
}

func writeCreatedJSONResponse(w http.ResponseWriter, payload any) {
	writeJSONResponse(w, http.StatusCreated, payload)
}
