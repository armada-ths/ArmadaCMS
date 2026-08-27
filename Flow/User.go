package Flow

import (
	"ArmadaCMS/main/db"
	"ArmadaCMS/main/models"
	"ArmadaCMS/main/utils"
	"errors"
	"fmt"
	"log"

	"gorm.io/gorm"
)

var ErrInvalidCredentials = errors.New("wrong username or password")

func VerifyLoginWithPassword(username, password string) (*models.Tokens, uint, error) {

	var user models.User
	if err := db.DB.Preload("Roles").Where("username = ?", username).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, 0, ErrInvalidCredentials
		}
		return nil, 0, err
	}
	if err := utils.CheckPasswordHash(password, user.Password); err != nil {
		log.Println(err)
		return nil, 0, ErrInvalidCredentials
	}

	// Collect role names and merge permissions from all assigned roles.
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

	refreshToken, err := utils.GenerateRefreshToken()
	if err != nil {
		return nil, 0, errors.New("not authenticated (2)")
	}

	accessToken, err := utils.GenerateAccessToken(int(user.ID), roleNames, permissions)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to generate access token: %w", err)
	}

	return &models.Tokens{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, user.ID, nil
}
