package utils

import (
	"context"
	"fmt"
	"os"
	"strings"
)

func LaunchTranslationWorker(ctx context.Context) error {
	name := strings.TrimSpace(os.Getenv("TRANSLATION_WORKER_JOB_NAME"))
	if name == "" {
		return fmt.Errorf("translation worker is not configured")
	}
	return StartCloudRunJob(ctx, name)
}
