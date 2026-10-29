package controllers

import (
	"ArmadaCMS/main/audit"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"

	"ArmadaCMS/main/db"
	"ArmadaCMS/main/models"
	"ArmadaCMS/main/utils"

	"gorm.io/gorm"
)

type eventroRecruitmentsResponse struct {
	Recruitments []eventroRecruitmentResponse `json:"recruitments"`
	Count        int                          `json:"count"`
}

type eventroRecruitmentResponse struct {
	ID                     string                         `json:"id"`
	Name                   string                         `json:"name"`
	OpenedAt               string                         `json:"openedAt"`
	ClosedAt               string                         `json:"closedAt"`
	RecruitmentPeriodRoles []eventroRecruitmentRoleDetail `json:"recruitmentPeriodRoles"`
}

type eventroRecruitmentRoleDetail struct {
	ID              string `json:"id"`
	Role            string `json:"role"`
	RoleDescription string `json:"roleDescription"`
	Group           string `json:"group"`
	Category        string `json:"category"`
	Team            string `json:"team"`
}

type recruitmentRoleResponse struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type recruitmentResponse struct {
	Name      string                               `json:"name"`
	Link      string                               `json:"link"`
	StartDate string                               `json:"start_date"`
	EndDate   string                               `json:"end_date"`
	Groups    map[string][]recruitmentRoleResponse `json:"groups"`
}

// FetchRecruitmentsEventro syncs recruitment data from the Eventro API.
// @Summary Sync recruitment data from Eventro
// @Tags eventro
// @Produce json
// @Success 200 {string} string "Sync complete"
// @Failure 500 {string} string "Failed to fetch from Eventro"
// @Security BearerAuth
// @Router /eventrorecruitments [get]
func FetchRecruitmentsEventro(w http.ResponseWriter, r *http.Request) {
	group, err := startAuditGroup(r, "sync", "recruitment", "eventro-recruitments", map[string]any{"source": "eventro", "status": "running"})
	if err != nil {
		http.Error(w, "failed to start audit log", http.StatusInternalServerError)
		return
	}
	inserted, updated, failed, err := syncRecruitmentsFromEventro(audit.WithParent(r, group.ID))
	if err != nil {
		_ = finalizeAuditGroup(group.ID, "failed", map[string]any{"source": "eventro", "error": err.Error()})
		log.Printf("❌ Recruitment sync failed: %v", err)
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	status := auditGroupStatus(inserted+updated, failed)
	if err := finalizeAuditGroup(group.ID, status, map[string]any{"source": "eventro", "inserted": inserted, "updated": updated, "failed": failed}); err != nil {
		http.Error(w, "failed to finalize audit log", http.StatusInternalServerError)
		return
	}
	if inserted+updated > 0 {
		go utils.RevalidateTag("recruitment")
		go utils.RevalidateTag("organization")
	}

	message := fmt.Sprintf("Sync completed — inserted: %d, updated: %d, failed: %d", inserted, updated, failed)
	log.Printf("✅ Recruitment sync completed — inserted: %d, updated: %d", inserted, updated)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(message))
}

// GetRecruitment returns all active recruitment periods with their roles,
// filtered to only include open periods.
// @Summary Get active recruitment
// @Tags public
// @Produce json
// @Success 200 {array} models.RecruitmentPeriod
// @Router /recruitment [get]
func GetRecruitment(w http.ResponseWriter, r *http.Request) {
	var periods []models.RecruitmentPeriod
	if err := db.DB.
		Preload("Roles", func(tx *gorm.DB) *gorm.DB { return tx.Order("id ASC") }).
		Preload("Roles.Team").
		Find(&periods).Error; err != nil {
		http.Error(w, "failed to fetch recruitment periods", http.StatusInternalServerError)
		return
	}

	selected := selectRecruitmentPeriod(periods, time.Now().UTC())
	w.Header().Set("Content-Type", "application/json")

	if selected == nil {
		_, _ = w.Write([]byte("null"))
		return
	}

	response := mapRecruitmentPeriodToResponse(*selected)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "failed to encode recruitment response", http.StatusInternalServerError)
	}
}

