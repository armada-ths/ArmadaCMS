package controllers

import (
	"ArmadaCMS/main/db"
	"ArmadaCMS/main/models"
	"ArmadaCMS/main/utils"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"gorm.io/gorm"
)

// GetEvents returns a paginated list of events.
// @Summary List events
// @Tags events
// @Produce json
// @Param range query string false "Pagination range, e.g. [0,24]"
// @Param sort query string false "Sort, e.g. [\"eventStart\",\"ASC\"]"
// @Param filter query string false "Filter, e.g. {\"show\":true}"
// @Param public query bool false "If true, only return events with show=true"
// @Success 200 {array} models.Event
// @Header 200 {string} Content-Range "events 0-24/100"
// @Router /events [get]
func GetEvents(w http.ResponseWriter, r *http.Request) {
	params, _ := utils.ParseListParams(r.URL.Query())

	var events []models.Event
	query := db.DB.Model(&models.Event{})

	// Apply dynamic filters from React Admin
	for k, v := range params.Filter {
		query = query.Where(k+" = ?", v)
	}

	// Apply `show=true` only if query param `public=true` is passed
	publicOnly := r.URL.Query().Get("public")
	if publicOnly == "true" {
		query = query.Where("show = ?", true)
	}

	// Sorting
	if len(params.Sort) == 2 {
		column := params.Sort[0]

		// Fix camelCase fields from React-Admin
		switch column {
		case "eventStart":
			column = "event_start"
		case "eventEnd":
			column = "event_end"
		case "registrationEnd":
			column = "registration_end"
		case "registrationRequired":
			column = "registration_required"
			// add any others as needed
		}

		query = query.Order(fmt.Sprintf("%s %s", column, params.Sort[1]))
	}

	// Pagination
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

// GetEventByID returns a single event by ID.
// @Summary Get event by ID
// @Tags events
// @Produce json
// @Param id path int true "Event ID"
// @Success 200 {object} models.Event
// @Failure 404 {string} string "Event not found"
// @Router /events/{id} [get]
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

// CreateEvent creates a new event. Accepts multipart/form-data.
// @Summary Create event
// @Tags events
// @Accept mpfd
// @Produce json
// @Param name formData string true "Event name"
// @Param location formData string true "Location"
// @Param eventStart formData string true "Start time (RFC3339)"
// @Param eventEnd formData string true "End time (RFC3339)"
// @Param description formData string false "Description"
// @Param food formData string false "Food information"
// @Param registrationRequired formData bool false "Registration required"
// @Param fee formData string false "Fee information"
// @Param signupLink formData string false "Signup link URL"
// @Param eventMaxCapacity formData int false "Max capacity"
// @Param registrationEnd formData string false "Registration end time (RFC3339)"
// @Param show formData bool false "Whether to show the event publicly"
// @Param file formData file false "Event image"
// @Success 201 {object} models.Event
// @Failure 400 {string} string "Bad request"
// @Failure 500 {string} string "Create failed"
// @Security BearerAuth
// @Router /events [post]
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

		parseTime := func(key string) (time.Time, error) {
			val := r.FormValue(key)
			t, err := time.Parse(time.RFC3339, val)
			if err != nil {
				return time.Time{}, err
			}
			return t, nil
		}

		event.Name = r.FormValue("name")
		event.EventroID = r.FormValue("eventroId")
		event.Location = r.FormValue("location")
		eventStart, err := parseTime("eventStart")
		if err != nil {
			http.Error(w, "Invalid eventStart: expected RFC3339 datetime", http.StatusBadRequest)
			return
		}
		event.EventStart = eventStart

		eventEnd, err := parseTime("eventEnd")
		if err != nil {
			http.Error(w, "Invalid eventEnd: expected RFC3339 datetime", http.StatusBadRequest)
			return
		}
		event.EventEnd = eventEnd

		if val := r.FormValue("registrationEnd"); val != "" {
			if t, err := time.Parse(time.RFC3339, val); err == nil {
				event.RegistrationEnd = &t
			}
		}

		event.Description = utils.StringPtr(r.FormValue("description"))
		event.Food = utils.StringPtr(r.FormValue("food"))
		event.SignupLink = utils.StringPtr(r.FormValue("signupLink"))
		event.RegistrationRequired = r.FormValue("registrationRequired") == "true"
		event.Fee = utils.StringPtr(r.FormValue("fee"))

		imageUrl := r.FormValue("imageUrl")
		if imageUrl != "" {
			event.ImageURL = &imageUrl
		} else {
			file, header, err := r.FormFile("file")
			if err == nil {
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
				event.ImageURL = &fileURL
			}
		}
	}

	if err := createWithAudit(r, "events", &event, func(tx *gorm.DB) error {
		return tx.Create(&event).Error
	}, nil, "events"); err != nil {
		http.Error(w, "Create failed", http.StatusInternalServerError)
		return
	}

	writeCreatedJSONResponse(w, event)
}

