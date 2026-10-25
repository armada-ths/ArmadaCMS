package utils

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
)

// Opt-in live check of the local key and API authentication; creates no photos.
func TestPhotoRecaptchaEnterpriseIntegration(t *testing.T) {
	if os.Getenv("PHOTO_RECAPTCHA_INTEGRATION_TEST") != "true" {
		t.Skip("set PHOTO_RECAPTCHA_INTEGRATION_TEST=true in the configured local backend")
	}
	if os.Getenv("K_SERVICE") != "" {
		t.Fatal("this integration check is for the local development backend only")
	}
	err := VerifyPhotoRecaptcha(context.Background(), "invalid-integration-test-token")
	if err == nil || err.Error() != "photo reCAPTCHA assessment rejected" {
		t.Fatalf("expected a successful API assessment rejecting an invalid browser token; got %v", err)
	}
}

type photoRecaptchaTransport func(*http.Request) (*http.Response, error)

func (fn photoRecaptchaTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	return fn(request)
}

func TestPhotoRecaptchaAssessment(t *testing.T) {
	for _, hosted := range []bool{false, true} {
		name := "local API key"
		if hosted {
			name = "Cloud Run workload identity"
		}
		t.Run(name, func(t *testing.T) {
			t.Setenv("RECAPTCHA_PROJECT_ID", "test-project")
			t.Setenv("PHOTO_RECAPTCHA_SITE_KEY", "test-site-key")
			t.Setenv("PHOTO_RECAPTCHA_API_KEY", "local-test-api-key")
			t.Setenv("PHOTO_RECAPTCHA_HOSTNAMES", "192.168.1.119,localhost")
			t.Setenv("K_SERVICE", "")
			if hosted {
				t.Setenv("K_SERVICE", "armadacms")
			}
			originalClient := recaptchaClient
			t.Cleanup(func() { recaptchaClient = originalClient })
			assessmentJSON := `{"tokenProperties":{"valid":true,"action":"photo_upload","hostname":"192.168.1.119"},"riskAnalysis":{"score":1}}`
			metadataCalls := 0
			recaptchaClient = &http.Client{Transport: photoRecaptchaTransport(func(request *http.Request) (*http.Response, error) {
				response := assessmentJSON
				if request.URL.Host == "metadata.google.internal" {
					metadataCalls++
					response = `{"access_token":"test-workload-token"}`
				} else {
					if request.URL.String() != "https://recaptchaenterprise.googleapis.com/v1/projects/test-project/assessments" {
						t.Fatalf("unexpected assessment endpoint: %s", request.URL)
					}
					if hosted {
						if request.Header.Get("Authorization") != "Bearer test-workload-token" || request.Header.Get("X-Goog-Api-Key") != "" {
							t.Fatal("hosted assessments must use workload identity, ignoring the local API key")
						}
					} else if request.Header.Get("X-Goog-Api-Key") != "local-test-api-key" || request.Header.Get("Authorization") != "" {
						t.Fatal("local assessment must use the dedicated API key")
					}
					var payload struct {
						Event struct{ Token, SiteKey, ExpectedAction string }
					}
					if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
						t.Fatal(err)
					}
					if payload.Event.Token != "browser-token" || payload.Event.SiteKey != "test-site-key" || payload.Event.ExpectedAction != "photo_upload" {
						t.Fatalf("unexpected assessment payload: %+v", payload)
					}
				}
				return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(response)), Header: make(http.Header)}, nil
			})}
			if err := VerifyPhotoRecaptcha(context.Background(), "browser-token"); err != nil {
				t.Fatal(err)
			}
			if hosted && metadataCalls != 1 {
				t.Fatal("hosted assessments did not use metadata")
			}
			if !hosted && metadataCalls != 0 {
				t.Fatal("local assessments must not use metadata")
			}
			for _, rejected := range []string{
				`{"tokenProperties":{"valid":false,"action":"photo_upload","hostname":"192.168.1.119"},"riskAnalysis":{"score":1}}`,
				`{"tokenProperties":{"valid":true,"action":"wrong_action","hostname":"192.168.1.119"},"riskAnalysis":{"score":1}}`,
				`{"tokenProperties":{"valid":true,"action":"photo_upload","hostname":"untrusted.example"},"riskAnalysis":{"score":1}}`,
				`{"tokenProperties":{"valid":true,"action":"photo_upload","hostname":"192.168.1.119"},"riskAnalysis":{"score":0.1}}`,
			} {
				assessmentJSON = rejected
				if err := VerifyPhotoRecaptcha(context.Background(), "browser-token"); err == nil {
					t.Fatalf("accepted invalid assessment: %s", rejected)
				}
			}
		})
	}
}
