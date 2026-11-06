package translations

import "testing"

func TestTranslationConfigurationLimits(t *testing.T) {
	t.Setenv("TRANSLATION_AI_ENABLED", "")
	t.Setenv("TRANSLATION_MODEL", "")
	t.Setenv("TRANSLATION_REGION", "")
	for _, sample := range []struct {
		value           string
		records, output int
	}{
		{"2147483647", 50, 65536},
		{"999999999999999999999999999999999999", 50, 8192},
		{"-1", 50, 8192},
		{"12", 12, 12},
	} {
		t.Setenv("TRANSLATION_MAX_RECORDS", sample.value)
		t.Setenv("TRANSLATION_MAX_OUTPUT_TOKENS", sample.value)
		config := ConfigFromEnv()
		if config.Enabled || config.Model != "gemini-3.5-flash-lite" || config.Region != "eu" || config.MaxRecords != sample.records || config.MaxOutput != sample.output {
			t.Fatalf("configuration for %s: %+v", sample.value, config)
		}
	}
}
