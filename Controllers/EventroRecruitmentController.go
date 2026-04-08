package controllers

import (
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
	inserted, updated, err := syncRecruitmentsFromEventro()
	if err != nil {
		log.Printf("❌ Recruitment sync failed: %v", err)
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}

	message := fmt.Sprintf("Sync completed — inserted: %d, updated: %d", inserted, updated)
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

func syncRecruitmentsFromEventro() (int, int, error) {
	client := &http.Client{Timeout: 30 * time.Second}
	baseURL := "https://app.eventro.se/api/v1/recruitments/"

	allRecruitments := []eventroRecruitmentResponse{}
	page := 0

	for {
		url := fmt.Sprintf("%s?pageIndex=%d", baseURL, page)
		req, err := http.NewRequest(http.MethodGet, url, nil)
		if err != nil {
			return 0, 0, fmt.Errorf("failed to create Eventro request: %w", err)
		}

		req.Header.Set("Authorization", "Bearer "+os.Getenv("EVENTRO_API"))
		req.Header.Set("organization", os.Getenv("EVENTRO_ORG"))

		resp, err := client.Do(req)
		if err != nil {
			return 0, 0, fmt.Errorf("failed to contact Eventro API: %w", err)
		}

		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			return 0, 0, fmt.Errorf("unexpected Eventro status: %s", resp.Status)
		}

		var result eventroRecruitmentsResponse
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			resp.Body.Close()
			return 0, 0, fmt.Errorf("failed to decode Eventro recruitments response: %w", err)
		}

		resp.Body.Close()

		if len(result.Recruitments) == 0 {
			break
		}

		allRecruitments = append(allRecruitments, result.Recruitments...)
		if len(allRecruitments) >= result.Count {
			break
		}

		page++
	}

	inserted := 0
	updated := 0

	for _, incoming := range allRecruitments {
		periodID, wasInserted, wasUpdated, err := upsertRecruitmentPeriod(incoming)
		if err != nil {
			log.Printf("❌ Failed upserting recruitment period %q (%s): %v", incoming.Name, incoming.ID, err)
			continue
		}

		if wasInserted {
			inserted++
		}
		if wasUpdated {
			updated++
		}

		for _, role := range incoming.RecruitmentPeriodRoles {
			roleInserted, roleUpdated, roleErr := upsertRecruitmentRole(periodID, role)
			if roleErr != nil {
				log.Printf("❌ Failed upserting recruitment role %q (%s): %v", role.Role, role.ID, roleErr)
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

	return inserted, updated, nil
}

func upsertRecruitmentPeriod(incoming eventroRecruitmentResponse) (uint, bool, bool, error) {
	eventroID := strings.TrimSpace(incoming.ID)
	if eventroID == "" {
		return 0, false, false, fmt.Errorf("missing Eventro recruitment id")
	}

	startAt := parseEventroTime(incoming.OpenedAt)
	endAt := parseEventroTime(incoming.ClosedAt)
	name := strings.TrimSpace(incoming.Name)
	link := buildRecruitmentLink(eventroID)

	var existing models.RecruitmentPeriod
	err := db.DB.Where("eventro_id = ?", eventroID).First(&existing).Error
	if err != nil {
		if err != gorm.ErrRecordNotFound {
			return 0, false, false, err
		}

		period := models.RecruitmentPeriod{
			EventroID: &eventroID,
			Name:      name,
			Link:      link,
			StartDate: startAt,
			EndDate:   endAt,
		}

		if createErr := db.DB.Create(&period).Error; createErr != nil {
			return 0, false, false, createErr
		}

		return period.ID, true, false, nil
	}

	updates := map[string]interface{}{}
	if existing.EventroID == nil {
		updates["eventro_id"] = eventroID
	}
	if strings.TrimSpace(existing.Name) == "" && name != "" {
		updates["name"] = name
	}
	if strings.TrimSpace(existing.Link) == "" && link != "" {
		updates["link"] = link
	}
	if existing.StartDate == nil && startAt != nil {
		updates["start_date"] = *startAt
	}
	if existing.EndDate == nil && endAt != nil {
		updates["end_date"] = *endAt
	}

	if len(updates) > 0 {
		if updateErr := db.DB.Model(&existing).Updates(updates).Error; updateErr != nil {
			return 0, false, false, updateErr
		}
		return existing.ID, false, true, nil
	}

	return existing.ID, false, false, nil
}

func upsertRecruitmentRole(periodID uint, incoming eventroRecruitmentRoleDetail) (bool, bool, error) {
	eventroRoleID := strings.TrimSpace(incoming.ID)
	if eventroRoleID == "" {
		return false, false, fmt.Errorf("missing Eventro recruitment role id")
	}

	name := strings.TrimSpace(incoming.Role)
	description := strings.TrimSpace(incoming.RoleDescription)
	teamName := deriveRecruitmentRoleTeamName(incoming)
	teamID, teamErr := findOrCreateTeamIDByName(teamName)
	if teamErr != nil {
		return false, false, fmt.Errorf("failed resolving team for role %q: %w", name, teamErr)
	}

	var existing models.RecruitmentRole
	err := db.DB.Where("eventro_role_id = ?", eventroRoleID).First(&existing).Error
	if err != nil {
		if err != gorm.ErrRecordNotFound {
			return false, false, err
		}

		// fallback dedupe: if same role name already exists for this recruitment period,
		// reuse it instead of creating a new row
		errByName := db.DB.
			Where("recruitment_id = ?", periodID).
			Where("LOWER(name) = ?", strings.ToLower(name)).
			First(&existing).Error

		if errByName != nil && errByName != gorm.ErrRecordNotFound {
			return false, false, errByName
		}

		if errByName == nil {
			updates := map[string]interface{}{}
			if existing.EventroRoleID == nil {
				updates["eventro_role_id"] = eventroRoleID
			}
			if existing.RecruitmentID == 0 && periodID > 0 {
				updates["recruitment_id"] = periodID
			}
			if strings.TrimSpace(existing.Description) == "" && description != "" {
				updates["description"] = description
			}
			if existing.TeamID == nil && teamID != nil {
				updates["team_id"] = *teamID
			}

			if len(updates) > 0 {
				if updateErr := db.DB.Model(&existing).Updates(updates).Error; updateErr != nil {
					return false, false, updateErr
				}
				return false, true, nil
			}

			return false, false, nil
		}

		role := models.RecruitmentRole{
			EventroRoleID: &eventroRoleID,
			RecruitmentID: periodID,
			TeamID:        teamID,
			Name:          name,
			Description:   description,
		}

		if createErr := db.DB.Create(&role).Error; createErr != nil {
			return false, false, createErr
		}

		return true, false, nil
	}

	updates := map[string]interface{}{}
	if existing.EventroRoleID == nil {
		updates["eventro_role_id"] = eventroRoleID
	}
	if existing.RecruitmentID == 0 && periodID > 0 {
		updates["recruitment_id"] = periodID
	}
	if strings.TrimSpace(existing.Name) == "" && name != "" {
		updates["name"] = name
	}
	if strings.TrimSpace(existing.Description) == "" && description != "" {
		updates["description"] = description
	}
	if existing.TeamID == nil && teamID != nil {
		updates["team_id"] = *teamID
	}

	if len(updates) > 0 {
		if updateErr := db.DB.Model(&existing).Updates(updates).Error; updateErr != nil {
			return false, false, updateErr
		}
		return false, true, nil
	}

	return false, false, nil
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

func findOrCreateTeamIDByName(teamName string) (*uint, error) {
	trimmed := strings.TrimSpace(teamName)
	if trimmed == "" {
		return nil, nil
	}

	var team models.Team
	err := db.DB.Where("LOWER(team_name) = ?", strings.ToLower(trimmed)).First(&team).Error
	if err != nil {
		if err != gorm.ErrRecordNotFound {
			return nil, err
		}

		team = models.Team{TeamName: trimmed}
		if createErr := db.DB.Create(&team).Error; createErr != nil {
			return nil, createErr
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
