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
	var updateUser = models.User{
		Username: userUpdateBody.Username,
		Password: userUpdateBody.Password,
		Name:     userUpdateBody.Name,
		Avatar:   userUpdateBody.Avatar,
		RoleID:   userUpdateBody.RoleID,
		ID:       uint(userUpdateBody.ID),
	}
	if len(updateUser.Password) > 0 {
		updateUser.Password = utils.HashPassword(updateUser.Password)
	} else {
		updateUser.Password = user.Password
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
func DeleteUser(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	writeDeleteResponseWithAudit[models.User](w, r, "customusers", id, "user not found", func(tx *gorm.DB) *gorm.DB {
		return tx.Preload("Role")
	})
}

// GetMe returns the current authenticated user's info including role/permissions.
// Used by the frontend to resolve identity and permissions after login.
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
