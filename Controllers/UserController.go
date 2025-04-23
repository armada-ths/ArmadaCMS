package controllers

import (
	"ArmadaCMS/main/db"
	"ArmadaCMS/main/models"
	"ArmadaCMS/main/utils"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/mux"
)

type userBody struct {
	ID        int       `json:"id"`
	Username  string    `json:"username"`
	Password  string    `json:"password"`
	Name      string    `json:"name"`
	Avatar    string    `json:"avatar"`
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
	query := db.DB.Model(&models.User{})

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
	if err := db.DB.First(&user, id).Error; err != nil {
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
	}
	user.Password = utils.HashPassword(userBody.Password)
	if err := db.DB.Create(&user).Error; err != nil {
		log.Println(err)
		http.Error(w, "Create failed", http.StatusInternalServerError)
		return
	}
	log.Println(user.Password)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
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
		ID:       uint(userUpdateBody.ID),
	}
	if len(updateUser.Password) > 0 {
		updateUser.Password = utils.HashPassword(updateUser.Password)
	} else {
		updateUser.Password = user.Password
	}

	db.DB.Model(&user).Updates(updateUser)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}
func DeleteUser(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	if err := db.DB.Delete(&models.User{}, id).Error; err != nil {
		http.Error(w, "Delete failed", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