// UpdateEvent updates an event by ID. Accepts multipart/form-data.
// @Summary Update event
// @Tags events
// @Accept mpfd
// @Produce json
// @Param id path int true "Event ID"
// @Param name formData string false "Event name"
// @Param location formData string false "Location"
// @Param eventStart formData string false "Start time (RFC3339)"
// @Param eventEnd formData string false "End time (RFC3339)"
// @Param show formData bool false "Whether to show the event publicly"
// @Param file formData file false "Event image"
// @Success 200 {object} models.Event
// @Failure 400 {string} string "Bad request"
// @Failure 404 {string} string "Not found"
// @Failure 500 {string} string "Update failed"
// @Security BearerAuth
// @Router /events/{id} [put]
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

	parseTime := func(key string) (*time.Time, error) {
		val := r.FormValue(key)
		if val == "" {
			return nil, nil
		}

		t, err := time.Parse(time.RFC3339, val)
		if err != nil {
			return nil, err
		}

		return &t, nil
	}

	var updates models.Event
	updates.Name = r.FormValue("name")
	updates.Description = utils.StringPtr(r.FormValue("description"))
	updates.Location = r.FormValue("location")
	eventStart, err := parseTime("eventStart")
	if err != nil {
		http.Error(w, "Invalid eventStart: expected RFC3339 datetime", http.StatusBadRequest)
		return
	}
	if eventStart != nil {
		updates.EventStart = *eventStart
	} else {
		updates.EventStart = event.EventStart
	}

	eventEnd, err := parseTime("eventEnd")
	if err != nil {
		http.Error(w, "Invalid eventEnd: expected RFC3339 datetime", http.StatusBadRequest)
		return
	}
	if eventEnd != nil {
		updates.EventEnd = *eventEnd
	} else {
		updates.EventEnd = event.EventEnd
	}
	updates.Food = utils.StringPtr(r.FormValue("food"))
	updates.SignupLink = utils.StringPtr(r.FormValue("signupLink"))
	updates.RegistrationRequired = r.FormValue("registrationRequired") == "true"
	updates.Show = r.FormValue("show") == "true"
	updates.Fee = utils.StringPtr(r.FormValue("fee"))

	if val := r.FormValue("registrationEnd"); val != "" {
		if t, err := time.Parse(time.RFC3339, val); err == nil {
			updates.RegistrationEnd = &t
		}
	}

	fieldsToUpdate := []string{
		"name", "description", "location", "event_start",
		"event_end", "food", "registration_end", "signup_link",
		"registration_required", "show", "fee",
	}

	imageUrl := r.FormValue("imageUrl")
	if imageUrl != "" {
		updates.ImageURL = &imageUrl
		fieldsToUpdate = append(fieldsToUpdate, "image_url")
	} else {
		file, header, err := r.FormFile("file")
		if err == nil {
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
			updates.ImageURL = &fileURL
			fieldsToUpdate = append(fieldsToUpdate, "image_url")
		}
	}

	before := event
	if err := updateWithAudit(r, "events", id, before, &event, func(tx *gorm.DB) error {
		return tx.Model(&event).Select(fieldsToUpdate).Updates(updates).Error
	}, func(tx *gorm.DB) error {
		return tx.First(&event, id).Error
	}, "events"); err != nil {
		http.Error(w, "Update failed", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(updates)
}

// DeleteEvent deletes an event by ID.
// @Summary Delete event
// @Tags events
// @Produce json
// @Param id path int true "Event ID"
// @Success 200 {string} string "Deleted"
// @Failure 404 {string} string "Event not found"
// @Security BearerAuth
// @Router /events/{id} [delete]
func DeleteEvent(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	writeDeleteResponseWithAudit[models.Event](w, r, "events", id, "event not found", nil, "events")
}
