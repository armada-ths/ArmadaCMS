package controllers

import (
	"ArmadaCMS/main/models"
	"fmt"
	"mime/multipart"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestEveryPublicTranslationHasSparseUpdateSemantics(t *testing.T) {
	resources := map[string]func() any{
		"highlight":  func() any { return &models.HighlightCard{} },
		"blog":       func() any { return &models.Blogpost{} },
		"event":      func() any { return &models.Event{} },
		"exhibitor":  func() any { return &models.Exhibitor{} },
		"employment": func() any { return &models.Employment{} },
		"industry":   func() any { return &models.Industry{} },
		"program":    func() any { return &models.Program{} },
		"team":       func() any { return &models.Team{} },
		"profile":    func() any { return &models.Profile{} },
		"period":     func() any { return &models.RecruitmentPeriod{} },
		"role":       func() any { return &models.RecruitmentRole{} },
		"fair":       func() any { return &models.FairDateConfig{} },
		"era":        func() any { return &models.TimelineEra{} },
		"entry":      func() any { return &models.TimelineEntry{} },
	}
	for resource, factory := range resources {
		fields := translationValues(factory())
		if len(fields) == 0 {
			t.Fatalf("%s has no translation fields", resource)
		}
		for key := range fields {
			for _, body := range []string{"{}", fmt.Sprintf("{%q:null}", key), fmt.Sprintf("{%q:%q}", key, ""), fmt.Sprintf("{%q:%q}", key, "  "), fmt.Sprintf("{%q:%q}", key, " Svenska ")} {
				t.Run(resource+"/"+key+"/"+body, func(t *testing.T) {
					target := factory()
					old := "Previous"
					*translationValues(target)[key] = &old
					raw, err := decodeWithFields(strings.NewReader(body), target)
					if err != nil {
						t.Fatal(err)
					}
					updates := map[string]any{}
					mergeJSONTranslations(raw, target, updates)
					dest := translationValues(target)[key]
					if body == "{}" {
						if len(updates) != 0 || *dest == nil || **dest != old {
							t.Fatal("omitted translation was changed")
						}
					} else if strings.Contains(body, "Svenska") {
						if len(updates) != 1 || *dest == nil || **dest != "Svenska" {
							t.Fatal("translation not normalized", updates)
						}
					} else if len(updates) != 1 || *dest != nil {
						t.Fatal("translation not cleared", updates)
					}
				})
			}
			for _, value := range []string{"", "  ", " Svenska "} {
				t.Run(resource+"/"+key+"/multipart/"+value, func(t *testing.T) {
					target := factory()
					old := "Previous"
					*translationValues(target)[key] = &old
					req := httptest.NewRequest("PUT", "/", nil)
					req.MultipartForm = &multipart.Form{Value: map[string][]string{key: {value}}}
					req.Form = req.MultipartForm.Value
					updates := map[string]any{}
					mergeMultipartTranslations(req, target, updates)
					dest := translationValues(target)[key]
					if len(updates) != 1 {
						t.Fatal("incorrect update map", updates)
					}
					if strings.TrimSpace(value) == "" {
						if *dest != nil {
							t.Fatal("multipart did not clear translation")
						}
					} else if *dest == nil || **dest != "Svenska" {
						t.Fatal("multipart did not normalize translation")
					}
				})
			}
		}
	}
}
