package controllers

import (
	"ArmadaCMS/main/Flow"
	"ArmadaCMS/main/db"
	"ArmadaCMS/main/models"
	"ArmadaCMS/main/utils"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"
	"time"
)

// loginRequest is the body for the Login endpoint.
type loginRequest struct {
	Username string `json:"username" example:"admin"`
	Password string `json:"password" example:"s3cr3t"`
}

// Login authenticates a user and returns access and refresh tokens.
// @Summary Login
// @Tags auth
// @Accept json
// @Produce json
// @Param body body controllers.loginRequest true "Credentials"
// @Success 200 {object} models.Tokens
// @Failure 400 {string} string "Invalid JSON"
// @Failure 401 {string} string "Wrong username or password"
// @Failure 500 {string} string "Login failed"
// @Router /login [post]
func Login(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var data loginRequest
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

// RefreshAccessToken exchanges a valid refresh token for a new access/refresh token pair.
// @Summary Refresh access token
// @Tags auth
// @Produce json
// @Success 200 {object} models.Tokens
// @Failure 401 {string} string "Invalid or expired refresh token"
// @Failure 500 {string} string "Token refresh failed"
// @Router /refreshAccessToken [get]
func RefreshAccessToken(w http.ResponseWriter, r *http.Request) {
	header := r.Header.Get("X-RefreshAuthorization")
	if header == "" {
		http.Error(w, "missing X-RefreshAuthorization header", http.StatusUnauthorized)
		return
	}

	parts := strings.SplitN(header, " ", 2)
	if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
		http.Error(w, "invalid X-RefreshAuthorization format", http.StatusUnauthorized)
		return
	}
	tokenStr := parts[1]

	var rt models.RefreshToken
	if err := db.DB.Preload("User.Roles").Where(
		"refresh_token = ? AND enabled = true AND valid_to > ?", tokenStr, time.Now(),
	).First(&rt).Error; err != nil {
		http.Error(w, "invalid or expired refresh token", http.StatusUnauthorized)
		return
	}

	// Invalidate used refresh token (rotation)
	if err := db.DB.Model(&rt).Update("enabled", false).Error; err != nil {
		log.Println("RefreshAccessToken: failed to disable old token:", err)
		http.Error(w, "token refresh failed", http.StatusInternalServerError)
		return
	}

	user := rt.User
	roleNames := make([]string, 0, len(user.Roles))
	seen := make(map[string]struct{})
	permissions := make([]string, 0)
	for _, role := range user.Roles {
		roleNames = append(roleNames, role.Name)
		for _, p := range role.Permissions {
			if _, exists := seen[p]; !exists {
				seen[p] = struct{}{}
				permissions = append(permissions, p)
			}
		}
	}

	newRefreshTokenStr, err := utils.GenerateRefreshToken()
	if err != nil {
		http.Error(w, "token refresh failed", http.StatusInternalServerError)
		return
	}

	newRT := models.RefreshToken{
		RefreshToken: newRefreshTokenStr,
		UserID:       rt.UserID,
		ValidFrom:    time.Now(),
		ValidTo:      time.Now().Add(7 * 24 * time.Hour),
		Enabled:      true,
	}
	if err := db.DB.Create(&newRT).Error; err != nil {
		log.Println("RefreshAccessToken: failed to insert new token:", err)
		http.Error(w, "token refresh failed", http.StatusInternalServerError)
		return
	}

	accessToken, err := utils.GenerateAccessToken(int(rt.UserID), roleNames, permissions)
	if err != nil {
		http.Error(w, "token refresh failed", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(models.Tokens{
		AccessToken:  accessToken,
		RefreshToken: newRefreshTokenStr,
	})
}
