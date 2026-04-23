package utils

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var jwtSecret = []byte(os.Getenv("jwtsecret_laganda"))

func GenerateRefreshToken() (string, error) {
	bytes := make([]byte, 64)
	_, err := rand.Read(bytes)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

func GenerateAccessToken(userID int, role string, permissions []string) (string, error) {
	claims := jwt.MapClaims{
		"user_id":     userID,
		"role":        role,
		"permissions": permissions,
		"exp":         time.Now().Add(15 * time.Minute).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtSecret)
}

func VerifyAccessToken(tokenString string) (*jwt.MapClaims, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return jwtSecret, nil
	})

	if err != nil {
		log.Println(err)
		return nil, err //!!invalid token!!
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		if exp, ok := claims["exp"].(float64); ok {
			if time.Now().Unix() > int64(exp) {
				return nil, fmt.Errorf("token expired")
			}
		}
		return &claims, nil
	}

	return nil, fmt.Errorf("invalid token claims")
}

func GetUserIdFromAccessToken(tokenString string) *int {
	claims := jwt.MapClaims{}

	new(jwt.Parser).ParseUnverified(tokenString, claims) //nolint:errcheck // intentionally ignores expiration

	UserIDFloat, ok := claims["user_id"].(float64)
	if !ok {
		log.Println("user_id not found or invalid type in token")
		return nil
	}

	UserID := int(UserIDFloat)
	return &UserID
}
