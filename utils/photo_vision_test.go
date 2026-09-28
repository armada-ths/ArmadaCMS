package utils

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func lowRiskLikelihoods() map[string]string {
	return map[string]string{
		"adult": "VERY_UNLIKELY", "spoof": "UNLIKELY", "medical": "VERY_UNLIKELY",
		"violence": "UNLIKELY", "racy": "VERY_UNLIKELY",
	}
}

func TestPhotoSafeSearchVerdict(t *testing.T) {
	if got := PhotoSafeSearchVerdict(lowRiskLikelihoods()); got != "safe" {
		t.Fatalf("low-risk image verdict = %q", got)
	}
	for _, category := range photoVisionCategories {
		for _, likelihood := range []string{"POSSIBLE", "LIKELY", "VERY_LIKELY", "UNKNOWN", ""} {
			t.Run(category+"/"+likelihood, func(t *testing.T) {
				values := lowRiskLikelihoods()
				values[category] = likelihood
				if got := PhotoSafeSearchVerdict(values); got != "review" {
					t.Fatalf("verdict = %q", got)
				}
			})
		}
		values := lowRiskLikelihoods()
		delete(values, category)
		if got := PhotoSafeSearchVerdict(values); got != "review" {
			t.Fatalf("missing %s verdict = %q", category, got)
		}
	}
}

func TestPhotoVisionHTTPClient(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/v1/images:annotate" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		var payload struct {
			Requests []struct {
				Image struct {
					Content string `json:"content"`
				} `json:"image"`
				Features []struct {
					Type string `json:"type"`
				} `json:"features"`
			} `json:"requests"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Errorf("decode request: %v", err)
		}
		if len(payload.Requests) != 1 || len(payload.Requests[0].Features) != 1 || payload.Requests[0].Features[0].Type != "SAFE_SEARCH_DETECTION" {
			t.Errorf("unexpected Vision features: %+v", payload.Requests)
		}
		decoded, err := base64.StdEncoding.DecodeString(payload.Requests[0].Image.Content)
		if err != nil || string(decoded) != "test-jpeg" {
			t.Errorf("unexpected image bytes: %q, %v", decoded, err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"responses":[{"safeSearchAnnotation":{"adult":"VERY_UNLIKELY","spoof":"UNLIKELY","medical":"VERY_UNLIKELY","violence":"UNLIKELY","racy":"VERY_UNLIKELY"}}]}`))
	}))
	defer server.Close()
	client := photoVisionHTTPClient{client: server.Client(), endpoint: server.URL + "/v1/images:annotate"}
	result, err := client.analyze(context.Background(), []byte("test-jpeg"))
	if err != nil || result.Status != "safe" {
		t.Fatalf("assessment = %+v, err = %v", result, err)
	}
}

func TestPhotoVisionMissingAssessment(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"responses":[{"error":{"code":429}}]}`))
	}))
	defer server.Close()
	client := photoVisionHTTPClient{client: server.Client(), endpoint: server.URL}
	if _, err := client.analyze(context.Background(), []byte("test-jpeg")); err == nil {
		t.Fatal("Vision error response was accepted")
	}
}

func TestPhotoVisionLocalMock(t *testing.T) {
	t.Setenv("K_SERVICE", "")
	for _, status := range []string{"safe", "review", "error"} {
		t.Run(status, func(t *testing.T) {
			t.Setenv("PHOTO_VISION_MOCK_RESULT", status)
			result, err := AnalyzePhotoSafeSearch(context.Background(), []byte("test-jpeg"))
			if status == "error" {
				if err == nil {
					t.Fatal("mock failure was accepted")
				}
				return
			}
			if err != nil || result.Status != status {
				t.Fatalf("mock assessment = %+v, err = %v", result, err)
			}
		})
	}
}
