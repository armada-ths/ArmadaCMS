package controllers

import (
	"ArmadaCMS/main/auth"
	"ArmadaCMS/main/db"
	"ArmadaCMS/main/models"
	"ArmadaCMS/main/utils"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"gorm.io/gorm"
)

type userBody struct {
	ID        int       `json:"id"`
	Username  string    `json:"username"`
	Password  string    `json:"password"`
	Name      string    `json:"name"`
	Avatar    string    `json:"avatar"`
	RoleID    *uint     `json:"role_id"`
	UpdatedAt time.Time `json:"updated_at"`
	CreatedAt time.Time `json:"created_at"`
}

func GetUserEndpoint(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodPost {
		log.Println("err")
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var user models.User
	db.DB.First(&user)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}

// GetUsers returns a paginated list of admin users.
// @Summary List users
// @Tags users
// @Produce json
// @Param range query string false "Pagination range, e.g. [0,24]"
// @Param sort query string false "Sort, e.g. [\"username\",\"ASC\"]"
// @Param filter query string false "Filter, e.g. {\"name\":\"test\"}"
// @Success 200 {array} models.User
// @Header 200 {string} Content-Range "users 0-24/100"
// @Failure 500 {string} string "Internal server error"
// @Security BearerAuth
// @Router /customusers [get]
func GetUsers(w http.ResponseWriter, r *http.Request) {
	params, _ := utils.ParseListParams(r.URL.Query())

	var users []models.User
	query := db.DB.Model(&models.User{}).Preload("Role")

	for k, v := range params.Filter {
		query = query.Where(k+" = ?", v)
	}

	if len(params.Sort) == 2 {
		query = query.Order(params.Sort[0] + " " + params.Sort[1])
	}

	start, end := params.Range[0], params.Range[1]
	limit := end - start + 1

	var total int64
	db.DB.Model(&models.User{}).Count(&total)

	query = query.Offset(start).Limit(limit)
	query.Find(&users)

	w.Header().Set("Access-Control-Expose-Headers", "Content-Range")
	w.Header().Set("Content-Range", fmt.Sprintf("users %d-%d/%d", start, end, total))
	w.Header().Set("Content-Type", "application/json")

	json.NewEncoder(w).Encode(users)
}

// GetUserByID returns a single admin user by ID.
// @Summary Get user by ID
// @Tags users
// @Produce json
// @Param id path int true "User ID"
// @Success 200 {object} models.User
// @Failure 404 {string} string "User not found"
// @Security BearerAuth
// @Router /customusers/{id} [get]
func GetUserByID(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	var user models.User
	if err := db.DB.Preload("Role").First(&user, id).Error; err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}

// CreateUser creates a new admin user.
// @Summary Create user
// @Tags users
// @Accept json
// @Produce json
// @Param body body controllers.userBody true "User data"
// @Success 201 {object} models.User
// @Failure 400 {string} string "Invalid body"
// @Failure 500 {string} string "Create failed"
// @Security BearerAuth
// @Router /customusers [post]
func CreateUser(w http.ResponseWriter, r *http.Request) {
	var userBody userBody
	if err := json.NewDecoder(r.Body).Decode(&userBody); err != nil {
		log.Println(err)
		http.Error(w, "Invalid body", http.StatusBadRequest)
		return
	}
	log.Println(userBody.Password)
	var user = models.User{
		Username: userBody.Username,
		Password: userBody.Password,
		Name:     userBody.Name,
		Avatar:   userBody.Avatar,
		RoleID:   userBody.RoleID,
	}
	user.Password = utils.HashPassword(userBody.Password)
	if err := createWithAudit(r, "customusers", &user, func(tx *gorm.DB) error {
		return tx.Create(&user).Error
	}, func(tx *gorm.DB) error {
		return tx.Preload("Role").First(&user, user.ID).Error
	}); err != nil {
		log.Println(err)
		http.Error(w, "Create failed", http.StatusInternalServerError)
		return
	}
	log.Println(user.Password)

	writeCreatedJSONResponse(w, user)
}

// UpdateUser updates an existing admin user.
// @Summary Update user
// @Tags users
// @Accept json
// @Produce json
// @Param id path int true "User ID"
// @Param body body controllers.userBody true "Updated user data"
// @Success 200 {object} models.User
// @Failure 400 {string} string "Invalid data"
// @Failure 404 {string} string "Not found"
// @Failure 500 {string} string "Update failed"
// @Security BearerAuth
// @Router /customusers/{id} [put]
func UpdateUser(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	var user models.User
	var userUpdateBody userBody

	if err := db.DB.First(&user, id).Error; err != nil {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}

	if err := json.NewDecoder(r.Body).Decode(&userUpdateBody); err != nil {
		http.Error(w, "Invalid data", http.StatusBadRequest)
		return
	}

	newPassword := user.Password
	if len(userUpdateBody.Password) > 0 {
		newPassword = utils.HashPassword(userUpdateBody.Password)
	}

	updateUser := map[string]any{
		"username": userUpdateBody.Username,
		"password": newPassword,
		"name":     userUpdateBody.Name,
		"avatar":   userUpdateBody.Avatar,
		"role_id":  userUpdateBody.RoleID,
	}

	before := user
	if err := updateWithAudit(r, "customusers", id, before, &user, func(tx *gorm.DB) error {
		return tx.Model(&user).Updates(updateUser).Error
	}, func(tx *gorm.DB) error {
		return tx.Preload("Role").First(&user, id).Error
	}); err != nil {
		http.Error(w, "Update failed", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}

// DeleteUser deletes an admin user by ID.
// @Summary Delete user
// @Tags users
// @Produce json
// @Param id path int true "User ID"
// @Success 200 {string} string "Deleted"
// @Failure 404 {string} string "Not found"
// @Security BearerAuth
// @Router /customusers/{id} [delete]
func DeleteUser(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	writeDeleteResponseWithAudit[models.User](w, r, "customusers", id, "user not found", func(tx *gorm.DB) *gorm.DB {
		return tx.Preload("Role")
	})
}

// GetMe returns the current authenticated user's info including role/permissions.
// @Summary Get current user
// @Tags auth
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Failure 401 {string} string "Unauthorized"
// @Failure 404 {string} string "User not found"
// @Security BearerAuth
// @Router /me [get]
func GetMe(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.GetUserIDFromContext(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var user models.User
	if err := db.DB.Preload("Role").First(&user, userID).Error; err != nil {
		http.Error(w, "user not found", http.StatusNotFound)
		return
	}

	role := "admin"
	var permissions []string
	if user.Role != nil {
		role = user.Role.Name
		permissions = user.Role.Permissions
	} else {
		permissions = []string{"*"}
	}

	resp := map[string]interface{}{
		"id":          user.ID,
		"username":    user.Username,
		"name":        user.Name,
		"avatar":      user.Avatar,
		"role":        role,
		"permissions": permissions,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

type changePasswordBody struct {
	OldPassword string `json:"oldPassword"`
	NewPassword string `json:"newPassword"`
}

// ChangeOwnPassword lets an authenticated user change their own password.
// @Summary Change own password
// @Tags auth
// @Accept json
// @Produce json
// @Param body body controllers.changePasswordBody true "Old and new password"
// @Success 204 "Password changed"
// @Failure 400 {string} string "Invalid body or missing fields"
// @Failure 401 {string} string "Unauthorized"
// @Failure 403 {string} string "Old password is incorrect"
// @Failure 500 {string} string "Update failed"
// @Security BearerAuth
// @Router /me/password [put]
func ChangeOwnPassword(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.GetUserIDFromContext(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var body changePasswordBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "Invalid body", http.StatusBadRequest)
		return
	}
	if body.OldPassword == "" || body.NewPassword == "" {
		http.Error(w, "Both oldPassword and newPassword are required", http.StatusBadRequest)
		return
	}
	if body.OldPassword == body.NewPassword {
		http.Error(w, "New password must be different from the old password", http.StatusBadRequest)
		return
	}

	var user models.User
	if err := db.DB.First(&user, userID).Error; err != nil {
		http.Error(w, "user not found", http.StatusNotFound)
		return
	}

	if err := utils.CheckPasswordHash(body.OldPassword, user.Password); err != nil {
		http.Error(w, "Old password is incorrect", http.StatusForbidden)
		return
	}

	before := user
	newHash := utils.HashPassword(body.NewPassword)
	if err := updateWithAudit(r, "customusers", fmt.Sprint(userID), before, &user, func(tx *gorm.DB) error {
		return tx.Model(&user).Update("password", newHash).Error
	}, func(tx *gorm.DB) error {
		return tx.First(&user, userID).Error
	}); err != nil {
		http.Error(w, "Update failed", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
