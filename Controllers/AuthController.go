package controllers

import (
	"ArmadaCMS/main/Flow"
	"ArmadaCMS/main/audit"
	"ArmadaCMS/main/auth"
	"ArmadaCMS/main/db"
	"ArmadaCMS/main/models"
	"ArmadaCMS/main/utils"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"
	"time"

	"gorm.io/gorm"
)

type refreshTokenAuditData struct {
	ID        uint      `json:"id"`
	UserID    uint      `json:"user_id"`
	ValidFrom time.Time `json:"valid_from"`
	ValidTo   time.Time `json:"valid_to"`
	Enabled   bool      `json:"enabled"`
}

func newRefreshTokenAuditData(token models.RefreshToken) refreshTokenAuditData {
	return refreshTokenAuditData{
		ID:        token.ID,
		UserID:    token.UserID,
		ValidFrom: token.ValidFrom,
		ValidTo:   token.ValidTo,
		Enabled:   token.Enabled,
	}
}

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
	setTokenResponseHeaders(w)

	response, userID, err := Flow.VerifyLoginWithPassword(username, password)
	if err != nil {
		log.Println(err)
		if errors.Is(err, Flow.ErrInvalidCredentials) {
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}

		http.Error(w, "Login failed", http.StatusInternalServerError)
		return
	}

	now := time.Now()
	refreshToken := models.RefreshToken{
		RefreshToken: utils.HashRefreshToken(response.RefreshToken),
		UserID:       userID,
		ValidFrom:    now,
		ValidTo:      now.Add(7 * 24 * time.Hour),
		Enabled:      true,
	}
	auditRequest := auth.WithUserID(r, int(userID))
	if err := db.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&refreshToken).Error; err != nil {
			return err
		}
		return audit.LogCreate(tx, auditRequest, "refreshtokens", refreshToken.ID, newRefreshTokenAuditData(refreshToken))
	}); err != nil {
		log.Println("Login: failed to persist refresh token:", err)
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
// @Router /refreshAccessToken [post]
func RefreshAccessToken(w http.ResponseWriter, r *http.Request) {
	setTokenResponseHeaders(w)
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
		"refresh_token = ? AND enabled = true AND valid_to > ?", utils.HashRefreshToken(tokenStr), time.Now(),
	).First(&rt).Error; err != nil {
		http.Error(w, "invalid or expired refresh token", http.StatusUnauthorized)
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
		RefreshToken: utils.HashRefreshToken(newRefreshTokenStr),
		UserID:       rt.UserID,
		ValidFrom:    time.Now(),
		ValidTo:      time.Now().Add(7 * 24 * time.Hour),
		Enabled:      true,
	}

	accessToken, err := utils.GenerateAccessToken(int(rt.UserID), roleNames, permissions)
	if err != nil {
		http.Error(w, "token refresh failed", http.StatusInternalServerError)
		return
	}

	auditRequest := auth.WithUserID(r, int(rt.UserID))
	if err := db.DB.Transaction(func(tx *gorm.DB) error {
		group, err := audit.StartGroup(tx, auditRequest, "update", "sessions", rt.UserID, map[string]any{
			"operation": "refresh_token_rotation",
		})
		if err != nil {
			return err
		}
		childRequest := audit.WithParent(auditRequest, group.ID)

		oldToken := newRefreshTokenAuditData(rt)
		if err := tx.Model(&rt).Update("enabled", false).Error; err != nil {
			return err
		}
		if err := audit.LogUpdate(tx, childRequest, "refreshtokens", rt.ID, oldToken, newRefreshTokenAuditData(rt)); err != nil {
			return err
		}
		if err := tx.Create(&newRT).Error; err != nil {
			return err
		}
		if err := audit.LogCreate(tx, childRequest, "refreshtokens", newRT.ID, newRefreshTokenAuditData(newRT)); err != nil {
			return err
		}
		return audit.FinalizeGroup(tx, group.ID, "completed", map[string]any{
			"operation": "refresh_token_rotation",
			"revoked":   1,
			"created":   1,
		})
	}); err != nil {
		log.Println("RefreshAccessToken: failed to rotate token:", err)
		http.Error(w, "token refresh failed", http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(models.Tokens{
		AccessToken:  accessToken,
		RefreshToken: newRefreshTokenStr,
	})
}

func setTokenResponseHeaders(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Pragma", "no-cache")
}

func revokeUserRefreshTokensWithAudit(tx *gorm.DB, r *http.Request, userID uint) error {
	var tokens []models.RefreshToken
	if err := tx.Where("user_id = ? AND enabled = true", userID).Find(&tokens).Error; err != nil {
		return err
	}

	for i := range tokens {
		before := newRefreshTokenAuditData(tokens[i])
		if err := tx.Model(&tokens[i]).Update("enabled", false).Error; err != nil {
			return err
		}
		tokens[i].Enabled = false
		if err := audit.LogUpdate(tx, r, "refreshtokens", tokens[i].ID, before, newRefreshTokenAuditData(tokens[i])); err != nil {
			return err
		}
	}

	return nil
}
