package controllers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"
)

const eventroFairsURL = "https://app.eventro.se/api/v1/fairs/"

type eventroFairsResponse struct {
	FairInstances []eventroFairResponse `json:"fairInstances"`
}

type eventroFairResponse struct {
	ID              string                 `json:"id"`
	Name            string                 `json:"name"`
	StartDate       string                 `json:"startDate"`
	EndDate         string                 `json:"endDate"`
	TimelineEntries []eventroTimelineEntry `json:"timelineEntries"`
}

type eventroTimelineEntry struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	Date  string `json:"date"`
}

// GetEventroFairs returns the active fair instances available to the configured Eventro organization.
// @Summary List active Eventro fair instances
// @Tags eventro
// @Produce json
// @Success 200 {object} eventroFairsResponse
// @Failure 502 {string} string "Failed to fetch from Eventro"
// @Security BearerAuth
// @Router /eventrofairs [get]
func GetEventroFairs(w http.ResponseWriter, r *http.Request) {
	result, err := fetchEventroFairs(
		r.Context(),
		&http.Client{Timeout: 30 * time.Second},
		eventroFairsURL,
		os.Getenv("EVENTRO_API"),
		os.Getenv("EVENTRO_ORG"),
	)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}

	writeJSONResponse(w, http.StatusOK, result)
}

func fetchEventroFairs(ctx context.Context, client *http.Client, url, apiKey, organizationID string) (eventroFairsResponse, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return eventroFairsResponse{}, fmt.Errorf("failed to create Eventro request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("organization", organizationID)

	resp, err := client.Do(req)
	if err != nil {
		return eventroFairsResponse{}, fmt.Errorf("failed to contact Eventro API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return eventroFairsResponse{}, fmt.Errorf("unexpected Eventro status: %s", resp.Status)
	}

	var result eventroFairsResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return eventroFairsResponse{}, fmt.Errorf("failed to decode Eventro response: %w", err)
	}
	return result, nil
}

func fetchEventroFairByID(ctx context.Context, client *http.Client, url, apiKey, organizationID, fairID string) (eventroFairResponse, error) {
	result, err := fetchEventroFairs(ctx, client, url, apiKey, organizationID)
	if err != nil {
		return eventroFairResponse{}, err
	}

	for _, fair := range result.FairInstances {
		if fair.ID == fairID {
			return fair, nil
		}
	}
	return eventroFairResponse{}, fmt.Errorf("selected Eventro fair %q was not returned by the active fairs endpoint", fairID)
}

func requireEventroFairID(w http.ResponseWriter, r *http.Request) (string, bool) {
	fairID := strings.TrimSpace(r.URL.Query().Get("fairId"))
	if fairID == "" {
		http.Error(w, "fairId query parameter is required", http.StatusBadRequest)
		return "", false
	}
	return fairID, true
}
