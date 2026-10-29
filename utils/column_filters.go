package utils

import (
	"fmt"
	"math"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/schema"
)

// ApplyColumnFilters matches exact values in public scalar model fields. Both
// JSON field names and database column names are accepted. Identifiers come
// from GORM's schema, never from the request. Hidden, derived and relation
// fields cannot be used as filter predicates.
func ApplyColumnFilters(query *gorm.DB, filters map[string]string) (*gorm.DB, error) {
	statement := &gorm.Statement{DB: query}
	if err := statement.Parse(query.Statement.Model); err != nil {
		return nil, fmt.Errorf("invalid list model: %w", err)
	}
	fields := make(map[string]*schema.Field)
	for _, field := range statement.Schema.Fields {
		jsonName := strings.Split(field.StructField.Tag.Get("json"), ",")[0]
		if field.DBName == "" || jsonName == "" || jsonName == "-" || !field.Readable || field.Serializer != nil {
			continue
		}
		kind := field.IndirectFieldType.Kind()
		scalar := kind == reflect.String || kind == reflect.Bool ||
			kind >= reflect.Int && kind <= reflect.Float64 || field.DataType == schema.Time
		if !scalar {
			continue
		}
		fields[strings.ToLower(jsonName)] = field
		fields[strings.ToLower(field.DBName)] = field
	}
	// Deterministic predicates simplify diagnostics and make equivalent requests
	// produce identical query shapes.
	keys := make([]string, 0, len(filters))
	for key := range filters {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		value := filters[key]
		if key == "q" || value == "" {
			continue
		}
		field := fields[strings.ToLower(key)]
		if field == nil {
			return nil, fmt.Errorf("unsupported filter %q", key)
		}
		parsed, err := columnFilterValue(field, value)
		if err != nil {
			return nil, fmt.Errorf("invalid value for filter %q", key)
		}
		query = query.Where(clause.Eq{
			Column: clause.Column{Table: statement.Table, Name: field.DBName},
			Value:  parsed,
		})
	}
	return query, nil
}

func columnFilterValue(field *schema.Field, value string) (any, error) {
	if field.DataType == schema.Time {
		return time.Parse(time.RFC3339Nano, value)
	}
	switch field.IndirectFieldType.Kind() {
	case reflect.String:
		return value, nil
	case reflect.Bool:
		return strconv.ParseBool(value)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return strconv.ParseInt(value, 10, field.IndirectFieldType.Bits())
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		// PostgreSQL integer columns are signed, including Go uint IDs.
		return strconv.ParseUint(value, 10, min(field.IndirectFieldType.Bits(), 64)-1)
	case reflect.Float32, reflect.Float64:
		number, err := strconv.ParseFloat(value, field.IndirectFieldType.Bits())
		if err != nil || math.IsNaN(number) || math.IsInf(number, 0) {
			return nil, fmt.Errorf("invalid number")
		}
		return number, nil
	default:
		return nil, fmt.Errorf("unsupported field type")
	}
}
