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

func VerifyLoginWithPassword(username, password string) (*models.Tokens, error) {

	var user models.User
	if err := db.DB.Where("username = ?", username).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("wrong username or password")
		}
		return nil, err
	}
	log.Println(user.Password)
	if err := utils.CheckPasswordHash(password, user.Password); err != nil {
		log.Println(err)
		return nil, errors.New("wrong username or password")
	}

	refreshToken, err := utils.GenerateRefreshToken()
	if err != nil {
		return nil, errors.New("not authenticated (2)")
	}
	if !insertRefreshToken(int(user.ID), refreshToken) {
		return nil, errors.New("not authenticated (3)")
	}

	accessToken, _ := utils.GenerateAccessToken(int(user.ID))

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
