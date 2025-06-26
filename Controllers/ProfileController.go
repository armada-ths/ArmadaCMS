package controllers

import (
	"ArmadaCMS/main/db"
	"ArmadaCMS/main/models"
	"ArmadaCMS/main/utils"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
)

func GetProfiles(w http.ResponseWriter, r *http.Request) {
	params, _ := utils.ParseListParams(r.URL.Query())

	var profiles []models.Profile
	query := db.DB.Model(&models.Profile{})

	for k, v := range params.Filter {
		query = query.Where(k+" = ?", v)
	}

	if len(params.Sort) == 2 {
		query = query.Order(params.Sort[0] + " " + params.Sort[1])
	}

	start, end := params.Range[0], params.Range[1]
	limit := end - start + 1

	var total int64
	db.DB.Model(&models.Profile{}).Count(&total)

	query = query.Offset(start).Limit(limit)
	query.Preload("Team").Find(&profiles)

	w.Header().Set("Access-Control-Expose-Headers", "Content-Range")
	w.Header().Set("Content-Range", fmt.Sprintf("profiles %d-%d/%d", start, end, total))
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(profiles)
}
func GetProfileByID(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	var profile models.Profile
	if err := db.DB.Preload("Team").First(&profile, id).Error; err != nil {
		http.Error(w, "profile not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(profile)
}

func CreateProfile(w http.ResponseWriter, r *http.Request) {
	var profile models.Profile
	// if err := json.NewDecoder(r.Body).Decode(&profile); err != nil {
	// 	log.Println(err)
	// 	http.Error(w, "Invalid body", http.StatusBadRequest)
	// 	return
	// }
	if err := r.ParseMultipartForm(20 << 20); err != nil { // 20 MB max memory
		log.Println(err)
		http.Error(w, "Unable to parse multipart form", http.StatusBadRequest)
		return
	}

	teamIDStr := r.FormValue("team_id")
	var err error
	teamID, err := strconv.Atoi(teamIDStr)
	if err != nil {
		log.Println("Invalid team_id:", teamIDStr)
		http.Error(w, "team_id must be an integer", http.StatusBadRequest)
		return
	}
	profile.TeamID = int32(teamID)
	profile.Name = r.FormValue("name")
	profile.Title = r.FormValue("title")
	profile.Linkedin = r.FormValue("linkedin")
	profile.Email = r.FormValue("email")

	photoUrl := r.FormValue("photoUrl")
	if photoUrl != "" {
		profile.Photo = photoUrl
	} else {
		photoFile, header, err := r.FormFile("file")
		if err != nil {
			log.Println("Error retrieving the file:", err)
			http.Error(w, "File is required", http.StatusBadRequest)
			return
		}
		defer photoFile.Close()
		fileURL, err := utils.UploadToS3(photoFile, header)
		if err != nil {
			log.Println("Error uploading the file:", err)
			http.Error(w, "Error uploading file", http.StatusBadRequest)
			return
		}
		profile.Photo = fileURL
	}

	if err := db.DB.Create(&profile).Error; err != nil {
		http.Error(w, "Create failed", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(profile)
}
func UpdateProfile(w http.ResponseWriter, r *http.Request) {
	var updates models.Profile
	var profile models.Profile

	// if err := json.NewDecoder(r.Body).Decode(&profile); err != nil {
	// 	log.Println(err)
	// 	http.Error(w, "Invalid body", http.StatusBadRequest)
	// 	return
	// }
	id := mux.Vars(r)["id"] // if you have the ID in URL

	if err := db.DB.First(&profile, id).Error; err != nil {
		http.Error(w, "Profile not found", http.StatusNotFound)
		return
	}
	if err := r.ParseMultipartForm(20 << 20); err != nil { // 20 MB max memory
		log.Println(err)
		http.Error(w, "Unable to parse multipart form", http.StatusBadRequest)
		return
	}

	teamIDStr := r.FormValue("team_id")
	var err error
	teamID, err := strconv.Atoi(teamIDStr)
	if err != nil {
		log.Println("Invalid team_id:", teamIDStr)
		http.Error(w, "team_id must be an integer", http.StatusBadRequest)
		return
	}
	updates.TeamID = int32(teamID)
	updates.Name = r.FormValue("name")
	updates.Title = r.FormValue("title")
	updates.Linkedin = r.FormValue("linkedin")
	updates.Email = r.FormValue("email")

	photoUrl := r.FormValue("photoUrl")
	if photoUrl != "" {
		updates.Photo = photoUrl
	} else {
		photoFile, header, err := r.FormFile("file")
		if err != nil && err != http.ErrMissingFile {
			log.Println("Error retrieving the file:", err)
			http.Error(w, "File is corrupt", http.StatusBadRequest)
			return
		} else if err == nil {
			defer photoFile.Close()
			fileURL, err := utils.UploadToS3(photoFile, header)
			if err != nil {
				log.Println("Error uploading the file:", err)
				http.Error(w, "Error uploading file", http.StatusBadRequest)
				return
			}
			updates.Photo = fileURL
		}
	}

	db.DB.Model(&profile).Updates(updates)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(updates)
}

// TODO change to formdata multipart yadda yadda
// func UpdateProfile(w http.ResponseWriter, r *http.Request) {
// 	vars := mux.Vars(r)
// 	id := vars["id"]

// 	var profile models.Profile
// 	if err := db.DB.First(&profile, id).Error; err != nil {
// 		http.Error(w, "Not found", http.StatusNotFound)
// 		return
// 	}

// 	var updates models.Profile
// 	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
// 		http.Error(w, "Invalid data", http.StatusBadRequest)
// 		return
// 	}

// 	db.DB.Model(&profile).Updates(updates)

//		w.Header().Set("Content-Type", "application/json")
//		json.NewEncoder(w).Encode(profile)
//	}
func DeleteProfile(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	if err := db.DB.Delete(&models.Profile{}, id).Error; err != nil {
		http.Error(w, "Delete failed", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
