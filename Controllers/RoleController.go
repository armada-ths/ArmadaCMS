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

func CreateRole(w http.ResponseWriter, r *http.Request) {
	var role models.Role
	if err := json.NewDecoder(r.Body).Decode(&role); err != nil {
		http.Error(w, "Invalid body", http.StatusBadRequest)
		return
	}
	if err := db.DB.Create(&role).Error; err != nil {
		log.Println(err)
		http.Error(w, "Create failed", http.StatusInternalServerError)
		return
	}
	writeCreatedJSONResponse(w, role)
}

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

	db.DB.Model(&role).Updates(models.Role{
		Name:        body.Name,
		Permissions: body.Permissions,
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(role)
}

func DeleteRole(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	writeDeleteResponse(w, db.DB.Delete(&models.Role{}, id), "role not found")
}

// SeedRoles creates the default roles if they don't exist yet.
func SeedRoles(database *gorm.DB) error {
	roles := []models.Role{
		{
			Name:        "admin",
			Permissions: models.Permissions{"*"},
		},
		{
			Name: "member",
			Permissions: models.Permissions{
				"profiles.list",
				"profiles.show",
				"profiles.create",
				"profiles.edit",
				"teams.list",
				"teams.show",
			},
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
