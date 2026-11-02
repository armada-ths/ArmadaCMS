package auth

import (
	"cloud.google.com/go/auth/credentials/idtoken"
	"context"
	"net/http"
	"strings"
	"time"
)

type GoogleIDTokenValidator func(context.Context, string, string) (*idtoken.Payload, error)

// RequireGoogleServiceAccount authenticates machine calls independently of CMS
// user JWTs. Audience and identity come from deployment configuration, not headers.
func RequireGoogleServiceAccount(audience, email string, validate GoogleIDTokenValidator, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		if audience == "" || email == "" || validate == nil {
			http.Error(w, "Worker dispatcher is not configured", http.StatusServiceUnavailable)
			return
		}
		fields := strings.Fields(r.Header.Get("Authorization"))
		if len(fields) != 2 || !strings.EqualFold(fields[0], "Bearer") {
			http.Error(w, "Google ID token required", http.StatusUnauthorized)
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()
		payload, err := validate(ctx, fields[1], audience)
		if err != nil || payload == nil {
			http.Error(w, "Invalid Google ID token", http.StatusUnauthorized)
			return
		}
		verified, _ := payload.Claims["email_verified"].(bool)
		account, _ := payload.Claims["email"].(string)
		if (payload.Issuer != "https://accounts.google.com" && payload.Issuer != "accounts.google.com") || !verified || account != email || payload.Subject == "" {
			http.Error(w, "Unexpected service account", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}
