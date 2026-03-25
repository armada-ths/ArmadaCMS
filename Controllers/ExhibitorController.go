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

	"github.com/gorilla/mux"
)

func GetExhibitors(w http.ResponseWriter, r *http.Request) {
	params, _ := utils.ParseListParams(r.URL.Query())
	log.Print(params)
	var exhibitors []models.Exhibitor
	query := db.DB.Model(&models.Exhibitor{})

	for k, v := range params.Filter {
		query = query.Where(k+" = ?", v)
	}

	all := r.URL.Query().Get("limit") == "all" || r.URL.Query().Get("all") == "true"

	if !all && len(params.Sort) == 2 {
		query = query.Order(params.Sort[0] + " " + params.Sort[1])
	}

	var total int64
	db.DB.Model(&models.Exhibitor{}).Count(&total)

	if all {
		query = query.Order(`
			CASE 
				WHEN tier = 'Gold' THEN 1
				WHEN tier = 'Silver' THEN 2
				WHEN tier = 'Bronze' THEN 3
				ELSE 4
			END
		`)

		// No limit — return all exhibitors
		query.Preload("Industries").Preload("Programs").Preload("Employments").Find(&exhibitors)

		// Set Content-Range header to full range
		w.Header().Set("Access-Control-Expose-Headers", "Content-Range")
		w.Header().Set("Content-Range", fmt.Sprintf("exhibitors 0-%d/%d", total-1, total))
	} else {
		// Default paginated behavior
		start, end := params.Range[0], params.Range[1]
		limit := end - start + 1
		query = query.Offset(start).Limit(limit)
		query.Preload("Industries").Preload("Programs").Preload("Employments").Find(&exhibitors)

		w.Header().Set("Access-Control-Expose-Headers", "Content-Range")
		w.Header().Set("Content-Range", fmt.Sprintf("exhibitors %d-%d/%d", start, end, total))
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(exhibitors)
}

func GetExhibitorByID(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	var exhibitor models.Exhibitor
	if err := db.DB.Preload("Industries").Preload("Programs").Preload("Employments").First(&exhibitor, id).Error; err != nil {
		http.Error(w, "Exhibitor not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(exhibitor)
}

func CreateExhibitor(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(20 << 20); err != nil {
		http.Error(w, "Unable to parse multipart form", http.StatusBadRequest)
		return
	}

	var exhibitor models.Exhibitor
	var err error

	exhibitor.Name = r.FormValue("name")
	exhibitor.Type = r.FormValue("type")
	exhibitor.FairLocation = r.FormValue("fairLocation")
	exhibitor.CompanyWebsite = utils.StringPtr(r.FormValue("companyWebsite"))
	exhibitor.About = utils.StringPtr(r.FormValue("about"))
	exhibitor.Purpose = utils.StringPtr(r.FormValue("purpose"))
	exhibitor.ClimateCompensation = r.FormValue("climateCompensation") == "true"
	exhibitor.Flyer = r.FormValue("flyer")

	if tierStr := r.FormValue("tier"); tierStr != "" {
		t := models.Tier(tierStr)
		exhibitor.Tier = &t
	}

	// Handle logo
	logoUrl := r.FormValue("logoFreesize")
	if logoUrl != "" {
		exhibitor.LogoFreesizeUrl = &logoUrl
	} else if file, header, err := r.FormFile("file"); err == nil {
		defer file.Close()
		fileURL, err := utils.UploadToS3(file, header)
		if err != nil {
			if errors.Is(err, utils.ErrUnsupportedImageFormat) {
				http.Error(w, "Unsupported image format. Allowed formats: JPG, JPEG, PNG, WEBP, GIF.", http.StatusBadRequest)
				return
			}
			http.Error(w, "Failed to upload image", http.StatusInternalServerError)
			return
		}
		exhibitor.LogoFreesizeUrl = &fileURL
	}

	// Decode M2M arrays
	if val := r.FormValue("programs"); val != "" {
		json.Unmarshal([]byte(val), &exhibitor.Programs)
	}
	if val := r.FormValue("industries"); val != "" {
		json.Unmarshal([]byte(val), &exhibitor.Industries)
	}
	if val := r.FormValue("employments"); val != "" {
		json.Unmarshal([]byte(val), &exhibitor.Employments)
	}

	// Create
	if err = db.DB.Create(&exhibitor).Error; err != nil {
		http.Error(w, "Create failed", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(exhibitor)
}

func UpdateExhibitor(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]

	var exhibitor models.Exhibitor
	if err := db.DB.Preload("Programs").
		Preload("Industries").
		Preload("Employments").
		First(&exhibitor, id).Error; err != nil {
		http.Error(w, "Exhibitor not found", http.StatusNotFound)
		return
	}

	// --- Parse multipart form ---
	if err := r.ParseMultipartForm(20 << 20); err != nil {
		http.Error(w, "Unable to parse multipart form", http.StatusBadRequest)
		return
	}

	// --- Basic scalar fields ---
	exhibitor.Name = r.FormValue("name")
	exhibitor.Type = r.FormValue("type")
	exhibitor.FairLocation = r.FormValue("fairLocation")
	exhibitor.CompanyWebsite = utils.StringPtr(r.FormValue("companyWebsite"))
	exhibitor.About = utils.StringPtr(r.FormValue("about"))
	exhibitor.Purpose = utils.StringPtr(r.FormValue("purpose"))
	exhibitor.ClimateCompensation = r.FormValue("climateCompensation") == "true"
	exhibitor.Flyer = r.FormValue("flyer")

	// --- Tier (custom type) ---
	if tierStr := r.FormValue("tier"); tierStr != "" {
		t := models.Tier(tierStr)
		exhibitor.Tier = &t
	}

	// --- Handle logo upload or link ---
	logoUrl := r.FormValue("logoFreesize")
	if logoUrl != "" {
		exhibitor.LogoFreesizeUrl = &logoUrl
	} else if file, header, err := r.FormFile("file"); err == nil {
		defer file.Close()
		fileURL, err := utils.UploadToS3(file, header)
		if err != nil {
			if errors.Is(err, utils.ErrUnsupportedImageFormat) {
				http.Error(w, "Unsupported image format. Allowed formats: JPG, JPEG, PNG, WEBP, GIF.", http.StatusBadRequest)
				return
			}
			http.Error(w, "Failed to upload image", http.StatusInternalServerError)
			return
		}
		exhibitor.LogoFreesizeUrl = &fileURL
	}

	// --- Decode many-to-many JSON arrays ---
	var updates models.Exhibitor

	if val := r.FormValue("programs"); val != "" {
		if err := json.Unmarshal([]byte(val), &updates.Programs); err != nil {
			log.Printf("❌ Failed to parse programs JSON: %v", err)
		}
	}
	if val := r.FormValue("industries"); val != "" {
		if err := json.Unmarshal([]byte(val), &updates.Industries); err != nil {
			log.Printf("❌ Failed to parse industries JSON: %v", err)
		}
	}
	if val := r.FormValue("employments"); val != "" {
		if err := json.Unmarshal([]byte(val), &updates.Employments); err != nil {
			log.Printf("❌ Failed to parse employments JSON: %v", err)
		}
	}

	// --- Update main exhibitor fields ---
	if err := db.DB.Model(&exhibitor).Select("*").Updates(exhibitor).Error; err != nil {
		http.Error(w, "Failed to update exhibitor", http.StatusInternalServerError)
		return
	}

	// --- Replace associations ---
	if updates.Programs != nil {
		if err := db.DB.Model(&exhibitor).Association("Programs").Replace(updates.Programs); err != nil {
			http.Error(w, "Failed to update programs", http.StatusInternalServerError)
			return
		}
	}
	if updates.Industries != nil {
		if err := db.DB.Model(&exhibitor).Association("Industries").Replace(updates.Industries); err != nil {
			http.Error(w, "Failed to update industries", http.StatusInternalServerError)
			return
		}
	}
	if updates.Employments != nil {
		if err := db.DB.Model(&exhibitor).Association("Employments").Replace(updates.Employments); err != nil {
			http.Error(w, "Failed to update employments", http.StatusInternalServerError)
			return
		}
	}

	// --- Respond ---
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(exhibitor)
}

func DeleteExhibitor(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	if err := db.DB.Delete(&models.Exhibitor{}, id).Error; err != nil {
		http.Error(w, "Delete failed", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
