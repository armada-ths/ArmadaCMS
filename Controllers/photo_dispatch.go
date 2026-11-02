package controllers

import (
	"ArmadaCMS/main/db"
	"context"
	"gorm.io/gorm"
	"net/http"
	"time"
)

// photoWorkerHasWork mirrors all work performed by runPhotoWorker, including
// retention and thumbnails. It must not reduce recovery to just the export queue.
func photoWorkerHasWork(ctx context.Context, database *gorm.DB, now time.Time) (bool, error) {
	var work bool
	err := database.WithContext(ctx).Raw(`SELECT
 EXISTS (SELECT 1 FROM photo_exports pe JOIN photo_events e ON e.id=pe.event_id
  WHERE pe.status='queued' AND e.deletion_requested_at IS NULL)
 OR EXISTS (SELECT 1 FROM photo_exports WHERE status='running' AND started_at < ?)
 OR EXISTS (SELECT 1 FROM photo_exports WHERE status='completed' AND expires_at <= ?)
 OR EXISTS (SELECT 1 FROM event_photos WHERE status='rejected' AND object_key IS NOT NULL AND rejected_delete_at <= ?)
 OR EXISTS (SELECT 1 FROM photo_events e WHERE e.deletion_requested_at IS NOT NULL AND e.deletion_completed_at IS NULL
  AND NOT EXISTS (SELECT 1 FROM photo_exports pe WHERE pe.event_id=e.id AND pe.status='running'))
 OR EXISTS (SELECT 1 FROM event_photos p JOIN photo_events e ON e.id=p.event_id
  WHERE p.object_key IS NOT NULL AND p.thumbnail_object_key IS NULL AND e.deletion_requested_at IS NULL
  AND (p.status <> 'rejected' OR p.rejected_delete_at > ?))`, now.Add(-2*time.Hour), now, now, now).Scan(&work).Error
	return work, err
}
func dispatchPhotoWorker(ctx context.Context, database *gorm.DB, now time.Time, launch func(context.Context) error) (bool, error) {
	ready, err := photoWorkerHasWork(ctx, database, now)
	if err != nil || !ready {
		return false, err
	}
	if err = launch(ctx); err != nil {
		return false, err
	}
	return true, nil
}

// DispatchPhotoJobs checks the queue and maintenance tasks before starting a job.
// @Summary Dispatch runnable photo work
// @Tags photos
// @Security SchedulerOIDC
// @Success 202 "Worker start requested"
// @Success 204 "No runnable work"
// @Failure 401 {string} string "Google ID token required"
// @Failure 403 {string} string "Unexpected service account"
// @Failure 503 {string} string "Dispatcher unavailable"
// @Router /internal/photos/dispatch [post]
func DispatchPhotoJobs(w http.ResponseWriter, r *http.Request) {
	started, err := dispatchPhotoWorker(r.Context(), db.DB, time.Now().UTC(), startPhotoWorkerExecution)
	if err != nil {
		http.Error(w, "Photo dispatcher unavailable", http.StatusServiceUnavailable)
		return
	}
	if started {
		w.WriteHeader(http.StatusAccepted)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
