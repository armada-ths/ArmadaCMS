package controllers

import (
	"ArmadaCMS/main/Flow"
	"encoding/json"
	"errors"
	"log"
	"net/http"
)

func Login(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var data struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	err := json.NewDecoder(r.Body).Decode(&data)
	if err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	username := data.Username
	password := data.Password
	w.Header().Set("Content-Type", "application/json")

	response, err := Flow.VerifyLoginWithPassword(username, password)
	if err != nil {
		log.Println(err)
		if errors.Is(err, Flow.ErrInvalidCredentials) {
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}

		http.Error(w, "Login failed", http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(response)

}
