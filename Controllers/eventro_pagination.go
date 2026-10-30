package controllers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

// Fetch the entire validated export before mutating any CMS records.
func fetchEventroPages[T any](ctx context.Context, client *http.Client, endpoint, key, token, organization string) ([]T, error) {
	all := []T{}
	seen := map[string]bool{}
	expected := -1
	for page := 0; page < 10000; page++ {
		u, err := url.Parse(endpoint)
		if err != nil {
			return nil, err
		}
		query := u.Query()
		query.Set("pageIndex", strconv.Itoa(page))
		query.Set("pageSize", "100")
		u.RawQuery = query.Encode()
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("organization", organization)
		resp, err := client.Do(req)
		if err != nil {
			return nil, err
		}
		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			return nil, fmt.Errorf("unexpected Eventro status %d", resp.StatusCode)
		}
		var envelope map[string]json.RawMessage
		err = json.NewDecoder(resp.Body).Decode(&envelope)
		resp.Body.Close()
		if err != nil {
			return nil, err
		}
		var count int
		if len(envelope["count"]) == 0 || string(envelope["count"]) == "null" {
			return nil, fmt.Errorf("missing Eventro count")
		}
		if err := json.Unmarshal(envelope["count"], &count); err != nil {
			var text string
			if json.Unmarshal(envelope["count"], &text) != nil {
				return nil, fmt.Errorf("invalid Eventro count")
			}
			count, err = strconv.Atoi(text)
			if err != nil {
				return nil, fmt.Errorf("invalid Eventro count")
			}
		}
		if expected < 0 {
			expected = count
		}
		if count < 0 || count != expected {
			return nil, fmt.Errorf("inconsistent Eventro count")
		}
		var rawItems []json.RawMessage
		if json.Unmarshal(envelope[key], &rawItems) != nil || string(envelope[key]) == "null" {
			return nil, fmt.Errorf("invalid Eventro %s list", key)
		}
		if len(all)+len(rawItems) > expected || (len(rawItems) == 0 && len(all) < expected) {
			return nil, fmt.Errorf("incomplete Eventro pagination")
		}
		for _, raw := range rawItems {
			var identifier struct {
				ID string `json:"id"`
			}
			if json.Unmarshal(raw, &identifier) != nil || strings.TrimSpace(identifier.ID) == "" || seen[identifier.ID] {
				return nil, fmt.Errorf("invalid or duplicate Eventro identity")
			}
			seen[identifier.ID] = true
			var item T
			if err := json.Unmarshal(raw, &item); err != nil {
				return nil, err
			}
			all = append(all, item)
		}
		if len(all) == expected {
			return all, nil
		}
	}
	return nil, fmt.Errorf("eventro pagination limit exceeded")
}
