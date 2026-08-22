package controllers

import (
	"ArmadaCMS/main/audit"
	"ArmadaCMS/main/db"
	"ArmadaCMS/main/models"
	"ArmadaCMS/main/utils"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
	"gorm.io/gorm"
)

// GetTeams returns a paginated list of teams.
// @Summary List teams
// @Tags teams
// @Produce json
// @Param range query string false "Pagination range, e.g. [0,24]"
// @Param sort query string false "Sort, e.g. [\"team_name\",\"ASC\"]"
// @Param filter query string false "Filter, e.g. {\"team_name\":\"core\"}"
// @Success 200 {array} models.Team
// @Header 200 {string} Content-Range "teams 0-24/10"
// @Router /teams [get]
func GetTeams(w http.ResponseWriter, r *http.Request) {
	params, _ := utils.ParseListParams(r.URL.Query())

	var teams []models.Team
	query := db.DB.Model(&models.Team{})

	for k, v := range params.Filter {
		query = query.Where(k+" = ?", v)
	}

	if len(params.Sort) == 2 {
		query = query.Order(params.Sort[0] + " " + params.Sort[1])
	}

	start, end := params.Range[0], params.Range[1]
	limit := end - start + 1

	var total int64
	db.DB.Model(&models.Team{}).Count(&total)

	query = query.Offset(start).Limit(limit)
	query.Find(&teams)

	w.Header().Set("Access-Control-Expose-Headers", "Content-Range")
	w.Header().Set("Content-Range", fmt.Sprintf("teams %d-%d/%d", start, end, total))
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(teams)
}

// GetTeamByID returns a single team by ID.
// @Summary Get team by ID
// @Tags teams
// @Produce json
// @Param id path int true "Team ID"
// @Success 200 {object} models.Team
// @Failure 404 {string} string "team not found"
// @Router /teams/{id} [get]
func GetTeamByID(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	var team models.Team
	if err := db.DB.First(&team, id).Error; err != nil {
		http.Error(w, "team not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(team)
}

// CreateTeam creates a new team.
// @Summary Create team
// @Tags teams
// @Accept json
// @Produce json
// @Param body body models.Team true "Team data"
// @Success 201 {object} models.Team
// @Failure 400 {string} string "Invalid body"
// @Failure 500 {string} string "Create failed"
// @Security BearerAuth
// @Router /teams [post]
func CreateTeam(w http.ResponseWriter, r *http.Request) {
	var team models.Team
	if err := json.NewDecoder(r.Body).Decode(&team); err != nil {
		http.Error(w, "Invalid body", http.StatusBadRequest)
		return
	}
	if err := createWithAudit(r, "teams", &team, func(tx *gorm.DB) error {
		return tx.Create(&team).Error
	}, nil, "organization"); err != nil {
		http.Error(w, "Create failed", http.StatusInternalServerError)
		return
	}

	writeCreatedJSONResponse(w, team)
}

// UpdateTeam updates a team by ID.
// @Summary Update team
// @Tags teams
// @Accept json
// @Produce json
// @Param id path int true "Team ID"
// @Param body body models.Team true "Updated team data"
// @Success 200 {object} models.Team
// @Failure 400 {string} string "Invalid data"
// @Failure 404 {string} string "Not found"
// @Failure 500 {string} string "Update failed"
// @Security BearerAuth
// @Router /teams/{id} [put]
func UpdateTeam(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	var team models.Team
	if err := db.DB.First(&team, id).Error; err != nil {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}

	var updates models.Team
	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		http.Error(w, "Invalid data", http.StatusBadRequest)
		return
	}

	before := team
	if err := updateWithAudit(r, "teams", id, before, &team, func(tx *gorm.DB) error {
		return tx.Model(&team).Updates(updates).Error
	}, func(tx *gorm.DB) error {
		return tx.First(&team, id).Error
	}, "organization"); err != nil {
		http.Error(w, "Update failed", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(team)
}

// DeleteTeam deletes a team by ID.
// @Summary Delete team
// @Tags teams
// @Produce json
// @Param id path int true "Team ID"
// @Success 200 {string} string "Deleted"
// @Failure 404 {string} string "team not found"
// @Security BearerAuth
// @Router /teams/{id} [delete]
func DeleteTeam(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	writeGroupedDeleteResponseWithAudit(w, r, "teams", id, "team not found", nil, func(tx *gorm.DB, team *models.Team, childRequest *http.Request) error {
		var profiles []models.Profile
		if err := tx.Where("team_id = ?", team.ID).Find(&profiles).Error; err != nil {
			return err
		}
		for i := range profiles {
			before := profiles[i]
			if err := tx.Model(&profiles[i]).Update("team_id", nil).Error; err != nil {
				return err
			}
			profiles[i].TeamID = nil
			profiles[i].Team = nil
			if err := audit.LogUpdate(tx, childRequest, "profiles", profiles[i].ID, before, profiles[i]); err != nil {
				return err
			}
		}

		var roles []models.RecruitmentRole
		if err := tx.Where("team_id = ?", team.ID).Find(&roles).Error; err != nil {
			return err
		}
		for i := range roles {
			before := roles[i]
			if err := tx.Model(&roles[i]).Update("team_id", nil).Error; err != nil {
				return err
			}
			roles[i].TeamID = nil
			roles[i].Team = nil
			if err := audit.LogUpdate(tx, childRequest, "recruitmentroles", roles[i].ID, before, roles[i]); err != nil {
				return err
			}
		}
		return nil
	}, map[string]any{"operation": "delete_team_and_unassign_dependents"}, "organization", "recruitment")
}