func syncRecruitmentsFromEventro(auditRequest *http.Request) (int, int, int, error) {
	client := &http.Client{Timeout: 30 * time.Second}
	baseURL := "https://app.eventro.se/api/v1/recruitments/"

	allRecruitments, err := fetchEventroPages[eventroRecruitmentResponse](auditRequest.Context(), client, baseURL, "recruitments", os.Getenv("EVENTRO_API"), os.Getenv("EVENTRO_ORG"))
	if err != nil {
		return 0, 0, 0, err
	}

	inserted := 0
	updated := 0
	failed := 0

	for _, incoming := range allRecruitments {
		var periodID uint
		var wasInserted, wasUpdated bool
		err := db.DB.Transaction(func(tx *gorm.DB) error {
			var err error
			periodID, wasInserted, wasUpdated, err = upsertRecruitmentPeriod(tx, auditRequest, incoming)
			return err
		})
		if err != nil {
			log.Printf("❌ Failed upserting recruitment period %q (%s): %v", incoming.Name, incoming.ID, err)
			failed++
			continue
		}

		if wasInserted {
			inserted++
		}
		if wasUpdated {
			updated++
		}

		for _, role := range incoming.RecruitmentPeriodRoles {
			var roleInserted, roleUpdated bool
			roleErr := db.DB.Transaction(func(tx *gorm.DB) error {
				var err error
				roleInserted, roleUpdated, err = upsertRecruitmentRole(tx, auditRequest, periodID, role)
				return err
			})
			if roleErr != nil {
				log.Printf("❌ Failed upserting recruitment role %q (%s): %v", role.Role, role.ID, roleErr)
				failed++
				continue
			}

			if roleInserted {
				inserted++
			}
			if roleUpdated {
				updated++
			}
		}
	}

	return inserted, updated, failed, nil
}

func upsertRecruitmentPeriod(tx *gorm.DB, r *http.Request, incoming eventroRecruitmentResponse) (uint, bool, bool, error) {
	id := strings.TrimSpace(incoming.ID)
	if id == "" || !validEventroTimestamp(incoming.OpenedAt) || !validEventroTimestamp(incoming.ClosedAt) {
		return 0, false, false, fmt.Errorf("invalid recruitment identity or dates")
	}
	period := models.RecruitmentPeriod{EventroID: &id, Name: strings.TrimSpace(incoming.Name), Link: buildRecruitmentLink(id), StartDate: parseEventroTime(incoming.OpenedAt), EndDate: parseEventroTime(incoming.ClosedAt)}
	return syncEventroRecordWithAudit(tx, r, "recruitmentperiods", &period, "eventro_id", id, nil, nil)
}

func upsertRecruitmentRole(tx *gorm.DB, r *http.Request, periodID uint, incoming eventroRecruitmentRoleDetail) (bool, bool, error) {
	id := strings.TrimSpace(incoming.ID)
	if id == "" {
		return false, false, fmt.Errorf("missing Eventro role id")
	}
	teamID, err := findOrCreateTeamIDByName(tx, r, deriveRecruitmentRoleTeamName(incoming))
	if err != nil {
		return false, false, err
	}
	role := models.RecruitmentRole{EventroRoleID: &id, RecruitmentID: periodID, TeamID: teamID, Name: strings.TrimSpace(incoming.Role), Description: strings.TrimSpace(incoming.RoleDescription)}
	fields := []string{"name", "description", "recruitmentId"}
	if teamID != nil {
		fields = append(fields, "team_id")
	}
	_, inserted, updated, err := syncEventroRecordWithAudit(tx, r, "recruitmentroles", &role, "eventro_role_id", id, map[string]any{"recruitment_id": periodID, "LOWER(name)": strings.ToLower(role.Name)}, fields)
	return inserted, updated, err
}

func parseEventroTime(input string) *time.Time {
	value := strings.TrimSpace(input)
	if value == "" {
		return nil
	}

	t, err := time.Parse(time.RFC3339, value)
	if err != nil {
		log.Printf("⚠️ Failed to parse Eventro time %q: %v", value, err)
		return nil
	}

	return &t
}

func trimStringPointer(v *string) *string {
	if v == nil {
		return nil
	}

	trimmed := strings.TrimSpace(*v)
	if trimmed == "" {
		return nil
	}

	return &trimmed
}

