package controllers

import (
	"ArmadaCMS/main/db"
	"ArmadaCMS/main/models"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"unicode"

	"gorm.io/gorm"
)

type eventroMembersResponse struct {
	Groups []eventroSecurityGroup `json:"groups"`
}

type eventroSecurityGroup struct {
	Group   string          `json:"group"`
	Members []eventroMember `json:"members"`
}

type eventroMember struct {
	FirstName string `json:"firstName"`
	Surname   string `json:"surname"`
	Role      string `json:"role"`
	Image     string `json:"image"`
}

func FetchMembersEventro(w http.ResponseWriter, r *http.Request) {
	endpoint := "https://app.eventro.se/api/v1/members/export"
	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		http.Error(w, "failed to create request", http.StatusInternalServerError)
		return
	}
	req.Header.Set("Authorization", "Bearer "+os.Getenv("EVENTRO_API"))
	req.Header.Set("organization", os.Getenv("EVENTRO_ORG"))

	resp, err := (&http.Client{}).Do(req)
	if err != nil {
		http.Error(w, "failed to contact Eventro API", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		http.Error(w, "unexpected Eventro status: "+resp.Status, http.StatusBadGateway)
		return
	}

	var result eventroMembersResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		log.Printf("❌ Failed to decode Eventro members response: %v", err)
		http.Error(w, "failed to decode Eventro response", http.StatusInternalServerError)
		return
	}

	inserted := 0
	updated := 0

	for _, group := range result.Groups {
		eventroGroup := strings.TrimSpace(group.Group)
		rank := resolveAllowedRank(eventroGroup)

		for _, m := range group.Members {
			firstName := strings.TrimSpace(m.FirstName)
			surname := strings.TrimSpace(m.Surname)
			fullName := strings.TrimSpace(strings.TrimSpace(firstName + " " + surname))
			if fullName == "" {
				continue
			}

			eventroKey := buildEventroMemberKey(firstName, surname)
			if eventroKey == "" {
				continue
			}

			role := strings.TrimSpace(m.Role)

			email := ""
			if shouldGeneratePGEmail(rank, role) && hasSingleLastName(surname) {
				email = buildArmadaEmail(firstName, surname)
			}

			image := strings.TrimSpace(m.Image)

			var existing models.Profile
			err := db.DB.Where("eventro_key = ?", eventroKey).First(&existing).Error
			if err != nil && err != gorm.ErrRecordNotFound {
				log.Printf("❌ Failed reading profile for key %s: %v", eventroKey, err)
				continue
			}

			if err == gorm.ErrRecordNotFound {
				// fallback dedupe: if a profile with the same name already exists,
				// reuse it instead of creating a duplicate
				errByName := db.DB.
					Where("LOWER(name) = ?", strings.ToLower(fullName)).
					First(&existing).Error

				if errByName != nil && errByName != gorm.ErrRecordNotFound {
					log.Printf("❌ Failed reading profile for name %s: %v", fullName, errByName)
					continue
				}

				if errByName == nil {
					updates := map[string]interface{}{}
					if existing.EventroKey == nil {
						updates["eventro_key"] = eventroKey
					}
					if strings.TrimSpace(existing.Rank) == "" && rank != "" {
						updates["rank"] = rank
					}
					if strings.TrimSpace(existing.Title) == "" && role != "" {
						updates["title"] = role
					}
					if strings.TrimSpace(existing.Email) == "" && email != "" {
						updates["email"] = email
					}
					if strings.TrimSpace(existing.Photo) == "" && image != "" {
						updates["photo"] = image
					}

					if len(updates) > 0 {
						if updateErr := db.DB.Model(&existing).Updates(updates).Error; updateErr != nil {
							log.Printf("❌ Failed updating existing profile by name %s: %v", fullName, updateErr)
							continue
						}
						updated++
					}

					continue
				}

				profile := models.Profile{
					EventroKey: &eventroKey,
					Name:       fullName,
					Rank:       rank,
					Title:      role,
					Linkedin:   "",
					Email:      email,
					Photo:      image,
				}

				if createErr := db.DB.Create(&profile).Error; createErr != nil {
					log.Printf("❌ Failed creating member %s: %v", fullName, createErr)
					continue
				}

				inserted++
				continue
			}

			updates := map[string]interface{}{}
			if strings.TrimSpace(existing.Name) == "" {
				updates["name"] = fullName
			}
			if strings.TrimSpace(existing.Rank) == "" && rank != "" {
				updates["rank"] = rank
			}
			if strings.TrimSpace(existing.Title) == "" && role != "" {
				updates["title"] = role
			}
			if strings.TrimSpace(existing.Email) == "" && email != "" {
				updates["email"] = email
			}
			if strings.TrimSpace(existing.Photo) == "" && image != "" {
				updates["photo"] = image
			}

			if len(updates) == 0 {
				continue
			}

			if updateErr := db.DB.Model(&existing).Updates(updates).Error; updateErr != nil {
				log.Printf("❌ Failed updating member %s: %v", fullName, updateErr)
				continue
			}

			updated++
		}
	}

	message := fmt.Sprintf(
		"Sync completed — inserted: %d, updated: %d",
		inserted,
		updated,
	)
	log.Printf("✅ %s", message)
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(message))
}

func resolveAllowedRank(eventroGroup string) string {
	normalized := strings.ToLower(strings.TrimSpace(eventroGroup))

	allowedRanks := map[string]string{
		"project manager": "Project Manager",
		"project group":   "Project Group",
		"operation team":  "Operation Team",
		"host":            "Host",
	}

	if rank, ok := allowedRanks[normalized]; ok {
		return rank
	}

	return ""
}

func shouldGeneratePGEmail(rank, role string) bool {
	normalized := strings.ToLower(strings.TrimSpace(rank + " " + role))

	if strings.Contains(normalized, "operation team") || strings.Contains(normalized, " ot") {
		return false
	}

	return strings.Contains(normalized, "project group") ||
		strings.Contains(normalized, " project manager") ||
		strings.HasPrefix(normalized, "pg") ||
		strings.Contains(normalized, " pg")
}

func hasSingleLastName(surname string) bool {
	parts := strings.Fields(strings.TrimSpace(surname))
	return len(parts) <= 1
}

func buildArmadaEmail(firstName, surname string) string {
	first := sanitizeNameForEmail(firstName)
	last := sanitizeNameForEmail(surname)

	switch {
	case first != "" && last != "":
		return fmt.Sprintf("%s.%s@armada.nu", first, last)
	case first != "":
		return fmt.Sprintf("%s@armada.nu", first)
	case last != "":
		return fmt.Sprintf("%s@armada.nu", last)
	default:
		return ""
	}
}

func buildEventroMemberKey(firstName, surname string) string {
	first := sanitizeNameForEmail(firstName)
	last := sanitizeNameForEmail(surname)

	if first == "" && last == "" {
		return ""
	}

	if last == "" {
		return first
	}
	if first == "" {
		return last
	}

	return first + "." + last
}

func sanitizeNameForEmail(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))

	replacer := strings.NewReplacer(
		"å", "a",
		"ä", "a",
		"ö", "o",
		"é", "e",
		"è", "e",
		"ê", "e",
		"ü", "u",
	)
	value = replacer.Replace(value)

	var out strings.Builder
	lastWasDot := false

	for _, r := range value {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			out.WriteRune(r)
			lastWasDot = false
		case r == ' ' || r == '-' || r == '_' || r == '.':
			if !lastWasDot {
				out.WriteRune('.')
				lastWasDot = true
			}
		}
	}

	sanitized := strings.Trim(out.String(), ".")
	return sanitized
}
