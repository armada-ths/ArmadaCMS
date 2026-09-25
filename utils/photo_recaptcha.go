package utils

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

var recaptchaClient = &http.Client{Timeout: 8 * time.Second}

func GoogleAccessToken(ctx context.Context) (string, error) {
	metadata, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://metadata.google.internal/computeMetadata/v1/instance/service-accounts/default/token", nil)
	if err != nil {
		return "", err
	}
	metadata.Header.Set("Metadata-Flavor", "Google")
	response, err := recaptchaClient.Do(metadata)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return "", errors.New("could not get workload identity")
	}
	var access struct {
		Token string `json:"access_token"`
	}
	if err := json.NewDecoder(response.Body).Decode(&access); err != nil {
		return "", err
	}
	if access.Token == "" {
		return "", errors.New("empty workload identity token")
	}
	return access.Token, nil
}

func VerifyPhotoRecaptcha(ctx context.Context, token string) error {
	project := strings.TrimSpace(os.Getenv("RECAPTCHA_PROJECT_ID"))
	siteKey := strings.TrimSpace(os.Getenv("PHOTO_RECAPTCHA_SITE_KEY"))
	if project == "" || siteKey == "" || token == "" {
		return errors.New("photo reCAPTCHA is not configured")
	}
	accessToken, err := GoogleAccessToken(ctx)
	if err != nil {
		return err
	}
	body, _ := json.Marshal(map[string]any{"event": map[string]string{"token": token, "siteKey": siteKey, "expectedAction": "photo_upload"}})
	endpoint := "https://recaptchaenterprise.googleapis.com/v1/projects/" + url.PathEscape(project) + "/assessments"
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return err
	}
	request.Header.Set("Authorization", "Bearer "+accessToken)
	request.Header.Set("Content-Type", "application/json")
	result, err := recaptchaClient.Do(request)
	if err != nil {
		return err
	}
	defer result.Body.Close()
	if result.StatusCode != http.StatusOK {
		return fmt.Errorf("reCAPTCHA assessment failed: %d", result.StatusCode)
	}
	var assessment struct {
		TokenProperties struct {
			Valid    bool   `json:"valid"`
			Action   string `json:"action"`
			Hostname string `json:"hostname"`
		} `json:"tokenProperties"`
		RiskAnalysis struct {
			Score float64 `json:"score"`
		} `json:"riskAnalysis"`
	}
	if err := json.NewDecoder(result.Body).Decode(&assessment); err != nil {
		return err
	}
	host := assessment.TokenProperties.Hostname
	allowedHost := host == "photos.armada.nu"
	for _, configured := range strings.Split(os.Getenv("PHOTO_RECAPTCHA_HOSTNAMES"), ",") {
		if host != "" && host == strings.TrimSpace(configured) {
			allowedHost = true
		}
	}
	if assessment.TokenProperties.Valid && assessment.TokenProperties.Action == "photo_upload" && assessment.RiskAnalysis.Score >= 0.5 && (allowedHost || (os.Getenv("PHOTO_ALLOW_LOCALHOST_RECAPTCHA") == "true" && host == "localhost")) {
		return nil
	}
	return errors.New("photo reCAPTCHA assessment rejected")
}
