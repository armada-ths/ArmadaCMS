package utils

import (
	"net/url"
	"reflect"
	"testing"
)

func TestParseListParamsParsesValidValues(t *testing.T) {
	q := url.Values{}
	q.Set("sort", `["name","DESC"]`)
	q.Set("range", `[10,19]`)
	q.Set("filter", `{"role":"admin"}`)

	params, err := ParseListParams(q)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !reflect.DeepEqual(params.Sort, []string{"name", "DESC"}) {
		t.Errorf("unexpected Sort: %#v", params.Sort)
	}
	if !reflect.DeepEqual(params.Range, []int{10, 19}) {
		t.Errorf("unexpected Range: %#v", params.Range)
	}
	if !reflect.DeepEqual(params.Filter, map[string]string{"role": "admin"}) {
		t.Errorf("unexpected Filter: %#v", params.Filter)
	}
}

func TestParseListParamsAppliesDefaultsForMissingValues(t *testing.T) {
	params, err := ParseListParams(url.Values{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !reflect.DeepEqual(params.Sort, []string{"id", "ASC"}) {
		t.Errorf("expected default Sort [id, ASC], got %#v", params.Sort)
	}
	if !reflect.DeepEqual(params.Range, []int{0, 24}) {
		t.Errorf("expected default Range [0, 24], got %#v", params.Range)
	}
	if params.Filter == nil || len(params.Filter) != 0 {
		t.Errorf("expected default Filter to be empty map, got %#v", params.Filter)
	}
}

func TestParseListParamsAppliesDefaultsForInvalidJSON(t *testing.T) {
	q := url.Values{}
	q.Set("sort", "not-json")
	q.Set("range", "not-json")
	q.Set("filter", "not-json")

	params, err := ParseListParams(q)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !reflect.DeepEqual(params.Sort, []string{"id", "ASC"}) {
		t.Errorf("expected fallback Sort [id, ASC], got %#v", params.Sort)
	}
	if !reflect.DeepEqual(params.Range, []int{0, 24}) {
		t.Errorf("expected fallback Range [0, 24], got %#v", params.Range)
	}
	if params.Filter == nil || len(params.Filter) != 0 {
		t.Errorf("expected fallback Filter to be empty map, got %#v", params.Filter)
	}
}
