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
func boundedPositive(name string, fallback, ceiling int) int {
	value, err := strconv.Atoi(os.Getenv(name))
	if err != nil || value <= 0 {
		return fallback
	}
	if value > ceiling {
		return ceiling
	}
	return value
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
	return Config{
		Enabled:        os.Getenv("TRANSLATION_AI_ENABLED") == "true",
		Model:          model,
		Project:        os.Getenv("GOOGLE_CLOUD_PROJECT"),
		Region:         region,
		JobName:        os.Getenv("TRANSLATION_WORKER_JOB_NAME"),
		DailyBudget:    positive("TRANSLATION_DAILY_TOKEN_BUDGET", 1000000),
		MaxRecords:     boundedPositive("TRANSLATION_MAX_RECORDS", 50, 50),
		MaxOutput:      boundedPositive("TRANSLATION_MAX_OUTPUT_TOKENS", 8192, 65536),
		RequestTimeout: 120 * time.Second,
	}
}
