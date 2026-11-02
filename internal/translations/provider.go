package translations

import (
	"ArmadaCMS/main/utils"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
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
	Config Config
	Client *http.Client
	Token  func(context.Context) (string, error)
}

func NewGemini(config Config) *Gemini {
	return &Gemini{config, &http.Client{Timeout: config.RequestTimeout}, utils.GoogleAccessToken}
}
func generationRequest(job Job, source Source, fields []string, maxOutput int) map[string]any {
	properties := map[string]any{}
	for _, f := range fields {
		properties[f] = map[string]any{"type": "STRING"}
	}
	text := encode(struct {
		Resource, Context string
		Originals         map[string]string
		Formats           map[string]string
		RequestedFields   []string
	}{source.Resource, source.Label, source.Fields, source.Formats, fields})
	return map[string]any{"systemInstruction": map[string]any{"parts": []any{map[string]string{"text": job.Prompt + "\nArmada glossary and reviewed examples:\n" + job.Glossary}}}, "contents": []any{map[string]any{"role": "user", "parts": []any{map[string]string{"text": text}}}}, "generationConfig": map[string]any{"maxOutputTokens": maxOutput, "responseMimeType": "application/json", "responseSchema": map[string]any{"type": "OBJECT", "properties": properties, "required": fields, "propertyOrdering": fields}, "thinkingConfig": map[string]any{"thinkingLevel": "MINIMAL"}}}
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
	access, err := g.Token(ctx)
	if err != nil {
		return Result{}, &ProviderError{"authentication_unavailable", true}
	}
	if g.Config.Project == "" || g.Config.Region == "" || strings.ContainsAny(g.Config.Project+g.Config.Region+job.Model, "/\\?#") {
		return Result{}, &ProviderError{"invalid_provider_configuration", false}
	}
	endpoint := "https://" + g.Config.Region + "-aiplatform.googleapis.com/v1/projects/" + url.PathEscape(g.Config.Project) + "/locations/" + url.PathEscape(g.Config.Region) + "/publishers/google/models/" + url.PathEscape(job.Model) + ":generateContent"
	body, _ := json.Marshal(generationRequest(job, source, fields, maxOutput))
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return Result{}, &ProviderError{"invalid_provider_request", false}
	}
	request.Header.Set("Authorization", "Bearer "+access)
	request.Header.Set("Content-Type", "application/json")
	response, err := g.Client.Do(request)
	if err != nil {
		return Result{}, &ProviderError{"provider_network_error", true}
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return Result{}, &ProviderError{fmt.Sprintf("provider_http_%d", response.StatusCode), response.StatusCode == 429 || response.StatusCode >= 500}
	}
	var output struct {
		Candidates []struct {
			FinishReason string `json:"finishReason"`
			Content      struct {
				Parts []struct {
					Text    string `json:"text"`
					Thought bool   `json:"thought"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
		Usage struct {
			Total int64 `json:"totalTokenCount"`
		} `json:"usageMetadata"`
	}
	if json.NewDecoder(io.LimitReader(response.Body, 2<<20)).Decode(&output) != nil {
		return Result{}, &ProviderError{"invalid_provider_response", false}
	}
	result := Result{Tokens: output.Usage.Total}
	if len(output.Candidates) != 1 || output.Candidates[0].FinishReason != "STOP" {
		return result, &ProviderError{"incomplete_or_blocked_response", false}
	}
	var text strings.Builder
	for _, part := range output.Candidates[0].Content.Parts {
		if !part.Thought {
			text.WriteString(part.Text)
		}
	}
	result.Fields, err = decodeFields(text.String(), fields)
	if err != nil {
		return result, &ProviderError{"invalid_response_fields", false}
	}
	return result, nil
}
