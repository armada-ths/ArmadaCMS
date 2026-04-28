package utils

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var jwtSecret = mustLoadJWTSecret()

// mustLoadJWTSecret loads the JWT signing secret from the jwtsecret_laganda
// environment variable and terminates the application if it is missing or empty.
func mustLoadJWTSecret() []byte {
	secret := strings.TrimSpace(os.Getenv("jwtsecret_laganda"))
	if secret == "" {
		log.Fatal("missing required environment variable jwtsecret_laganda: JWT secret must be non-empty")
	}
	return []byte(secret)
}

func GenerateRefreshToken() (string, error) {
	bytes := make([]byte, 64)
	_, err := rand.Read(bytes)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// accessClaims represents the custom JWT claims embedded in access tokens,
// combining application-specific fields with standard registered claims.
type accessClaims struct {
	UserID      int      `json:"user_id"`
	Role        string   `json:"role"`
	Permissions []string `json:"permissions"`
	jwt.RegisteredClaims
}

func GenerateAccessToken(userID int, role string, permissions []string) (string, error) {
	claims := accessClaims{
		UserID:      userID,
		Role:        role,
		Permissions: permissions,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(15 * time.Minute)),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtSecret)
}

func VerifyAccessToken(tokenString string) (*jwt.MapClaims, error) {
	claims := &accessClaims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return jwtSecret, nil
	})

	if err != nil {
		log.Println(err)
		return nil, err
	}

	if token.Valid {
		result := jwt.MapClaims{
			"user_id":     claims.UserID,
			"role":        claims.Role,
			"permissions": claims.Permissions,
		}
		if claims.ExpiresAt != nil {
			result["exp"] = claims.ExpiresAt.Unix()
		}
		return &result, nil
	}

	return nil, fmt.Errorf("invalid token claims")
}

func GetUserIdFromAccessToken(tokenString string) *int {
	claims := jwt.MapClaims{}

	_, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return jwtSecret, nil
	}, jwt.WithoutClaimsValidation()) // intentionally ignores expiration/claims validation, but still verifies signature
	if err != nil {
		log.Println(err)
		return nil
	}

	UserIDFloat, ok := claims["user_id"].(float64)
	if !ok {
		log.Println("user_id not found or invalid type in token")
		return nil
	}

	UserID := int(UserIDFloat)
	return &UserID
}
