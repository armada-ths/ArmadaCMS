package controllers

import (
	"ArmadaCMS/main/auth"
	"ArmadaCMS/main/db"
	"ArmadaCMS/main/internal/translations"
	"ArmadaCMS/main/utils"
	"context"
	"encoding/json"
	"errors"
	"github.com/gorilla/mux"
	"gorm.io/gorm"
	"io"
	"net/http"
	"strconv"
	"time"
)

type TranslationResourceSummary struct {
	ID             string `json:"id"`
	Label          string `json:"label"`
	AIAllowed      bool   `json:"aiAllowed"`
	CanEdit        bool   `json:"canEdit"`
	MissingRecords int    `json:"missingRecords"`
}
type TranslationOverviewResponse struct {
	Resources       []TranslationResourceSummary `json:"resources"`
	AIEnabled       bool                         `json:"aiEnabled"`
	DailyBudget     int64                        `json:"dailyBudget"`
	Usage           translations.Usage           `json:"usage"`
	PromptVersion   string                       `json:"promptVersion"`
	GlossaryVersion string                       `json:"glossaryVersion"`
}
type TranslationGenerationRequest struct {
	IDs        []uint64 `json:"ids"`
	Regenerate bool     `json:"regenerate"`
}

func translationPermission(r *http.Request, resource, action string) bool {
	return auth.HasPermission(auth.GetPermissionsFromContext(r), resource+"."+action)
}
func translationResource(w http.ResponseWriter, r *http.Request, write bool) (translations.Resource, bool) {
	resource, err := translations.Find(mux.Vars(r)["resource"])
	if err != nil {
		http.Error(w, "Unknown resource", http.StatusNotFound)
		return resource, false
	}
	if !translationPermission(r, resource.ID, "view") || (write && !translationPermission(r, resource.ID, "edit")) {
		http.Error(w, "Insufficient resource permissions", http.StatusForbidden)
		return resource, false
	}
	return resource, true
}
func translationError(w http.ResponseWriter, err error) {
	status := http.StatusBadRequest
	message := err.Error()
	switch {
	case errors.Is(err, translations.ErrConflict):
		status = http.StatusConflict
	case errors.Is(err, translations.ErrDisabled):
		status = http.StatusServiceUnavailable
	case errors.Is(err, gorm.ErrRecordNotFound):
		status = http.StatusNotFound
	default:
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			status = http.StatusServiceUnavailable
			message = "Translation service unavailable"
		}
	}
	// DB errors must not expose SQL or source/draft contents in HTTP responses.
	var pg interface{ SQLState() string }
	if errors.As(err, &pg) {
		status = http.StatusServiceUnavailable
		message = "Translation storage unavailable"
	}
	http.Error(w, message, status)
}
func translationID(w http.ResponseWriter, r *http.Request) (uint64, bool) {
	id, err := strconv.ParseUint(mux.Vars(r)["id"], 10, 64)
	if err != nil || id == 0 {
		http.Error(w, "Invalid record ID", http.StatusBadRequest)
		return 0, false
	}
	return id, true
}
func translationBody(w http.ResponseWriter, r *http.Request, value any) bool {
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	decoder.DisallowUnknownFields()
	if decoder.Decode(value) != nil || decoder.Decode(&struct{}{}) != io.EOF {
		http.Error(w, "Invalid translation request", http.StatusBadRequest)
		return false
	}
	return true
}

// GetTranslationsOverview lists only resources the editor can read.
// @Summary Translation review overview
// @Tags translations
// @Security BearerAuth
// @Produce json
// @Success 200 {object} TranslationOverviewResponse
// @Router /translations [get]
func GetTranslationsOverview(w http.ResponseWriter, r *http.Request) {
	s := translations.New(db.DB)
	response := TranslationOverviewResponse{Resources: []TranslationResourceSummary{}, AIEnabled: s.Config.Enabled, DailyBudget: s.Config.DailyBudget, PromptVersion: translations.PromptVersion, GlossaryVersion: translations.GlossaryVersion}
	for _, resource := range translations.Resources {
		if !translationPermission(r, resource.ID, "view") {
			continue
		}
		page, err := s.List(r.Context(), resource, "", "missing", 0, 1)
		if err != nil {
			translationError(w, err)
			return
		}
		response.Resources = append(response.Resources, TranslationResourceSummary{resource.ID, resource.Label, resource.AI, translationPermission(r, resource.ID, "edit"), page.Total})
	}
	if len(response.Resources) == 0 {
		http.Error(w, "No readable translation resources", http.StatusForbidden)
		return
	}
	zone, _ := time.LoadLocation("Europe/Stockholm")
	response.Usage.Day = time.Now().In(zone).Format("2006-01-02")
	if err := db.DB.WithContext(r.Context()).Where("day = ?", response.Usage.Day).Limit(1).Find(&response.Usage).Error; err != nil {
		translationError(w, err)
		return
	}
	writeJSONResponse(w, http.StatusOK, response)
}

