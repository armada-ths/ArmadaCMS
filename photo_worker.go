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
			return nil
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
		  WHERE id = (SELECT id FROM photo_exports WHERE status = 'queued' ORDER BY created_at, id FOR UPDATE SKIP LOCKED LIMIT 1)
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
	if err := db.DB.Where("delete_after <= ?", time.Now()).Find(&events).Error; err != nil {
		return err
	}
	for _, event := range events {
		if err := db.DB.Model(&event).Update("active", false).Error; err != nil {
			return err
		}
		var photos []models.EventPhoto
		if err := db.DB.Where("event_id = ?", event.ID).Find(&photos).Error; err != nil {
			return err
		}
		for _, photo := range photos {
			if photo.ObjectKey != nil {
				if err := utils.DeletePrivatePhoto(ctx, *photo.ObjectKey, false); err != nil {
					return err
				}
			}
			if err := db.DB.Delete(&photo).Error; err != nil {
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
	}
	return nil
}
