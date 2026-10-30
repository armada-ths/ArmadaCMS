package controllers

import (
	"ArmadaCMS/main/audit"
	"ArmadaCMS/main/db"
	"ArmadaCMS/main/models"
	"ArmadaCMS/main/utils"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
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

// FetchMembersEventro syncs member profiles from the Eventro API into the local database and returns the current list.
// @Summary Sync & list member profiles from Eventro
// @Tags eventro
// @Produce json
// @Success 200 {array} models.Profile
// @Failure 500 {string} string "Failed to fetch from Eventro"
// @Security BearerAuth
// @Router /eventromembers [get]
func FetchMembersEventro(w http.ResponseWriter, r *http.Request) {
	endpoint := "https://app.eventro.se/api/v1/members/export"
	req, err := http.NewRequestWithContext(r.Context(), http.MethodGet, endpoint, nil)
	if err != nil {
		http.Error(w, "failed to create request", http.StatusInternalServerError)
		return
	}
	req.Header.Set("Authorization", "Bearer "+os.Getenv("EVENTRO_API"))
	req.Header.Set("organization", os.Getenv("EVENTRO_ORG"))

	resp, err := (&http.Client{Timeout: 30 * time.Second}).Do(req)
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
	failed := 0
	group, err := startAuditGroup(r, "sync", "profiles", "eventro-members", map[string]any{"source": "eventro", "status": "running"})
	if err != nil {
		http.Error(w, "failed to start audit log", http.StatusInternalServerError)
		return
	}
	childRequest := audit.WithParent(r, group.ID)

	seenMembers := map[string]bool{}
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
			if eventroKey == "" || seenMembers[eventroKey] {
				continue
			}

			seenMembers[eventroKey] = true
			role := strings.TrimSpace(m.Role)

			email := ""
			if shouldGeneratePGEmail(rank, role) && hasSingleLastName(surname) {
				email = buildArmadaEmail(firstName, surname)
			}

			image := strings.TrimSpace(m.Image)

			profile := models.Profile{EventroKey: &eventroKey, Name: fullName, Rank: rank, Title: role, Email: email, Photo: image}
			var created, changed bool
			err := db.DB.Transaction(func(tx *gorm.DB) error {
				_, inserted, updated, err := syncEventroRecordWithAudit(tx, childRequest, "profiles", &profile, "eventro_key", eventroKey, map[string]any{"LOWER(name)": strings.ToLower(fullName)}, nil)
				created, changed = inserted, updated
				return err
			})
			if err != nil {
				failed++
				continue
			}
			if created {
				inserted++
			}
			if changed {
				updated++
			}
		}
	}

	status := auditGroupStatus(inserted+updated, failed)
	if err := finalizeAuditGroup(group.ID, status, map[string]any{"source": "eventro", "inserted": inserted, "updated": updated, "failed": failed}); err != nil {
		http.Error(w, "failed to finalize audit log", http.StatusInternalServerError)
		return
	}
	if inserted+updated > 0 {
		go utils.RevalidateTag("organization")
	}
	message := fmt.Sprintf(
		"Sync completed — inserted: %d, updated: %d, failed: %d",
		inserted,
		updated,
		failed,
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
