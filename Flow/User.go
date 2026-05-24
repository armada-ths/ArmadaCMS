package Flow

import (
	"ArmadaCMS/main/db"
	"ArmadaCMS/main/models"
	"ArmadaCMS/main/utils"
	"errors"
	"log"
	"time"

	"gorm.io/gorm"
)

var ErrInvalidCredentials = errors.New("wrong username or password")

func VerifyLoginWithPassword(username, password string) (*models.Tokens, error) {

	var user models.User
	if err := db.DB.Preload("Roles").Where("username = ?", username).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrInvalidCredentials
		}
		return nil, err
	}
	if err := utils.CheckPasswordHash(password, user.Password); err != nil {
		log.Println(err)
		return nil, ErrInvalidCredentials
	}

	// Collect role names and merge permissions from all assigned roles.
	roleNames := make([]string, 0, len(user.Roles))
	seen := make(map[string]struct{})
	var permissions []string
	for _, role := range user.Roles {
		roleNames = append(roleNames, role.Name)
		for _, p := range role.Permissions {
			if _, exists := seen[p]; !exists {
				seen[p] = struct{}{}
				permissions = append(permissions, p)
			}
		}
	}

	refreshToken, err := utils.GenerateRefreshToken()
	if err != nil {
		return nil, errors.New("not authenticated (2)")
	}
	if !insertRefreshToken(int(user.ID), refreshToken) {
		return nil, errors.New("not authenticated (3)")
	}

	accessToken, _ := utils.GenerateAccessToken(int(user.ID), roleNames, permissions)

	return &models.Tokens{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil

}
func insertRefreshToken(userId int, refreshToken string) bool {
	if userId == 0 {
		return false
	}

	token := models.RefreshToken{
		RefreshToken: refreshToken,
		UserID:       uint(userId),
		ValidFrom:    time.Now(),
		ValidTo:      time.Now().Add(7 * 24 * time.Hour), // +7 days
		Enabled:      true,
	}

	if err := db.DB.Create(&token).Error; err != nil {
		log.Println("InsertRefreshToken error:", err)
		return false
	}

	return true
}
