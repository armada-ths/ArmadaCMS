package utils

import (
	"encoding/json"
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
	if err := json.Unmarshal([]byte(filterRaw), &params.Filter); err != nil {
		params.Filter = map[string]string{}
	}

	return params, nil
}
