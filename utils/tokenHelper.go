package utils

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// loadJWTSecret loads the JWT signing secret from the jwtsecret_laganda
// environment variable.
func loadJWTSecret() ([]byte, error) {
	secret := strings.TrimSpace(os.Getenv("jwtsecret_laganda"))
	if secret == "" {
		return nil, fmt.Errorf("missing required environment variable jwtsecret_laganda: JWT secret must be non-empty")
	}
	return []byte(secret), nil
}

// ValidateJWTSecret verifies that the JWT signing secret is configured.
func ValidateJWTSecret() error {
	_, err := loadJWTSecret()
	return err
}

func GenerateRefreshToken() (string, error) {
	bytes := make([]byte, 64)
	_, err := rand.Read(bytes)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// HashRefreshToken returns the one-way representation stored in the database.
// The tokens are random and high-entropy, so a fast digest is safe here and
// allows equality lookups without retaining reusable credentials at rest.
func HashRefreshToken(token string) string {
	digest := sha256.Sum256([]byte(token))
	return hex.EncodeToString(digest[:])
}

// accessClaims represents the custom JWT claims embedded in access tokens,
// combining application-specific fields with standard registered claims.
type accessClaims struct {
	UserID      int      `json:"user_id"`
	Roles       []string `json:"roles"`
	Permissions []string `json:"permissions"`
	jwt.RegisteredClaims
}

func GenerateAccessToken(userID int, roles []string, permissions []string) (string, error) {
	jwtSecret, err := loadJWTSecret()
	if err != nil {
		return "", err
	}

	if roles == nil {
		roles = []string{}
	}

	claims := accessClaims{
		UserID:      userID,
		Roles:       roles,
		Permissions: permissions,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(15 * time.Minute)),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtSecret)
}

func VerifyAccessToken(tokenString string) (*jwt.MapClaims, error) {
	jwtSecret, err := loadJWTSecret()
	if err != nil {
		log.Println(err)
		return nil, err
	}

	claims := &accessClaims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		if token.Method.Alg() != jwt.SigningMethodHS256.Alg() {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return jwtSecret, nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}))

	if err != nil {
		log.Println(err)
		return nil, err
	}

	if token.Valid {
		result := jwt.MapClaims{
			"user_id":     claims.UserID,
			"roles":       claims.Roles,
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
	jwtSecret, err := loadJWTSecret()
	if err != nil {
		log.Println(err)
		return nil
	}

	claims := jwt.MapClaims{}

	_, err = jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		if token.Method.Alg() != jwt.SigningMethodHS256.Alg() {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return jwtSecret, nil
	}, jwt.WithoutClaimsValidation(), jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()})) // intentionally ignores expiration/claims validation, but still verifies signature
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
