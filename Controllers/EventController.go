package controllers

import (
	"ArmadaCMS/main/db"
	"ArmadaCMS/main/models"
	"ArmadaCMS/main/utils"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/shopspring/decimal"
)

func GetEvents(w http.ResponseWriter, r *http.Request) {
	params, _ := utils.ParseListParams(r.URL.Query())

	var events []models.Event
	query := db.DB.Model(&models.Event{})

	for k, v := range params.Filter {
		query = query.Where(k+" = ?", v)
	}

	if len(params.Sort) == 2 {
		query = query.Order(params.Sort[0] + " " + params.Sort[1])
	}

	start, end := params.Range[0], params.Range[1]
	limit := end - start + 1

	var total int64
	db.DB.Model(&models.Event{}).Count(&total)

	query.Offset(start).Limit(limit).Find(&events)

	w.Header().Set("Access-Control-Expose-Headers", "Content-Range")
	w.Header().Set("Content-Range", fmt.Sprintf("events %d-%d/%d", start, end, total))
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(events)
}

func GetEventByID(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	var event models.Event
	if err := db.DB.First(&event, id).Error; err != nil {
		http.Error(w, "Event not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(event)
}

func CreateEvent(w http.ResponseWriter, r *http.Request) {
	contentType := r.Header.Get("Content-Type")
	var event models.Event

	if contentType == "" || len(contentType) < 19 || contentType[:19] != "multipart/form-data" {
		if err := json.NewDecoder(r.Body).Decode(&event); err != nil {
			http.Error(w, "Invalid JSON body", http.StatusBadRequest)
			return
		}
	} else {
		if err := r.ParseMultipartForm(20 << 20); err != nil {
			http.Error(w, "Unable to parse multipart form", http.StatusBadRequest)
			return
		}

		parseTime := func(key string) time.Time {
			val := r.FormValue(key)
			t, err := time.Parse(time.RFC3339, val)
			if err != nil {
				return time.Now().UTC()
			}
			return t
		}

		event.Name = r.FormValue("name")
		event.EventroID = r.FormValue("eventroId")
		event.Location = r.FormValue("location")
		event.EventStart = parseTime("eventStart")
		event.EventEnd = parseTime("eventEnd")

		if val := r.FormValue("registrationEnd"); val != "" {
			if t, err := time.Parse(time.RFC3339, val); err == nil {
				event.RegistrationEnd = &t
			}
		}

		event.Description = utils.StringPtr(r.FormValue("description"))
		event.Food = utils.StringPtr(r.FormValue("food"))
		event.SignupLink = utils.StringPtr(r.FormValue("signupLink"))
		event.RegistrationRequired = r.FormValue("registrationRequired") == "true"

		if feeStr := r.FormValue("fee"); feeStr != "" {
			if fee, err := decimal.NewFromString(feeStr); err == nil {
				event.Fee = &fee
			}
		}

		imageUrl := r.FormValue("imageUrl")
		if imageUrl != "" {
			event.ImageURL = &imageUrl
		} else {
			file, header, err := r.FormFile("file")
			if err == nil {
				defer file.Close()
				fileURL, err := utils.UploadToS3(file, header)
				if err != nil {
					http.Error(w, "Failed to upload image", http.StatusInternalServerError)
					return
				}
				event.ImageURL = &fileURL
			}
		}
	}

	if event.Fee == nil {
		zero := decimal.NewFromInt(0)
		event.Fee = &zero
	}

	if err := db.DB.Create(&event).Error; err != nil {
		http.Error(w, "Create failed", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(event)
}

func UpdateEvent(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	var event models.Event
	if err := db.DB.First(&event, id).Error; err != nil {
		http.Error(w, "Event not found", http.StatusNotFound)
		return
	}

	if err := r.ParseMultipartForm(20 << 20); err != nil {
		http.Error(w, "Unable to parse multipart form", http.StatusBadRequest)
		return
	}

	parseTime := func(key string) time.Time {
		val := r.FormValue(key)
		t, err := time.Parse(time.RFC3339, val)
		if err != nil {
			return time.Now().UTC()
		}
		return t
	}

	var updates models.Event
	updates.Name = r.FormValue("name")
	updates.Description = utils.StringPtr(r.FormValue("description"))
	updates.Location = r.FormValue("location")
	updates.EventStart = parseTime("eventStart")
	updates.EventEnd = parseTime("eventEnd")
	updates.Food = utils.StringPtr(r.FormValue("food"))
	updates.SignupLink = utils.StringPtr(r.FormValue("signupLink"))
	updates.RegistrationRequired = r.FormValue("registrationRequired") == "true"

	if val := r.FormValue("registrationEnd"); val != "" {
		if t, err := time.Parse(time.RFC3339, val); err == nil {
			updates.RegistrationEnd = &t
		}
	}

	if feeStr := r.FormValue("fee"); feeStr != "" {
		if fee, err := decimal.NewFromString(feeStr); err == nil {
			updates.Fee = &fee
		}
	}

	imageUrl := r.FormValue("imageUrl")
	if imageUrl != "" {
		updates.ImageURL = &imageUrl
	} else {
		file, header, err := r.FormFile("file")
		if err == nil {
			defer file.Close()
			fileURL, err := utils.UploadToS3(file, header)
			if err == nil {
				updates.ImageURL = &fileURL
			}
		}
	}

	db.DB.Model(&event).Select("name", "description", "location", "event_start",
		"event_end", "food", "registration_end", "signup_link",
		"registration_required", "image_url").Updates(updates)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(updates)
}

func DeleteEvent(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	if err := db.DB.Delete(&models.Event{}, id).Error; err != nil {
		http.Error(w, "Delete failed", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
