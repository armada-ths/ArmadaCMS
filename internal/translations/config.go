package translations

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	Enabled                         bool
	Model, Project, Region, JobName string
	DailyBudget                     int64
	MaxRecords, MaxOutput           int
	RequestTimeout                  time.Duration
}

func positive(name string, fallback int64) int64 {
	v, err := strconv.ParseInt(os.Getenv(name), 10, 64)
	if err != nil || v <= 0 {
		return fallback
	}
	return v
}
func ConfigFromEnv() Config {
	model := os.Getenv("TRANSLATION_MODEL")
	if model == "" {
		model = "gemini-3.5-flash-lite"
	}
	region := os.Getenv("TRANSLATION_REGION")
	if region == "" {
		region = "eu"
	}
	project := os.Getenv("GOOGLE_CLOUD_PROJECT")
	return Config{os.Getenv("TRANSLATION_AI_ENABLED") == "true", model, project, region, os.Getenv("TRANSLATION_WORKER_JOB_NAME"), positive("TRANSLATION_DAILY_TOKEN_BUDGET", 1000000), int(min(positive("TRANSLATION_MAX_RECORDS", 50), 50)), int(min(positive("TRANSLATION_MAX_OUTPUT_TOKENS", 8192), 65536)), 120 * time.Second}
}
