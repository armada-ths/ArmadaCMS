package utils

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// StartCloudRunJob requests an execution using the runtime service account.
// Queue eligibility and worker-specific configuration remain with the caller.
func StartCloudRunJob(ctx context.Context, jobName string) error {
	return startCloudRunJob(ctx, jobName, GoogleAccessToken, &http.Client{Timeout: 10 * time.Second})
}

func startCloudRunJob(ctx context.Context, jobName string, accessToken func(context.Context) (string, error), client *http.Client) error {
	jobName = strings.TrimSpace(jobName)
	if jobName == "" {
		return fmt.Errorf("cloud run job name missing")
	}
	token, err := accessToken(ctx)
	if err != nil {
		return err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://run.googleapis.com/v2/"+jobName+":run", bytes.NewBufferString("{}"))
	if err != nil {
		return err
	}
	request.Header.Set("Authorization", "Bearer "+token)
	request.Header.Set("Content-Type", "application/json")
	response, err := client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return fmt.Errorf("cloud run job launch status %d", response.StatusCode)
	}
	return nil
}
