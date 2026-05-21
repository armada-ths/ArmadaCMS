package auth

import (
	"ArmadaCMS/main/utils"
	"context"
	"net/http"
	"strings"
)

type contextKey string

const userIDKey contextKey = "user_id"
const roleKey contextKey = "role"
const permissionsKey contextKey = "permissions"

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
		uid, ok := (*claims)["user_id"].(int)
		if !ok {
			http.Error(w, "invalid user_id in token", http.StatusUnauthorized)
			return
		}

		// add user_id to request context
		ctx := context.WithValue(r.Context(), userIDKey, uid)

		// Extract role
		if role, ok := (*claims)["role"].(string); ok {
			ctx = context.WithValue(ctx, roleKey, role)
		}

		// Extract permissions
		if permsRaw, ok := (*claims)["permissions"]; ok {
			if perms, ok := permsRaw.([]string); ok {
				ctx = context.WithValue(ctx, permissionsKey, perms)
			}
		}

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func GetUserIDFromContext(r *http.Request) (int, bool) {
	uid, ok := r.Context().Value(userIDKey).(int)
	return uid, ok
}

func GetPermissionsFromContext(r *http.Request) []string {
	perms, _ := r.Context().Value(permissionsKey).([]string)
	return perms
}

func GetRoleFromContext(r *http.Request) string {
	role, _ := r.Context().Value(roleKey).(string)
	return role
}

// HasPermission checks if the user's permissions include the required one.
// Supported wildcard formats:
//   - "*"            – grants access to every resource and action
//   - "resource.*"   – grants all actions on a specific resource
//   - "*.action"     – grants a specific action on every resource
func HasPermission(perms []string, required string) bool {
	requiredParts := strings.SplitN(required, ".", 2)
	hasResourceAction := len(requiredParts) == 2

	for _, p := range perms {
		if p == "*" || p == required {
			return true
		}
		if hasResourceAction {
			parts := strings.SplitN(p, ".", 2)
			if len(parts) == 2 {
				// "resource.*" grants all actions on that resource
				if parts[0] == requiredParts[0] && parts[1] == "*" {
					return true
				}
				// "*.action" grants that action on all resources
				if parts[0] == "*" && parts[1] == requiredParts[1] {
					return true
				}
			}
		}
	}
	return false
}

// RequirePermission returns middleware that checks the user has a specific permission.
// Used to wrap individual route handlers.
func RequirePermission(permission string, handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		perms := GetPermissionsFromContext(r)
		if !HasPermission(perms, permission) {
			http.Error(w, "forbidden: insufficient permissions", http.StatusForbidden)
			return
		}
		handler(w, r)
	}
}