// GetTranslationRecords returns paired original/published/suggested fields.
// @Summary List translation review records
// @Tags translations
// @Security BearerAuth
// @Produce json
// @Param resource path string true "Resource identifier"
// @Param page query int false "Zero-based page"
// @Param size query int false "Page size (maximum 50)"
// @Param q query string false "Search"
// @Param status query string false "Review status"
// @Success 200 {object} translations.Page
// @Router /translations/{resource} [get]
func GetTranslationRecords(w http.ResponseWriter, r *http.Request) {
	resource, ok := translationResource(w, r, false)
	if !ok {
		return
	}
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	size, _ := strconv.Atoi(r.URL.Query().Get("size"))
	page = max(page, 0)
	if size < 1 || size > 50 {
		size = 25
	}
	status := r.URL.Query().Get("status")
	if status == "" {
		status = "missing"
	}
	result, err := translations.New(db.DB).List(r.Context(), resource, r.URL.Query().Get("q"), status, page, size)
	if err != nil {
		translationError(w, err)
		return
	}
	writeJSONResponse(w, http.StatusOK, result)
}

// GenerateTranslations queues at most 50 explicitly selected records.
// @Summary Generate Swedish translation suggestions
// @Tags translations
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param resource path string true "Resource identifier"
// @Param request body TranslationGenerationRequest true "Selected records"
// @Success 202 {array} translations.Job
// @Router /translations/{resource}/generate [post]
func GenerateTranslations(w http.ResponseWriter, r *http.Request) {
	resource, ok := translationResource(w, r, true)
	if !ok {
		return
	}
	var input TranslationGenerationRequest
	if !translationBody(w, r, &input) {
		return
	}
	user, ok := auth.GetUserIDFromContext(r)
	if !ok {
		http.Error(w, "Missing editor identity", http.StatusUnauthorized)
		return
	}
	service := translations.New(db.DB)
	jobs, err := service.Generate(r.Context(), resource, input.IDs, user, input.Regenerate)
	if err != nil {
		translationError(w, err)
		return
	}
	if len(jobs) > 0 {
		_, _ = service.Dispatch(r.Context(), utils.LaunchTranslationWorker)
	}
	writeJSONResponse(w, http.StatusAccepted, jobs)
}

// ReviewTranslations saves drafts or approves/rejects selected fields atomically.
// @Summary Review Swedish translation fields
// @Tags translations
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param resource path string true "Resource identifier"
// @Param id path int true "Record ID"
// @Param request body translations.Review true "Versioned field review"
// @Success 200 {object} translations.Record
// @Failure 409 {string} string "Source, published value or draft changed"
// @Router /translations/{resource}/{id}/review [post]
func ReviewTranslations(w http.ResponseWriter, r *http.Request) {
	resource, ok := translationResource(w, r, true)
	if !ok {
		return
	}
	id, ok := translationID(w, r)
	if !ok {
		return
	}
	user, ok := auth.GetUserIDFromContext(r)
	if !ok {
		http.Error(w, "Missing editor identity", http.StatusUnauthorized)
		return
	}
	var input translations.Review
	if !translationBody(w, r, &input) {
		return
	}
	service := translations.New(db.DB)
	err := service.Review(r.Context(), resource, id, input, user, func(tx *gorm.DB, before, after any, updates map[string]any) error {
		return updateWithAuditInTransaction(tx, r, resource.ID, id, before, after, func(tx *gorm.DB) error { return tx.Model(before).Updates(updates).Error }, func(tx *gorm.DB) error { return tx.First(after, id).Error })
	})
	if err != nil {
		translationError(w, err)
		return
	}
	if input.Action == "approve" {
		for _, tag := range resource.Tags {
			go utils.RevalidateTag(tag)
		}
	}
	record, err := service.Record(r.Context(), resource, id)
	if err != nil {
		translationError(w, err)
		return
	}
	writeJSONResponse(w, http.StatusOK, record)
}

// CancelTranslationJobs cancels only queued/budget-waiting work for a record.
// @Summary Cancel pending translation generation
// @Tags translations
// @Security BearerAuth
// @Param resource path string true "Resource identifier"
// @Param id path int true "Record ID"
// @Success 204
// @Router /translations/{resource}/{id}/jobs [delete]
func CancelTranslationJobs(w http.ResponseWriter, r *http.Request) {
	resource, ok := translationResource(w, r, true)
	if !ok {
		return
	}
	id, ok := translationID(w, r)
	if !ok {
		return
	}
	if err := translations.New(db.DB).Cancel(r.Context(), resource, id); err != nil {
		translationError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
