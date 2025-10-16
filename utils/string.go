package utils

import "strings"

func StringPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func ToSnakeCase(s string) string {
	var result []rune
	for i, r := range s {
		if i > 0 && r >= 'A' && r <= 'Z' {
			result = append(result, '_', r+('a'-'A'))
		} else {
			result = append(result, r)
		}
	}
	return strings.ToLower(string(result))
}
