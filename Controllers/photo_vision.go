package controllers

import (
	"ArmadaCMS/main/db"
	"ArmadaCMS/main/models"
	"ArmadaCMS/main/utils"
	"context"
	"encoding/json"
	"log"
	"time"

	"gorm.io/gorm"
)

// assessUploadedPhoto never lets an unavailable or uncertain classifier publish a photo.
func assessUploadedPhoto(ctx context.Context, photo models.EventPhoto, jpeg []byte) string {
	started := time.Now()
	status := "error"
	var likelihoods map[string]string
	analysisImage, err := utils.PhotoSafeSearchImage(jpeg)
	if err == nil {
		var cancel context.CancelFunc
		ctx, cancel = utils.PhotoVisionTimeout(ctx)
		assessment, assessmentErr := utils.AnalyzePhotoSafeSearch(ctx, analysisImage)
		cancel()
		if assessmentErr == nil {
			status, likelihoods = assessment.Status, assessment.Likelihoods
		}
	}
	if status != "safe" && status != "review" {
		status = "error"
	}
	fields := map[string]any{"ai_review_status": status, "ai_checked_at": time.Now().UTC()}
	if likelihoods != nil {
		encoded, marshalErr := json.Marshal(likelihoods)
		if marshalErr != nil {
			status = "error"
			fields["ai_review_status"] = status
		} else {
			fields["ai_likelihoods"] = gorm.Expr("?::jsonb", string(encoded))
		}
	}
	if status == "safe" {
		fields["status"] = "approved"
		fields["moderated_at"] = time.Now().UTC()
		fields["moderation_source"] = "vision_safe_search"
	}
	result := db.DB.Model(&models.EventPhoto{}).
		Where("id = ? AND event_id = ? AND status = 'pending' AND object_key IS NOT NULL", photo.ID, photo.EventID).
		Where("EXISTS (SELECT 1 FROM photo_events WHERE id = ? AND auto_approve_safe_photos = true AND deletion_requested_at IS NULL)", photo.EventID).
		Updates(fields)
	if result.Error != nil {
		log.Printf("photo_ai result=database_error duration_ms=%d", time.Since(started).Milliseconds())
		return "pending"
	}
	if result.RowsAffected == 0 {
		log.Printf("photo_ai result=skipped duration_ms=%d", time.Since(started).Milliseconds())
		return "pending"
	}
	log.Printf("photo_ai result=%s duration_ms=%d", status, time.Since(started).Milliseconds())
	if status == "safe" {
		return "approved"
	}
	return "pending"
}
