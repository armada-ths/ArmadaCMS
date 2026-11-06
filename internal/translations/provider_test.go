package translations

import (
	"cloud.google.com/go/auth"
	"cloud.google.com/go/auth/httptransport"
	"context"
	"errors"
	"fmt"
	"google.golang.org/genai"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

type testTokenProvider struct{}

func (testTokenProvider) Token(context.Context) (*auth.Token, error) {
	return &auth.Token{Value: "test-token", Expiry: time.Now().Add(time.Hour)}, nil
}

// Exercise the real SDK and Google authorization transport without ADC discovery
// or network access. The production factory otherwise uses the same config.
func testGemini(t *testing.T, transport http.RoundTripper) *Gemini {
	t.Helper()
	g := NewGemini(Config{Project: "test-project", Region: "eu", RequestTimeout: time.Second})
	g.newClient = func(ctx context.Context, config *genai.ClientConfig) (*genai.Client, error) {
		if config.Backend != genai.BackendVertexAI || config.Location != g.Config.Region || config.Project != "test-project" || config.APIKey != "" || config.HTTPOptions.APIVersion != "v1" || config.HTTPOptions.BaseURL != "" || config.HTTPOptions.Timeout == nil || *config.HTTPOptions.Timeout != time.Second || config.HTTPOptions.RetryOptions == nil || config.HTTPOptions.RetryOptions.Attempts == nil || *config.HTTPOptions.RetryOptions.Attempts != 1 {
			t.Fatal("unexpected SDK configuration")
		}
		var err error
		config.HTTPClient, err = httptransport.NewClient(&httptransport.Options{
			Credentials:      auth.NewCredentials(&auth.CredentialsOptions{TokenProvider: testTokenProvider{}}),
			BaseRoundTripper: transport,
			DisableTelemetry: true,
		})
		if err != nil {
			return nil, err
		}
		return newGeminiClient(ctx, config)
	}
	return g
}

func generateFixture(g *Gemini) (Result, error) {
	return g.Generate(context.Background(), Job{Model: "gemini-3.5-flash-lite", Prompt: Prompt, Glossary: Glossary}, Source{Resource: "blogposts", Fields: map[string]string{"Title": "Original"}}, []string{"Title"}, 100)
}

func TestGeminiSDKDoesNotRetry(t *testing.T) {
	for _, status := range []int{400, 401, 403, 408, 429, 500, 503} {
		t.Run(fmt.Sprint(status), func(t *testing.T) {
			attempts := 0
			g := testGemini(t, roundTripper(func(r *http.Request) (*http.Response, error) {
				attempts++
				response := httptest.NewRecorder()
				response.WriteHeader(status)
				fmt.Fprintf(response, `{"error":{"code":%d,"message":"private provider text","status":"FAILED"}}`, status)
				return response.Result(), nil
			}))
			result, err := generateFixture(g)
			var failure *ProviderError
			if !errors.As(err, &failure) || failure.Category != fmt.Sprintf("provider_http_%d", status) || failure.Retry != (status == 429 || status >= 500) || result.Tokens != 0 || attempts != 1 || strings.Contains(err.Error(), "private") {
				t.Fatalf("attempts=%d result=%+v error=%v", attempts, result, err)
			}
		})
	}
}

func TestGeminiSDKResponseValidationAndUsage(t *testing.T) {
	for _, sample := range []struct {
		name, body, category string
		tokens               int64
	}{
		{"thoughts", `{"candidates":[{"finishReason":"STOP","content":{"parts":[{"thought":true,"text":"private reasoning"},{"text":"{\"Title\":\"Svenska\"}"}]}}],"usageMetadata":{"promptTokenCount":80,"candidatesTokenCount":20,"thoughtsTokenCount":23,"totalTokenCount":123}}`, "", 123},
		{"blocked", `{"candidates":[{"finishReason":"SAFETY"}],"usageMetadata":{"totalTokenCount":12}}`, "incomplete_or_blocked_response", 12},
		{"truncated", `{"candidates":[{"finishReason":"MAX_TOKENS","content":{"parts":[{"text":"{"}]}}],"usageMetadata":{"totalTokenCount":15}}`, "incomplete_or_blocked_response", 15},
		{"missing content", `{"candidates":[{"finishReason":"STOP"}],"usageMetadata":{"totalTokenCount":8}}`, "incomplete_or_blocked_response", 8},
		{"no candidate", `{"usageMetadata":{"totalTokenCount":9}}`, "incomplete_or_blocked_response", 9},
		{"duplicate fields", `{"candidates":[{"finishReason":"STOP","content":{"parts":[{"text":"{\"Title\":\"First\",\"Title\":\"Second\"}"}]}}],"usageMetadata":{"totalTokenCount":10}}`, "invalid_response_fields", 10},
		{"unknown fields", `{"candidates":[{"finishReason":"STOP","content":{"parts":[{"text":"{\"Other\":\"Text\"}"}]}}],"usageMetadata":{"totalTokenCount":11}}`, "invalid_response_fields", 11},
		{"malformed", `private invalid JSON`, "invalid_provider_response", 0},
		{"oversized", `{"candidates":[{"finishReason":"STOP","content":{"parts":[{"text":"` + strings.Repeat("a", 2<<20) + `"}]}}]}`, "invalid_provider_response", 0},
		{"no usage", `{"candidates":[{"finishReason":"STOP","content":{"parts":[{"text":"{\"Title\":\"Svenska\"}"}]}}]}`, "", 0},
	} {
		t.Run(sample.name, func(t *testing.T) {
			g := testGemini(t, roundTripper(func(*http.Request) (*http.Response, error) {
				response := httptest.NewRecorder()
				response.Body.WriteString(sample.body)
				return response.Result(), nil
			}))
			result, err := generateFixture(g)
			if result.Tokens != sample.tokens {
				t.Fatalf("tokens=%d want %d", result.Tokens, sample.tokens)
			}
			if sample.category == "" {
				if err != nil || result.Fields["Title"] != "Svenska" {
					t.Fatalf("result=%+v error=%v", result, err)
				}
			} else {
				var failure *ProviderError
				if !errors.As(err, &failure) || failure.Category != sample.category || failure.Retry || strings.Contains(err.Error(), "private") {
					t.Fatalf("result=%+v error=%v", result, err)
				}
			}
		})
	}
}

func TestGeminiSDKReservationBoundsSerializedRequest(t *testing.T) {
	job := Job{Model: "gemini-3.5-flash-lite", Prompt: strings.Repeat("instruction ", 200), Glossary: Glossary}
	source := Source{Resource: "blogposts", Label: "Context", Fields: map[string]string{"Title": strings.Repeat("\u00e5\u00e4\u00f6 <>& \u4e2d ", 2000)}}
	fields := []string{"Title"}
	g := testGemini(t, roundTripper(func(r *http.Request) (*http.Response, error) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatal(err)
		}
		if reservation(job, source, fields, 200) < int64(len(body))+200 {
			t.Fatal("reservation does not cover SDK request plus output")
		}
		if r.Method != http.MethodPost || r.URL.Path != "/v1/projects/test-project/locations/eu/publishers/google/models/gemini-3.5-flash-lite:generateContent" {
			t.Fatalf("unexpected request %s %s", r.Method, r.URL)
		}
		response := httptest.NewRecorder()
		response.Body.WriteString(`{"candidates":[{"finishReason":"STOP","content":{"parts":[{"text":"{\"Title\":\"Svenska\"}"}]}}]}`)
		return response.Result(), nil
	}))
	if _, err := g.Generate(context.Background(), job, source, fields, 200); err != nil {
		t.Fatal(err)
	}
}

