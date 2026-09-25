package controllers

import (
	"ArmadaCMS/main/auth"
	"ArmadaCMS/main/db"
	"ArmadaCMS/main/models"
	"ArmadaCMS/main/utils"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func StartPhotoExport(w http.ResponseWriter, r *http.Request) {
	var event models.PhotoEvent
	if db.DB.First(&event, mux.Vars(r)["id"]).Error != nil {
		http.NotFound(w, r)
		return
	}
	if event.DeletionRequestedAt != nil {
		http.Error(w, "Event deletion requested", http.StatusConflict)
		return
	}
	userID, _ := auth.GetUserIDFromContext(r)
	admin := uint(userID)
	job := models.PhotoExport{EventID: event.ID, Status: "queued", RequestedBy: &admin, CreatedAt: time.Now()}
	if err := createWithAudit(r, "photoexports", &job, func(tx *gorm.DB) error {
		var locked models.PhotoEvent
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&locked, event.ID).Error; err != nil {
			return err
		}
		if locked.DeletionRequestedAt != nil {
			return fmt.Errorf("event deletion requested")
		}
		return tx.Create(&job).Error
	}, nil); err != nil {
		http.Error(w, "Could not queue export", http.StatusInternalServerError)
		return
	}
	log.Print("photo_export result=queued")
	if err := launchPhotoWorker(r.Context()); err != nil {
		// The Scheduler launches the same job, so a failed direct start leaves a recoverable queue item.
		w.Header().Set("X-Photo-Export-Launch", "scheduled-retry")
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	_ = json.NewEncoder(w).Encode(job)
}

func GetPhotoExport(w http.ResponseWriter, r *http.Request) {
	var job models.PhotoExport
	if db.DB.First(&job, mux.Vars(r)["id"]).Error != nil {
		http.NotFound(w, r)
		return
	}
	response := map[string]any{"id": job.ID, "event_id": job.EventID, "status": job.Status, "created_at": job.CreatedAt, "expires_at": job.ExpiresAt, "error": job.Error}
	if job.Status == "completed" && job.ObjectKey != nil && job.ExpiresAt != nil && time.Now().Before(*job.ExpiresAt) {
		duration := time.Until(*job.ExpiresAt)
		link, err := utils.SignPrivatePhoto(r.Context(), *job.ObjectKey, true, duration)
		if err != nil {
			http.Error(w, "Export unavailable", http.StatusServiceUnavailable)
			return
		}
		response["download_url"] = link
	}
	w.Header().Set("Cache-Control", "no-store")
	photoAdminJSON(w, response)
}

func ListPhotoExports(w http.ResponseWriter, r *http.Request) {
	var jobs []models.PhotoExport
	query := db.DB.Order("created_at DESC").Limit(100)
	if eventID := r.URL.Query().Get("event_id"); eventID != "" {
		query = query.Where("event_id = ?", eventID)
	}
	if err := query.Find(&jobs).Error; err != nil {
		http.Error(w, "Export list unavailable", http.StatusServiceUnavailable)
		return
	}
	photoAdminJSON(w, jobs)
}

func launchPhotoWorker(ctx context.Context) error {
	name := strings.TrimSpace(os.Getenv("PHOTO_WORKER_JOB_NAME"))
	if name == "" {
		return fmt.Errorf("PHOTO_WORKER_JOB_NAME missing")
	}
	access, err := utils.GoogleAccessToken(ctx)
	if err != nil {
		return err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://run.googleapis.com/v2/"+name+":run", bytes.NewReader([]byte("{}")))
	if err != nil {
		return err
	}
	request.Header.Set("Authorization", "Bearer "+access)
	request.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: 10 * time.Second}
	response, err := client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return fmt.Errorf("worker launch: status %d", response.StatusCode)
	}
	return nil
}
