package utils

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"golang.org/x/oauth2/google"
)

const photoVisionEndpoint = "https://vision.googleapis.com/v1/images:annotate"

var photoVisionCategories = [...]string{"adult", "spoof", "medical", "violence", "racy"}

type PhotoVisionAssessment struct {
	Status      string
	Likelihoods map[string]string
}

type photoVisionHTTPClient struct {
	client   *http.Client
	endpoint string
}

// PhotoSafeSearchVerdict accepts only unambiguous low-risk results.
func PhotoSafeSearchVerdict(likelihoods map[string]string) string {
	for _, category := range photoVisionCategories {
		value := likelihoods[category]
		if value != "VERY_UNLIKELY" && value != "UNLIKELY" {
			return "review"
		}
	}
	return "safe"
}

// AnalyzePhotoSafeSearch sends only the temporary analysis copy to Google.
// The local mock is explicit and ignored by Cloud Run.
func AnalyzePhotoSafeSearch(ctx context.Context, jpeg []byte) (PhotoVisionAssessment, error) {
	if os.Getenv("K_SERVICE") == "" {
		switch strings.TrimSpace(os.Getenv("PHOTO_VISION_MOCK_RESULT")) {
		case "safe":
			return PhotoVisionAssessment{Status: "safe", Likelihoods: map[string]string{
				"adult": "VERY_UNLIKELY", "spoof": "VERY_UNLIKELY", "medical": "VERY_UNLIKELY", "violence": "VERY_UNLIKELY", "racy": "VERY_UNLIKELY",
			}}, nil
		case "review":
			return PhotoVisionAssessment{Status: "review", Likelihoods: map[string]string{
				"adult": "POSSIBLE", "spoof": "VERY_UNLIKELY", "medical": "VERY_UNLIKELY", "violence": "VERY_UNLIKELY", "racy": "VERY_UNLIKELY",
			}}, nil
		case "error":
			return PhotoVisionAssessment{}, errors.New("simulated Vision failure")
		}
	}
	httpClient, err := google.DefaultClient(ctx, "https://www.googleapis.com/auth/cloud-platform")
	if err != nil {
		return PhotoVisionAssessment{}, fmt.Errorf("find Vision credentials: %w", err)
	}
	httpClient.Timeout = 6 * time.Second
	return (&photoVisionHTTPClient{client: httpClient, endpoint: photoVisionEndpoint}).analyze(ctx, jpeg)
}

func (client *photoVisionHTTPClient) analyze(ctx context.Context, jpeg []byte) (PhotoVisionAssessment, error) {
	if len(jpeg) == 0 || len(jpeg) > 1024*1024 {
		return PhotoVisionAssessment{}, errors.New("invalid analysis image size")
	}
	requestBody, err := json.Marshal(map[string]any{"requests": []any{map[string]any{
		"image":    map[string]string{"content": base64.StdEncoding.EncodeToString(jpeg)},
		"features": []any{map[string]string{"type": "SAFE_SEARCH_DETECTION"}},
	}}})
	if err != nil {
		return PhotoVisionAssessment{}, err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, client.endpoint, bytes.NewReader(requestBody))
	if err != nil {
		return PhotoVisionAssessment{}, err
	}
	request.Header.Set("Content-Type", "application/json")
	response, err := client.client.Do(request)
	if err != nil {
		return PhotoVisionAssessment{}, err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return PhotoVisionAssessment{}, fmt.Errorf("vision returned HTTP %d", response.StatusCode)
	}
	var result struct {
		Responses []struct {
			SafeSearch map[string]string `json:"safeSearchAnnotation"`
			Error      *struct {
				Code int `json:"code"`
			} `json:"error"`
		} `json:"responses"`
	}
	if err := json.NewDecoder(io.LimitReader(response.Body, 64*1024)).Decode(&result); err != nil {
		return PhotoVisionAssessment{}, err
	}
	if len(result.Responses) != 1 || result.Responses[0].Error != nil || result.Responses[0].SafeSearch == nil {
		return PhotoVisionAssessment{}, errors.New("vision returned no usable assessment")
	}
	likelihoods := make(map[string]string, len(photoVisionCategories))
	for _, category := range photoVisionCategories {
		value := result.Responses[0].SafeSearch[category]
		if value == "" {
			value = "UNKNOWN"
		}
		likelihoods[category] = value
	}
	return PhotoVisionAssessment{Status: PhotoSafeSearchVerdict(likelihoods), Likelihoods: likelihoods}, nil
}

func PhotoVisionTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(ctx, 6*time.Second)
}