func TestGeminiSDKInitializationAndConfigurationFailures(t *testing.T) {
	g := NewGemini(Config{Project: "test-project", Region: "eu"})
	g.newClient = func(context.Context, *genai.ClientConfig) (*genai.Client, error) {
		return nil, errors.New("private credential details")
	}
	_, err := generateFixture(g)
	var failure *ProviderError
	if !errors.As(err, &failure) || failure.Category != "authentication_unavailable" || !failure.Retry {
		t.Fatal(err)
	}
	g.Config.Region = "eu/invalid"
	_, err = generateFixture(g)
	if !errors.As(err, &failure) || failure.Category != "invalid_provider_configuration" || failure.Retry {
		t.Fatal(err)
	}
}

func TestGeminiRejectsInvalidOutputLimitBeforeModelCall(t *testing.T) {
	g := NewGemini(Config{Project: "test-project", Region: "eu"})
	g.newClient = func(context.Context, *genai.ClientConfig) (*genai.Client, error) {
		t.Fatal("invalid output limit must not initialize the SDK or request credentials")
		return nil, nil
	}
	for _, limit := range []int{-1, 0, 65537, 2147483647} {
		result, err := g.Generate(context.Background(), Job{Model: "gemini-3.5-flash-lite"}, Source{}, nil, limit)
		var failure *ProviderError
		if !errors.As(err, &failure) || failure.Category != "invalid_provider_configuration" || failure.Retry || result.Tokens != 0 {
			t.Fatalf("limit=%d result=%+v error=%v", limit, result, err)
		}
	}
}

func TestGeminiSDKUsesDocumentedEndpointHosts(t *testing.T) {
	for _, sample := range []struct{ region, host string }{
		{"eu", "aiplatform.eu.rep.googleapis.com"},
		{"us", "aiplatform.us.rep.googleapis.com"},
		{"global", "aiplatform.googleapis.com"},
		{"europe-north1", "europe-north1-aiplatform.googleapis.com"},
	} {
		t.Run(sample.region, func(t *testing.T) {
			g := testGemini(t, roundTripper(func(r *http.Request) (*http.Response, error) {
				if r.URL.Host != sample.host || !strings.Contains(r.URL.Path, "/locations/"+sample.region+"/") {
					t.Fatalf("incorrect model endpoint: %s", r.URL)
				}
				response := httptest.NewRecorder()
				response.Body.WriteString(`{"candidates":[{"finishReason":"STOP","content":{"parts":[{"text":"{\"Title\":\"Svenska\"}"}]}}]}`)
				return response.Result(), nil
			}))
			g.Config.Region = sample.region
			if _, err := generateFixture(g); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestGeminiInvalidHostnameHasSafeActionableCategory(t *testing.T) {
	g := testGemini(t, roundTripper(func(*http.Request) (*http.Response, error) {
		response := httptest.NewRecorder()
		response.WriteHeader(http.StatusBadRequest)
		response.Body.WriteString(`{"error":{"code":400,"message":"Invalid hostname: private-host.example","status":"INVALID_ARGUMENT"}}`)
		return response.Result(), nil
	}))
	_, err := generateFixture(g)
	var failure *ProviderError
	if !errors.As(err, &failure) || failure.Category != "provider_invalid_endpoint" || failure.Retry || strings.Contains(err.Error(), "private-host") {
		t.Fatal(err)
	}
}
