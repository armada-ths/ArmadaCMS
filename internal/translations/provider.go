package translations

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"google.golang.org/genai"
	"io"
	"net/http"
	"net/url"
	"strings"
)

type Result struct {
	Fields map[string]string
	Tokens int64
}
type Generator interface {
	Generate(context.Context, Job, Source, []string, int) (Result, error)
}
type ProviderError struct {
	Category string
	Retry    bool
}

func (e *ProviderError) Error() string { return e.Category }
func retryable(err error) bool         { var e *ProviderError; return errors.As(err, &e) && e.Retry }

type Gemini struct {
	Config    Config
	newClient func(context.Context, *genai.ClientConfig) (*genai.Client, error)
}

func NewGemini(config Config) *Gemini {
	return &Gemini{Config: config, newClient: newGeminiClient}
}

// Keep the existing response-size bound while the SDK owns response decoding.
type providerResponseTransport struct{ base http.RoundTripper }
type providerResponseBody struct {
	io.Reader
	io.Closer
}

func (t providerResponseTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	response, err := t.base.RoundTrip(r)
	if response != nil && response.Body != nil {
		response.Body = providerResponseBody{io.LimitReader(response.Body, 2<<20), response.Body}
	}
	return response, err
}
func newGeminiClient(ctx context.Context, config *genai.ClientConfig) (*genai.Client, error) {
	client, err := genai.NewClient(ctx, config)
	if err != nil {
		return nil, err
	}
	httpClient := client.ClientConfig().HTTPClient
	base := httpClient.Transport
	if base == nil {
		base = http.DefaultTransport
	}
	httpClient.Transport = providerResponseTransport{base: base}
	return client, nil
}

type generationPayload struct {
	Contents []*genai.Content             `json:"contents"`
	Config   *genai.GenerateContentConfig `json:"config"`
}

func generationRequest(job Job, source Source, fields []string, maxOutput int) generationPayload {
	properties := map[string]*genai.Schema{}
	for _, f := range fields {
		properties[f] = &genai.Schema{Type: genai.TypeString}
	}
	text := encode(struct {
		Resource, Context string
		Originals         map[string]string
		Formats           map[string]string
		RequestedFields   []string
	}{source.Resource, source.Label, source.Fields, source.Formats, fields})
	return generationPayload{
		Contents: genai.Text(text),
		Config: &genai.GenerateContentConfig{
			SystemInstruction: &genai.Content{Parts: []*genai.Part{genai.NewPartFromText(job.Prompt + "\nArmada glossary and reviewed examples:\n" + job.Glossary)}},
			MaxOutputTokens:   int32(maxOutput),
			ResponseMIMEType:  "application/json",
			ResponseSchema:    &genai.Schema{Type: genai.TypeObject, Properties: properties, Required: fields, PropertyOrdering: fields},
			ThinkingConfig:    &genai.ThinkingConfig{ThinkingLevel: genai.ThinkingLevelMinimal},
		},
	}
}

// Text-only tokenization cannot use more tokens than UTF-8 bytes. Reserve that
// conservative input bound plus protocol headroom and the full output limit;
// settle to provider-reported total usage (including thoughts) after each call.
func reservation(job Job, source Source, fields []string, maxOutput int) int64 {
	return int64(len(encode(generationRequest(job, source, fields, maxOutput)))) + 4096 + int64(maxOutput)
}
func decodeFields(text string, expected []string) (map[string]string, error) {
	d := json.NewDecoder(strings.NewReader(text))
	start, err := d.Token()
	if err != nil || start != json.Delim('{') {
		return nil, fmt.Errorf("invalid structured response")
	}
	out := map[string]string{}
	for d.More() {
		key, err := d.Token()
		if err != nil {
			return nil, fmt.Errorf("invalid field identifier")
		}
		name, ok := key.(string)
		if !ok {
			return nil, fmt.Errorf("invalid field identifier")
		}
		if _, exists := out[name]; exists {
			return nil, fmt.Errorf("duplicate field identifier")
		}
		var value *string
		if d.Decode(&value) != nil || value == nil {
			return nil, fmt.Errorf("invalid field value")
		}
		out[name] = *value
	}
	if token, err := d.Token(); err != nil || token != json.Delim('}') {
		return nil, fmt.Errorf("invalid structured response")
	}
	if _, err = d.Token(); err != io.EOF {
		return nil, fmt.Errorf("trailing response content")
	}
	if len(out) != len(expected) {
		return nil, fmt.Errorf("wrong response fields")
	}
	for _, key := range expected {
		if _, ok := out[key]; !ok {
			return nil, fmt.Errorf("missing response field")
		}
	}
	return out, nil
}
func (g *Gemini) Generate(ctx context.Context, job Job, source Source, fields []string, maxOutput int) (Result, error) {
	if g.Config.Project == "" || g.Config.Region == "" || strings.ContainsAny(g.Config.Project+g.Config.Region+job.Model, "/\\?#") {
		return Result{}, &ProviderError{"invalid_provider_configuration", false}
	}
	// Explicit backend, endpoint and version preserve the deployed Vertex API.
	// ADC uses the Cloud Run runtime identity. No API key is configured.
	client, err := g.newClient(ctx, &genai.ClientConfig{
		Backend:  genai.BackendVertexAI,
		Project:  g.Config.Project,
		Location: g.Config.Region,
		HTTPOptions: genai.HTTPOptions{
			BaseURL:    "https://" + g.Config.Region + "-aiplatform.googleapis.com/",
			APIVersion: "v1",
			Timeout:    &g.Config.RequestTimeout,
			// Each worker attempt has its own budget reservation. Never retry inside it.
			RetryOptions: &genai.HTTPRetryOptions{Attempts: genai.Ptr(int32(1))},
		},
	})
	if err != nil {
		return Result{}, &ProviderError{"authentication_unavailable", true}
	}
	request := generationRequest(job, source, fields, maxOutput)
	output, err := client.Models.GenerateContent(ctx, job.Model, request.Contents, request.Config)
	if err != nil {
		var apiErr genai.APIError
		if errors.As(err, &apiErr) {
			return Result{}, &ProviderError{fmt.Sprintf("provider_http_%d", apiErr.Code), apiErr.Code == 429 || apiErr.Code >= 500}
		}
		var networkErr *url.Error
		if errors.As(err, &networkErr) || errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
			return Result{}, &ProviderError{"provider_network_error", true}
		}
		// SDK errors can include response text. Expose only a fixed category.
		return Result{}, &ProviderError{"invalid_provider_response", false}
	}
	if output == nil {
		return Result{}, &ProviderError{"invalid_provider_response", false}
	}
	result := Result{}
	if output.UsageMetadata != nil {
		result.Tokens = int64(output.UsageMetadata.TotalTokenCount)
	}
	if len(output.Candidates) != 1 || output.Candidates[0] == nil || output.Candidates[0].FinishReason != genai.FinishReasonStop || output.Candidates[0].Content == nil {
		return result, &ProviderError{"incomplete_or_blocked_response", false}
	}
	var text strings.Builder
	for _, part := range output.Candidates[0].Content.Parts {
		if part != nil && !part.Thought {
			text.WriteString(part.Text)
		}
	}
	result.Fields, err = decodeFields(text.String(), fields)
	if err != nil {
		return result, &ProviderError{"invalid_response_fields", false}
	}
	return result, nil
}
