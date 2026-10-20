package main

import (
	"ArmadaCMS/main/db"
	"ArmadaCMS/main/models"
	"ArmadaCMS/main/utils"
	"archive/zip"
	"context"
	"fmt"
	"io"
	"log"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// runPhotoWorker processes queued exports and retention. It is safe to run repeatedly.
func runPhotoWorker(ctx context.Context) error {
	if err := cleanupPhotos(ctx); err != nil {
		return err
	}
	for {
		export, err := claimPhotoExport()
		if err != nil {
			return err
		}
		if export == nil {
			return backfillPhotoThumbnails(ctx)
		}
		if err := writePhotoExport(ctx, export); err != nil {
			log.Printf("photo export %d failed: %v", export.ID, err)
			_ = db.DB.Model(export).Updates(map[string]any{"status": "failed", "error": "Export failed; retry from administration", "finished_at": time.Now()}).Error
		}
	}
}

func claimPhotoExport() (*models.PhotoExport, error) {
	var found models.PhotoExport
	err := db.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&models.PhotoExport{}).Where("status = 'running' AND started_at < ?", time.Now().Add(-2*time.Hour)).Updates(map[string]any{"status": "queued", "started_at": nil}).Error; err != nil {
			return err
		}
		return tx.Raw(`UPDATE photo_exports SET status = 'running', started_at = now()
		  WHERE id = (SELECT pe.id FROM photo_exports pe JOIN photo_events e ON e.id = pe.event_id
		    WHERE pe.status = 'queued' AND e.deletion_requested_at IS NULL
		    ORDER BY pe.created_at, pe.id FOR UPDATE OF pe SKIP LOCKED LIMIT 1)
		  RETURNING *`).Scan(&found).Error
	})
	if err != nil || found.ID == 0 {
		return nil, err
	}
	return &found, nil
}

func writePhotoExport(ctx context.Context, export *models.PhotoExport) error {
	started := time.Now()
	var photos []models.EventPhoto
	if err := db.DB.Where("event_id = ? AND status = 'approved' AND object_key IS NOT NULL", export.EventID).Order("id ASC").Find(&photos).Error; err != nil {
		return err
	}
	key := fmt.Sprintf("exports/%d.zip", export.ID)
	reader, writer := io.Pipe()
	writeDone := make(chan error, 1)
	go func() {
		archive := zip.NewWriter(writer)
		for _, photo := range photos {
			if photo.ObjectKey == nil {
				continue
			}
			body, err := utils.ReadPrivatePhoto(ctx, *photo.ObjectKey, false)
			if err != nil {
				_ = writer.CloseWithError(err)
				writeDone <- err
				return
			}
			entry, err := archive.Create(fmt.Sprintf("photo-%d.jpg", photo.ID))
			if err == nil {
				_, err = io.Copy(entry, body)
			}
			_ = body.Close()
			if err != nil {
				_ = writer.CloseWithError(err)
				writeDone <- err
				return
			}
		}
		err := archive.Close()
		if err != nil {
			_ = writer.CloseWithError(err)
		} else {
			_ = writer.Close()
		}
		writeDone <- err
	}()
	uploadErr := utils.UploadPrivatePhoto(ctx, key, reader, "application/zip", true)
	if uploadErr != nil {
		_ = reader.CloseWithError(uploadErr)
	}
	writeErr := <-writeDone
	if uploadErr != nil {
		return uploadErr
	}
	if writeErr != nil {
		_ = utils.DeletePrivatePhoto(ctx, key, true)
		return writeErr
	}
	now := time.Now()
	if err := db.DB.Model(export).Updates(map[string]any{"status": "completed", "object_key": key, "finished_at": now, "expires_at": now.Add(24 * time.Hour), "error": nil}).Error; err != nil {
		_ = utils.DeletePrivatePhoto(ctx, key, true)
		return err
	}
	log.Printf("photo_export result=completed photos=%d duration_ms=%d", len(photos), time.Since(started).Milliseconds())
	return nil
}

