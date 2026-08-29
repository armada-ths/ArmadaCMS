package utils

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
)

type ListParams struct {
	Sort   []string          `json:"sort"`
	Range  []int             `json:"range"`
	Filter map[string]string `json:"filter"`
}

func ParseListParams(q url.Values) (ListParams, error) {
	var params ListParams

	// Parse sort
	sortRaw := q.Get("sort")
	if err := json.Unmarshal([]byte(sortRaw), &params.Sort); err != nil || len(params.Sort) != 2 || !isSafeListField(params.Sort[0]) {
		params.Sort = []string{"id", "ASC"} // default
	} else {
		direction := strings.ToUpper(params.Sort[1])
		if direction != "ASC" && direction != "DESC" {
			params.Sort = []string{"id", "ASC"}
		} else {
			params.Sort[1] = direction
		}
	}

	// Parse range
	rangeRaw := q.Get("range")
	if err := json.Unmarshal([]byte(rangeRaw), &params.Range); err != nil ||
		len(params.Range) != 2 || params.Range[0] < 0 || params.Range[1] < params.Range[0] ||
		params.Range[1]-params.Range[0] >= 1000 {
		params.Range = []int{0, 24}
	}

	// Parse filter
	filterRaw := q.Get("filter")
	var filters map[string]any
	if err := json.Unmarshal([]byte(filterRaw), &filters); err != nil {
		params.Filter = map[string]string{}
	} else {
		params.Filter = make(map[string]string, len(filters))
		for key, value := range filters {
			if !isSafeListField(key) {
				continue
			}
			switch typedValue := value.(type) {
			case string:
				params.Filter[key] = typedValue
			case float64:
				params.Filter[key] = fmt.Sprintf("%v", typedValue)
			case bool:
				params.Filter[key] = fmt.Sprintf("%t", typedValue)
			}
		}
	}

	return params, nil
}

// isSafeListField accepts only plain identifier-like API field names. List
// controllers interpolate these names into GORM clauses, so SQL expressions,
// quoting, qualification, whitespace, and punctuation must never pass through.
func isSafeListField(field string) bool {
	if field == "" {
		return false
	}

	for index, char := range field {
		isLetter := char >= 'a' && char <= 'z' || char >= 'A' && char <= 'Z'
		isDigit := char >= '0' && char <= '9'
		if !(isLetter || char == '_' || index > 0 && isDigit) {
			return false
		}
	}

	return true
}
