package controllers

import (
	"ArmadaCMS/main/utils"
	"encoding/json"
	"io"
	"net/http"
	"reflect"
	"strings"
)

// translationValues only exposes declared nullable Swedish string fields.
func translationValues(target any) map[string]**string {
	values := map[string]**string{}
	value := reflect.ValueOf(target).Elem()
	typ := value.Type()
	for i := 0; i < value.NumField(); i++ {
		field := typ.Field(i)
		if field.Tag.Get("gorm") == "-" {
			continue
		}
		if !strings.HasSuffix(field.Name, "Sv") || field.Type != reflect.TypeOf((*string)(nil)) {
			continue
		}
		key := strings.Split(field.Tag.Get("json"), ",")[0]
		if key == "" || key == "-" {
			continue
		}
		values[key] = value.Field(i).Addr().Interface().(**string)
	}
	return values
}

func readMultipartTranslations(r *http.Request, target any) {
	if r.MultipartForm == nil {
		for _, dest := range translationValues(target) {
			NormalizeOptionalStringPointers(dest)
		}
		return
	}
	for key, dest := range translationValues(target) {
		if _, present := r.MultipartForm.Value[key]; !present {
			continue
		}
		value := r.FormValue(key)
		*dest = &value
		NormalizeOptionalStringPointers(dest)
	}
}

func mergeMultipartTranslations(r *http.Request, target any, updates map[string]any) {
	readMultipartTranslations(r, target)
	for key, dest := range translationValues(target) {
		column := utils.ToSnakeCase(key)
		delete(updates, column)
		if _, present := r.MultipartForm.Value[key]; present {
			updates[column] = *dest
		}
	}
}

func decodeWithFields(body io.Reader, target any) (map[string]any, error) {
	var raw map[string]any
	if err := json.NewDecoder(body).Decode(&raw); err != nil {
		return nil, err
	}
	encoded, err := json.Marshal(raw)
	if err != nil {
		return nil, err
	}
	return raw, json.Unmarshal(encoded, target)
}

func mergeJSONTranslations(raw map[string]any, target any, updates map[string]any) {
	for key, dest := range translationValues(target) {
		column := utils.ToSnakeCase(key)
		delete(updates, column)
		if _, present := raw[key]; present {
			NormalizeOptionalStringPointers(dest)
			updates[column] = *dest
		}
	}
}
