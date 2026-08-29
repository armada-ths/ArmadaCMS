package controllers

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"ArmadaCMS/main/models"
)

func TestRefreshTokenAuditDataExcludesTokenSecret(t *testing.T) {
	token := models.RefreshToken{
		ID:           4,
		RefreshToken: "do-not-log-this-token",
		UserID:       12,
		ValidFrom:    time.Date(2026, 8, 21, 12, 0, 0, 0, time.UTC),
		ValidTo:      time.Date(2026, 8, 28, 12, 0, 0, 0, time.UTC),
		Enabled:      true,
	}

	encoded, err := json.Marshal(newRefreshTokenAuditData(token))
	if err != nil {
		t.Fatalf("marshal audit data: %v", err)
	}
	payload := string(encoded)

	if strings.Contains(payload, token.RefreshToken) || strings.Contains(payload, "refresh_token") {
		t.Fatalf("audit payload contains refresh-token secret or field name: %s", payload)
	}
	if !strings.Contains(payload, `"user_id":12`) || !strings.Contains(payload, `"enabled":true`) {
		t.Fatalf("audit payload is missing expected metadata: %s", payload)
	}
}

func TestTokenResponseHeadersPreventCaching(t *testing.T) {
	response := httptest.NewRecorder()
	setTokenResponseHeaders(response)

	if got := response.Header().Get("Cache-Control"); got != "no-store" {
		t.Fatalf("Cache-Control = %q", got)
	}
	if got := response.Header().Get("Pragma"); got != "no-cache" {
		t.Fatalf("Pragma = %q", got)
	}
}
