package utils

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"
)

func LaunchTranslationWorker(ctx context.Context) error {
	name := strings.TrimSpace(os.Getenv("TRANSLATION_WORKER_JOB_NAME"))
	if name == "" {
		return fmt.Errorf("translation worker is not configured")
	}
	token, err := GoogleAccessToken(ctx)
	if err != nil {
		return err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://run.googleapis.com/v2/"+name+":run", bytes.NewBufferString("{}"))
	if err != nil {
		return err
	}
	request.Header.Set("Authorization", "Bearer "+token)
	request.Header.Set("Content-Type", "application/json")
	response, err := (&http.Client{Timeout: 10 * time.Second}).Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return fmt.Errorf("translation worker launch status %d", response.StatusCode)
	}
	return nil
}
