// Package eventrocategories normalizes Eventro catalogue labels using the configured CSV mapping.
package eventrocategories

import (
	_ "embed"
	"encoding/csv"
	"fmt"
	"sort"
	"strings"
)

// CategoryType identifies one Eventro catalogue category namespace.
type CategoryType string

const (
	Industry     CategoryType = "Industry"
	Employment   CategoryType = "Employment type"
	FieldOfStudy CategoryType = "Field of study"
)

//go:embed mapping.csv
var mappingCSV string

var mappings = mustLoadMappings(mappingCSV)

func lookupKey(value string) string {
	return strings.ToLower(strings.Join(strings.Fields(value), " "))
}

func mustLoadMappings(data string) map[CategoryType]map[string]string {
	result, err := loadMappings(data)
	if err != nil {
		panic(err)
	}
	return result
}

func loadMappings(data string) (map[CategoryType]map[string]string, error) {
	rows, err := csv.NewReader(strings.NewReader(strings.TrimPrefix(data, "\ufeff"))).ReadAll()
	if err != nil {
		return nil, fmt.Errorf("read Eventro category mapping: %w", err)
	}
	if len(rows) < 2 || strings.Join(rows[0], ",") != "Type,Original category,Normalized category,Review" {
		return nil, fmt.Errorf("invalid Eventro category mapping header or empty mapping")
	}
	result := map[CategoryType]map[string]string{}
	for index, row := range rows[1:] {
		kind := CategoryType(strings.TrimSpace(row[0]))
		if kind != Industry && kind != Employment && kind != FieldOfStudy {
			return nil, fmt.Errorf("unknown category type %q at CSV row %d", kind, index+2)
		}
		original, normalized := lookupKey(row[1]), strings.TrimSpace(row[2])
		if original == "" || normalized == "" {
			return nil, fmt.Errorf("empty category at CSV row %d", index+2)
		}
		if result[kind] == nil {
			result[kind] = map[string]string{}
		}
		for _, key := range []string{original, lookupKey(normalized)} {
			if previous, exists := result[kind][key]; exists && previous != normalized {
				return nil, fmt.Errorf("conflicting category %q at CSV row %d", key, index+2)
			}
			result[kind][key] = normalized
		}
	}
	return result, nil
}

// Normalize maps known aliases, retains unknown labels, and returns a sorted unique list.
// Types are separate: for example, Energy is an industry but maps to Energy Engineering as a field of study.
func Normalize(kind CategoryType, labels []string) []string {
	result := []string{}
	seen := map[string]bool{}
	for _, label := range labels {
		label = strings.Join(strings.Fields(label), " ")
		if normalized, exists := mappings[kind][lookupKey(label)]; exists {
			label = normalized
		}
		key := lookupKey(label)
		if key != "" && !seen[key] {
			seen[key] = true
			result = append(result, label)
		}
	}
	sort.Strings(result)
	return result
}
