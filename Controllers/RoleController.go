package controllers

import (
	"ArmadaCMS/main/db"
	"ArmadaCMS/main/models"
	"ArmadaCMS/main/utils"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/mux"
	"gorm.io/gorm"
)

// GetRoles returns a paginated list of roles.
// @Summary List roles
// @Tags roles
// @Produce json
// @Param range query string false "Pagination range, e.g. [0,24]"
// @Param sort query string false "Sort, e.g. [\"name\",\"ASC\"]"
// @Param filter query string false "Filter, e.g. {\"name\":\"admin\"}"
// @Success 200 {array} models.Role
// @Header 200 {string} Content-Range "roles 0-24/10"
// @Security BearerAuth
// @Router /roles [get]
func GetRoles(w http.ResponseWriter, r *http.Request) {
	params, _ := utils.ParseListParams(r.URL.Query())

	var roles []models.Role
	query := db.DB.Model(&models.Role{})

	for k, v := range params.Filter {
		query = query.Where(k+" = ?", v)
	}

	if len(params.Sort) == 2 {
		query = query.Order(params.Sort[0] + " " + params.Sort[1])
	}

	start, end := params.Range[0], params.Range[1]
	limit := end - start + 1

	var total int64
	db.DB.Model(&models.Role{}).Count(&total)

	query = query.Offset(start).Limit(limit)
	query.Find(&roles)

	w.Header().Set("Access-Control-Expose-Headers", "Content-Range")
	w.Header().Set("Content-Range", fmt.Sprintf("roles %d-%d/%d", start, end, total))
	w.Header().Set("Content-Type", "application/json")

	json.NewEncoder(w).Encode(roles)
}

// GetRoleByID returns a single role by ID.
// @Summary Get role by ID
// @Tags roles
// @Produce json
// @Param id path int true "Role ID"
// @Success 200 {object} models.Role
// @Failure 404 {string} string "Role not found"
// @Security BearerAuth
// @Router /roles/{id} [get]
func GetRoleByID(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	var role models.Role
	if err := db.DB.First(&role, id).Error; err != nil {
		http.Error(w, "Role not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(role)
}

// CreateRole creates a new role.
// @Summary Create role
// @Tags roles
// @Accept json
// @Produce json
// @Param body body models.Role true "Role data"
// @Success 201 {object} models.Role
// @Failure 400 {string} string "Invalid body"
// @Failure 500 {string} string "Create failed"
// @Security BearerAuth
// @Router /roles [post]
func CreateRole(w http.ResponseWriter, r *http.Request) {
	var role models.Role
	if err := json.NewDecoder(r.Body).Decode(&role); err != nil {
		http.Error(w, "Invalid body", http.StatusBadRequest)
		return
	}
	if err := createWithAudit(r, "roles", &role, func(tx *gorm.DB) error {
		return tx.Create(&role).Error
	}, nil); err != nil {
		log.Println(err)
		http.Error(w, "Create failed", http.StatusInternalServerError)
		return
	}
	writeCreatedJSONResponse(w, role)
}

// UpdateRole updates a role by ID.
// @Summary Update role
// @Tags roles
// @Accept json
// @Produce json
// @Param id path int true "Role ID"
// @Param body body models.Role true "Updated role data"
// @Success 200 {object} models.Role
// @Failure 400 {string} string "Invalid body"
// @Failure 404 {string} string "Role not found"
// @Failure 500 {string} string "Update failed"
// @Security BearerAuth
// @Router /roles/{id} [put]
func UpdateRole(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	var role models.Role
	if err := db.DB.First(&role, id).Error; err != nil {
		http.Error(w, "Role not found", http.StatusNotFound)
		return
	}

	var body models.Role
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "Invalid body", http.StatusBadRequest)
		return
	}

	before := role
	if err := updateWithAudit(r, "roles", id, before, &role, func(tx *gorm.DB) error {
		return tx.Model(&role).Updates(models.Role{
			Name:        body.Name,
			Permissions: body.Permissions,
		}).Error
	}, func(tx *gorm.DB) error {
		return tx.First(&role, id).Error
	}); err != nil {
		http.Error(w, "Update failed", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(role)
}

// DeleteRole deletes a role by ID.
// @Summary Delete role
// @Tags roles
// @Produce json
// @Param id path int true "Role ID"
// @Success 200 {string} string "Deleted"
// @Failure 404 {string} string "Role not found"
// @Security BearerAuth
// @Router /roles/{id} [delete]
func DeleteRole(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	writeDeleteResponseWithAudit[models.Role](w, r, "roles", id, "role not found", nil, nil)
}

// SeedRoles creates the default roles if they don't exist yet.
func SeedRoles(database *gorm.DB) error {
	roles := []models.Role{
		{
			Name:        "admin",
			Permissions: models.Permissions{"*"},
		},
	}

	for _, role := range roles {
		var existing models.Role
		if err := database.Where("name = ?", role.Name).First(&existing).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				if err := database.Create(&role).Error; err != nil {
					return fmt.Errorf("failed to seed role %q: %w", role.Name, err)
				}
				log.Printf("Seeded role: %s", role.Name)
			} else {
				return err
			}
		}
	}
	return nil
}