func cleanupPhotos(ctx context.Context) error {
	if err := cleanupRejectedPhotos(ctx); err != nil {
		return err
	}
	var exports []models.PhotoExport
	if err := db.DB.Where("status = 'completed' AND expires_at <= ?", time.Now()).Find(&exports).Error; err != nil {
		return err
	}
	for _, export := range exports {
		if export.ObjectKey != nil {
			if err := utils.DeletePrivatePhoto(ctx, *export.ObjectKey, true); err != nil {
				return err
			}
		}
		if err := db.DB.Delete(&export).Error; err != nil {
			return err
		}
	}
	var events []models.PhotoEvent
	if err := db.DB.Where("deletion_requested_at IS NOT NULL AND deletion_completed_at IS NULL").Find(&events).Error; err != nil {
		return err
	}
	for _, event := range events {
		var running int64
		if err := db.DB.Model(&models.PhotoExport{}).Where("event_id = ? AND status = 'running'", event.ID).Count(&running).Error; err != nil {
			return err
		}
		if running > 0 {
			continue
		}
		var photos []models.EventPhoto
		if err := db.DB.Where("event_id = ?", event.ID).Find(&photos).Error; err != nil {
			return err
		}
		for _, photo := range photos {
			if err := db.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
				var current models.EventPhoto
				if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&current, photo.ID).Error; err != nil {
					if err == gorm.ErrRecordNotFound {
						return nil
					}
					return err
				}
				if err := utils.DeletePhotoVariants(ctx, current); err != nil {
					return err
				}
				return tx.Delete(&current).Error
			}); err != nil {
				return err
			}
		}
		var eventExports []models.PhotoExport
		if err := db.DB.Where("event_id = ?", event.ID).Find(&eventExports).Error; err != nil {
			return err
		}
		for _, item := range eventExports {
			if item.ObjectKey != nil {
				if err := utils.DeletePrivatePhoto(ctx, *item.ObjectKey, true); err != nil {
					return err
				}
			}
			if err := db.DB.Delete(&item).Error; err != nil {
				return err
			}
		}
		if err := db.DB.Model(&event).Update("deletion_completed_at", time.Now().UTC()).Error; err != nil {
			return err
		}
	}
	return nil
}

func cleanupRejectedPhotos(ctx context.Context) error {
	for {
		found := false
		err := db.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			var photo models.EventPhoto
			err := tx.Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"}).
				Where("status = 'rejected' AND object_key IS NOT NULL AND rejected_delete_at <= ?", time.Now().UTC()).
				Order("rejected_delete_at, id").Limit(1).Find(&photo).Error
			if err != nil {
				return err
			}
			if photo.ID == 0 {
				return nil
			}
			found = true
			if err := utils.DeletePhotoVariants(ctx, photo); err != nil {
				return err
			}
			return tx.Model(&photo).Updates(map[string]any{"object_key": nil, "thumbnail_object_key": nil}).Error
		})
		if err != nil {
			return err
		}
		if !found {
			return nil
		}
	}
}

// Hosted cleanup is scheduled by Cloud Scheduler. Local development has no scheduler.
func runLocalRejectedPhotoCleanup(ctx context.Context) {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := cleanupRejectedPhotos(ctx); err != nil {
				log.Printf("local rejected photo cleanup failed: %v", err)
			}
		}
	}
}

// Each invocation visits each candidate at most once. Failed photos retry next
// run; row locks serialize workers, moderation and deletion.
func backfillPhotoThumbnails(ctx context.Context) error {
	var after uint64
	for {
		found := false
		var candidateID uint64
		err := db.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			var photo models.EventPhoto
			err := tx.Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"}).
				Where("id > ? AND object_key IS NOT NULL AND thumbnail_object_key IS NULL AND event_id IN (SELECT id FROM photo_events WHERE deletion_requested_at IS NULL) AND (status <> 'rejected' OR rejected_delete_at > ?)", after, time.Now()).
				Order("id").Limit(1).Find(&photo).Error
			if err != nil || photo.ID == 0 {
				return err
			}
			found = true
			candidateID = photo.ID
			body, err := utils.ReadPrivatePhoto(ctx, *photo.ObjectKey, false)
			if err != nil {
				return err
			}
			jpeg, err := io.ReadAll(io.LimitReader(body, utils.MaxPhotoOriginalBytes+1))
			_ = body.Close()
			if err != nil {
				return err
			}
			if len(jpeg) > utils.MaxPhotoOriginalBytes {
				return fmt.Errorf("stored photo exceeds size limit")
			}
			key, err := utils.StorePhotoThumbnail(ctx, *photo.ObjectKey, jpeg)
			if err != nil {
				return err
			}
			if err := tx.Model(&photo).Update("thumbnail_object_key", key).Error; err != nil {
				_ = utils.DeletePrivatePhoto(ctx, key, false)
				return err
			}
			return nil
		})
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if !found {
			return err
		}
		after = candidateID
		if err != nil {
			log.Printf("photo_thumbnail result=failed photo_id=%d: %v", candidateID, err)
		}
	}
}
