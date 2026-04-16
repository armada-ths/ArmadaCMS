package controllers

import (
	"ArmadaCMS/main/utils"
	"strings"
)

func NormalizeOptionalStringPointers(values ...**string) {
	for _, value := range values {
		if value == nil || *value == nil {
			continue
		}

		trimmed := strings.TrimSpace(**value)
		if trimmed == "" {
			*value = nil
			continue
		}

		**value = trimmed
	}
}

func BuildNormalizedSnakeCaseUpdateMap(raw map[string]any, optionalStringFields map[string]struct{}, ignoredFields map[string]struct{}) map[string]any {
	updates := make(map[string]any)

	for key, value := range raw {
		if _, ignored := ignoredFields[key]; ignored {
			continue
		}

		normalizedValue := value
		if _, normalize := optionalStringFields[key]; normalize {
			normalizedValue = normalizeOptionalStringValue(value)
		}

		updates[utils.ToSnakeCase(key)] = normalizedValue
	}

	return updates
}

func normalizeOptionalStringValue(value any) any {
	str, ok := value.(string)
	if !ok {
		return value
	}

	trimmed := strings.TrimSpace(str)
	if trimmed == "" {
		return nil
	}

	return trimmed
}
