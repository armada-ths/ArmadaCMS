package utils

import (
	"encoding/json"
	"fmt"
	"net/url"
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
	if err := json.Unmarshal([]byte(sortRaw), &params.Sort); err != nil {
		params.Sort = []string{"id", "ASC"} // default
	}

	// Parse range
	rangeRaw := q.Get("range")
	if err := json.Unmarshal([]byte(rangeRaw), &params.Range); err != nil {
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
