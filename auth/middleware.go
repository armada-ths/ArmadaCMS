package auth

import (
	"ArmadaCMS/main/utils"
	"context"
	"net/http"
	"strings"
)

type contextKey string

const userIDKey contextKey = "user_id"

func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, "missing Authorization header", http.StatusUnauthorized)
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			http.Error(w, "invalid Authorization header format", http.StatusUnauthorized)
			return
		}
		tokenStr := parts[1]

		claims, err := utils.VerifyAccessToken(tokenStr)
		if err != nil {
			http.Error(w, "invalid or expired token", http.StatusUnauthorized)
			return
		}

		// extract user_id
		uid, ok := (*claims)["user_id"].(float64)
		if !ok {
			http.Error(w, "invalid user_id in token", http.StatusUnauthorized)
			return
		}

		// add user_id to request context
		ctx := context.WithValue(r.Context(), userIDKey, int(uid))
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func GetUserIDFromContext(r *http.Request) (int, bool) {
	uid, ok := r.Context().Value(userIDKey).(int)
	return uid, ok
}