func deriveRecruitmentRoleTeamName(role eventroRecruitmentRoleDetail) string {
	for _, candidate := range []string{role.Group, role.Category, role.Team} {
		value := strings.TrimSpace(candidate)
		if value != "" {
			return value
		}
	}

	return ""
}

func findOrCreateTeamIDByName(tx *gorm.DB, auditRequest *http.Request, teamName string) (*uint, error) {
	trimmed := strings.TrimSpace(teamName)
	if trimmed == "" {
		return nil, nil
	}

	var team models.Team
	err := tx.Where("LOWER(team_name) = ?", strings.ToLower(trimmed)).First(&team).Error
	if err != nil {
		if err != gorm.ErrRecordNotFound {
			return nil, err
		}

		team = models.Team{TeamName: trimmed}
		if createErr := tx.Create(&team).Error; createErr != nil {
			return nil, createErr
		}
		if auditErr := audit.LogCreate(tx, auditRequest, "teams", team.ID, team); auditErr != nil {
			return nil, auditErr
		}
	}

	return &team.ID, nil
}

func buildRecruitmentLink(eventroRecruitmentID string) string {
	if eventroRecruitmentID == "" {
		return ""
	}

	return fmt.Sprintf("https://app.eventro.se/recruitments/%s", eventroRecruitmentID)
}

func selectRecruitmentPeriod(periods []models.RecruitmentPeriod, now time.Time) *models.RecruitmentPeriod {
	if len(periods) == 0 {
		return nil
	}

	open := make([]models.RecruitmentPeriod, 0)
	upcoming := make([]models.RecruitmentPeriod, 0)
	ended := make([]models.RecruitmentPeriod, 0)

	for _, period := range periods {
		if period.StartDate != nil && period.EndDate != nil {
			if !now.Before(*period.StartDate) && now.Before(*period.EndDate) {
				open = append(open, period)
				continue
			}

			if now.Before(*period.StartDate) {
				upcoming = append(upcoming, period)
				continue
			}

			ended = append(ended, period)
			continue
		}

		if period.StartDate != nil && now.Before(*period.StartDate) {
			upcoming = append(upcoming, period)
			continue
		}

		if period.EndDate != nil && !now.Before(*period.EndDate) {
			ended = append(ended, period)
			continue
		}

		open = append(open, period)
	}

	if len(open) > 0 {
		sort.Slice(open, func(i, j int) bool {
			if open[i].StartDate == nil {
				return false
			}
			if open[j].StartDate == nil {
				return true
			}
			return open[i].StartDate.Before(*open[j].StartDate)
		})
		return &open[0]
	}

	if len(upcoming) > 0 {
		sort.Slice(upcoming, func(i, j int) bool {
			if upcoming[i].StartDate == nil {
				return false
			}
			if upcoming[j].StartDate == nil {
				return true
			}
			return upcoming[i].StartDate.Before(*upcoming[j].StartDate)
		})
		return &upcoming[0]
	}

	sort.Slice(ended, func(i, j int) bool {
		if ended[i].EndDate == nil {
			return false
		}
		if ended[j].EndDate == nil {
			return true
		}
		return ended[i].EndDate.After(*ended[j].EndDate)
	})

	if len(ended) > 0 {
		return &ended[0]
	}

	return &periods[0]
}

func mapRecruitmentPeriodToResponse(period models.RecruitmentPeriod) recruitmentResponse {
	groups := map[string][]recruitmentRoleResponse{}

	for _, role := range period.Roles {
		teamName := ""
		if role.Team != nil {
			teamName = strings.TrimSpace(role.Team.TeamName)
		}

		groups[teamName] = append(groups[teamName], recruitmentRoleResponse{
			Name:        role.Name,
			Description: role.Description,
		})
	}

	startDate := ""
	if period.StartDate != nil {
		startDate = period.StartDate.UTC().Format(time.RFC3339)
	}

	endDate := ""
	if period.EndDate != nil {
		endDate = period.EndDate.UTC().Format(time.RFC3339)
	}

	return recruitmentResponse{
		Name:      period.Name,
		Link:      period.Link,
		StartDate: startDate,
		EndDate:   endDate,
		Groups:    groups,
	}
}
