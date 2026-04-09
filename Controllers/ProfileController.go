package controllers

import (
	"ArmadaCMS/main/db"
	"ArmadaCMS/main/models"
	"ArmadaCMS/main/utils"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/gorilla/mux"
	"gorm.io/gorm"
)

// GetProfiles returns a paginated list of profiles.
// @Summary List profiles
// @Tags profiles
// @Produce json
// @Param range query string false "Pagination range, e.g. [0,24]"
// @Param sort query string false "Sort, e.g. [\"name\",\"ASC\"]"
// @Param filter query string false "Filter, e.g. {\"team_id\":1}"
// @Success 200 {array} models.Profile
// @Header 200 {string} Content-Range "profiles 0-24/100"
// @Router /profiles [get]
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

// GetProfileByID returns a single profile by ID.
// @Summary Get profile by ID
// @Tags profiles
// @Produce json
// @Param id path int true "Profile ID"
// @Success 200 {object} models.Profile
// @Failure 404 {string} string "profile not found"
// @Router /profiles/{id} [get]
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

// CreateProfile creates a new profile. Accepts multipart/form-data.
// @Summary Create profile
// @Tags profiles
// @Accept mpfd
// @Produce json
// @Param name formData string true "Full name"
// @Param title formData string false "Job title"
// @Param rank formData string false "Rank within team"
// @Param email formData string false "Email address"
// @Param linkedin formData string false "LinkedIn URL"
// @Param team_id formData int false "Team ID"
// @Param file formData file false "Profile photo (JPG/PNG/WEBP/GIF)"
// @Param photoUrl formData string false "Photo URL (alternative to file upload)"
// @Success 201 {object} models.Profile
// @Failure 400 {string} string "Bad request"
// @Failure 500 {string} string "Create failed"
// @Security BearerAuth
// @Router /profiles [post]
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

	teamIDStr := strings.TrimSpace(r.FormValue("team_id"))
	if teamIDStr != "" {
		teamID, err := strconv.Atoi(teamIDStr)
		if err != nil {
			log.Println("Invalid team_id:", teamIDStr)
			http.Error(w, "team_id must be an integer", http.StatusBadRequest)
			return
		}
		teamIDInt32 := int32(teamID)
		profile.TeamID = &teamIDInt32
	}
	profile.Name = r.FormValue("name")
	profile.Rank = r.FormValue("rank")
	profile.Title = r.FormValue("title")
	profile.Linkedin = r.FormValue("linkedin")
	profile.Email = r.FormValue("email")

	photoUrl := r.FormValue("photoUrl")
	if photoUrl != "" {
		profile.Photo = photoUrl
	} else {
		photoFile, header, err := r.FormFile("file")
		if err != nil && err != http.ErrMissingFile {
			log.Println("Error retrieving the file:", err)
			http.Error(w, "File is corrupt", http.StatusBadRequest)
			return
		}
		if err == nil {
			defer photoFile.Close()
			fileURL, err := utils.UploadToS3(photoFile, header)
			if err != nil {
				log.Println("Error uploading the file:", err)
				if errors.Is(err, utils.ErrUnsupportedImageFormat) {
					http.Error(w, "Unsupported image format. Allowed formats: JPG, JPEG, PNG, WEBP, GIF.", http.StatusBadRequest)
					return
				}
				if errors.Is(err, utils.ErrFileTooLarge) {
					http.Error(w, "Image file is too large. Maximum allowed size is 15 MB.", http.StatusBadRequest)
					return
				}
				http.Error(w, "Error uploading file", http.StatusInternalServerError)
				return
			}
			profile.Photo = fileURL
		}
	}

	if err := createWithAudit(r, "profiles", &profile, func(tx *gorm.DB) error {
		return tx.Create(&profile).Error
	}, func(tx *gorm.DB) error {
		return tx.Preload("Team").First(&profile, profile.ID).Error
	}); err != nil {
		http.Error(w, "Create failed", http.StatusInternalServerError)
		return
	}

	writeCreatedJSONResponse(w, profile)
}

// UpdateProfile updates a profile by ID. Accepts multipart/form-data.
// @Summary Update profile
// @Tags profiles
// @Accept mpfd
// @Produce json
// @Param id path int true "Profile ID"
// @Param name formData string false "Full name"
// @Param title formData string false "Job title"
// @Param rank formData string false "Rank within team"
// @Param email formData string false "Email address"
// @Param linkedin formData string false "LinkedIn URL"
// @Param team_id formData int false "Team ID"
// @Param file formData file false "Profile photo (JPG/PNG/WEBP/GIF)"
// @Param photoUrl formData string false "Photo URL (alternative to file upload)"
// @Success 200 {object} models.Profile
// @Failure 400 {string} string "Bad request"
// @Failure 404 {string} string "Profile not found"
// @Failure 500 {string} string "Update failed"
// @Security BearerAuth
// @Router /profiles/{id} [put]
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

	teamIDStr := strings.TrimSpace(r.FormValue("team_id"))
	teamIDProvided := false
	if r.MultipartForm != nil {
		_, teamIDProvided = r.MultipartForm.Value["team_id"]
	}
	if teamIDStr != "" {
		teamID, err := strconv.Atoi(teamIDStr)
		if err != nil {
			log.Println("Invalid team_id:", teamIDStr)
			http.Error(w, "team_id must be an integer", http.StatusBadRequest)
			return
		}
		teamIDInt32 := int32(teamID)
		updates.TeamID = &teamIDInt32
	}
	updates.Name = r.FormValue("name")
	updates.Rank = r.FormValue("rank")
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
				if errors.Is(err, utils.ErrUnsupportedImageFormat) {
					http.Error(w, "Unsupported image format. Allowed formats: JPG, JPEG, PNG, WEBP, GIF.", http.StatusBadRequest)
					return
				}
				if errors.Is(err, utils.ErrFileTooLarge) {
					http.Error(w, "Image file is too large. Maximum allowed size is 15 MB.", http.StatusBadRequest)
					return
				}
				http.Error(w, "Error uploading file", http.StatusInternalServerError)
				return
			}
			updates.Photo = fileURL
		}
	}

	updateMap := map[string]interface{}{
		"name":     updates.Name,
		"rank":     updates.Rank,
		"title":    updates.Title,
		"linkedin": updates.Linkedin,
		"email":    updates.Email,
	}
	if teamIDProvided {
		updateMap["team_id"] = updates.TeamID
	}
	if updates.Photo != "" {
		updateMap["photo"] = updates.Photo
	}

	before := profile
	if err := updateWithAudit(r, "profiles", id, before, &profile, func(tx *gorm.DB) error {
		return tx.Model(&profile).Updates(updateMap).Error
	}, func(tx *gorm.DB) error {
		return tx.Preload("Team").First(&profile, id).Error
	}); err != nil {
		http.Error(w, "Update failed", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(updates)
}

// DeleteProfile deletes a profile by ID.
// @Summary Delete profile
// @Tags profiles
// @Produce json
// @Param id path int true "Profile ID"
// @Success 200 {string} string "Deleted"
// @Failure 404 {string} string "profile not found"
// @Security BearerAuth
// @Router /profiles/{id} [delete]
func DeleteProfile(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	writeDeleteResponseWithAudit[models.Profile](w, r, "profiles", id, "profile not found", func(tx *gorm.DB) *gorm.DB {
		return tx.Preload("Team")
	})
}
