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
	"gorm.io/gorm"
)

// GetExhibitors returns a paginated (or full) list of exhibitors.
// @Summary List exhibitors
// @Tags exhibitors
// @Produce json
// @Param range query string false "Pagination range, e.g. [0,24]"
// @Param sort query string false "Sort, e.g. [\"name\",\"ASC\"]"
// @Param filter query string false "Filter, e.g. {\"tier\":\"Gold\"}"
// @Param all query bool false "Return all exhibitors without pagination, sorted by tier"
// @Success 200 {array} models.Exhibitor
// @Header 200 {string} Content-Range "exhibitors 0-24/100"
// @Router /exhibitors [get]
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

// GetExhibitorByID returns a single exhibitor by ID.
// @Summary Get exhibitor by ID
// @Tags exhibitors
// @Produce json
// @Param id path int true "Exhibitor ID"
// @Success 200 {object} models.Exhibitor
// @Failure 404 {string} string "Exhibitor not found"
// @Router /exhibitors/{id} [get]
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

// CreateExhibitor creates a new exhibitor. Accepts multipart/form-data.
// @Summary Create exhibitor
// @Tags exhibitors
// @Accept mpfd
// @Produce json
// @Param name formData string true "Company name"
// @Param type formData string true "Company type"
// @Param fairLocation formData string true "Fair location"
// @Param tier formData string false "Tier (Standard/Bronze/Silver/Gold)"
// @Param about formData string false "About the company"
// @Param purpose formData string false "Purpose/mission"
// @Param companyWebsite formData string false "Company website URL"
// @Param climateCompensation formData bool false "Climate compensation"
// @Param cities formData string false "City/cities"
// @Param file formData file false "Logo image (JPG/PNG/WEBP/GIF)"
// @Param programs formData string false "JSON array of program IDs, e.g. [1,2,3]"
// @Param industries formData string false "JSON array of industry IDs"
// @Param employments formData string false "JSON array of employment IDs"
// @Success 201 {object} models.Exhibitor
// @Failure 400 {string} string "Bad request"
// @Failure 500 {string} string "Create failed"
// @Security BearerAuth
// @Router /exhibitors [post]
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
			if errors.Is(err, utils.ErrFileTooLarge) {
				http.Error(w, "Image file is too large. Maximum allowed size is 15 MB.", http.StatusBadRequest)
				return
			}
			http.Error(w, "Failed to upload image", http.StatusInternalServerError)
			return
		}
		exhibitor.LogoFreesizeUrl = &fileURL
	}

	// Decode M2M arrays
	if val := r.FormValue("programs"); val != "" {
		if err := json.Unmarshal([]byte(val), &exhibitor.Programs); err != nil {
			http.Error(w, "invalid programs value", http.StatusBadRequest)
			return
		}
	}
	if val := r.FormValue("industries"); val != "" {
		if err := json.Unmarshal([]byte(val), &exhibitor.Industries); err != nil {
			http.Error(w, "invalid industries value", http.StatusBadRequest)
			return
		}
	}
	if val := r.FormValue("employments"); val != "" {
		if err := json.Unmarshal([]byte(val), &exhibitor.Employments); err != nil {
			http.Error(w, "invalid employments value", http.StatusBadRequest)
			return
		}
	}

	// Create
	if err = createWithAudit(r, "exhibitors", &exhibitor, func(tx *gorm.DB) error {
		return tx.Create(&exhibitor).Error
	}, func(tx *gorm.DB) error {
		return tx.Preload("Industries").Preload("Programs").Preload("Employments").First(&exhibitor, exhibitor.ID).Error
	}, "exhibitors"); err != nil {
		http.Error(w, "Create failed", http.StatusInternalServerError)
		return
	}

	writeCreatedJSONResponse(w, exhibitor)
}

// UpdateExhibitor updates an exhibitor by ID. Accepts multipart/form-data.
// @Summary Update exhibitor
// @Tags exhibitors
// @Accept mpfd
// @Produce json
// @Param id path int true "Exhibitor ID"
// @Param name formData string false "Company name"
// @Param type formData string false "Company type"
// @Param tier formData string false "Tier (Standard/Bronze/Silver/Gold)"
// @Param file formData file false "Logo image (JPG/PNG/WEBP/GIF)"
// @Param programs formData string false "JSON array of program IDs"
// @Param industries formData string false "JSON array of industry IDs"
// @Param employments formData string false "JSON array of employment IDs"
// @Success 200 {object} models.Exhibitor
// @Failure 400 {string} string "Bad request"
// @Failure 404 {string} string "Not found"
// @Failure 500 {string} string "Update failed"
// @Security BearerAuth
// @Router /exhibitors/{id} [put]
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

	before := exhibitor

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
			if errors.Is(err, utils.ErrFileTooLarge) {
				http.Error(w, "Image file is too large. Maximum allowed size is 15 MB.", http.StatusBadRequest)
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

	if err := updateWithAudit(r, "exhibitors", id, before, &exhibitor, func(tx *gorm.DB) error {
		if err := tx.Model(&exhibitor).Select("*").Updates(exhibitor).Error; err != nil {
			return err
		}

		if updates.Programs != nil {
			if err := tx.Model(&exhibitor).Association("Programs").Replace(updates.Programs); err != nil {
				return err
			}
		}
		if updates.Industries != nil {
			if err := tx.Model(&exhibitor).Association("Industries").Replace(updates.Industries); err != nil {
				return err
			}
		}
		if updates.Employments != nil {
			if err := tx.Model(&exhibitor).Association("Employments").Replace(updates.Employments); err != nil {
				return err
			}
		}

		return nil
	}, func(tx *gorm.DB) error {
		return tx.Preload("Programs").Preload("Industries").Preload("Employments").First(&exhibitor, id).Error
	}, "exhibitors"); err != nil {
		http.Error(w, "Failed to update exhibitor", http.StatusInternalServerError)
		return
	}

	// --- Respond ---
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(exhibitor)
}

// DeleteExhibitor deletes an exhibitor by ID.
// @Summary Delete exhibitor
// @Tags exhibitors
// @Produce json
// @Param id path int true "Exhibitor ID"
// @Success 200 {string} string "Deleted"
// @Failure 404 {string} string "Exhibitor not found"
// @Security BearerAuth
// @Router /exhibitors/{id} [delete]
func DeleteExhibitor(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	writeDeleteResponseWithAudit[models.Exhibitor](w, r, "exhibitors", id, "exhibitor not found", func(tx *gorm.DB) *gorm.DB {
		return tx.Preload("Industries").Preload("Programs").Preload("Employments")
	}, "exhibitors")
}
