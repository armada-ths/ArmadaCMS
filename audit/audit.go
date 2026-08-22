package audit

import (
	"ArmadaCMS/main/auth"
	"ArmadaCMS/main/models"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"

	"gorm.io/gorm"
)

const defaultRetentionDays = 7

type parentIDContextKey struct{}

var pruneState struct {
	sync.Mutex
	lastRun time.Time
}

func LogCreate(tx *gorm.DB, r *http.Request, resourceType string, resourceID any, newData any) error {
	return insert(tx, r, "create", resourceType, resourceID, nil, newData)
}

func LogUpdate(tx *gorm.DB, r *http.Request, resourceType string, resourceID any, oldData any, newData any) error {
	return insert(tx, r, "update", resourceType, resourceID, oldData, newData)
}

func LogDelete(tx *gorm.DB, r *http.Request, resourceType string, resourceID any, oldData any) error {
	return insert(tx, r, "delete", resourceType, resourceID, oldData, nil)
}

func WithParent(r *http.Request, parentID uint) *http.Request {
	return r.WithContext(context.WithValue(r.Context(), parentIDContextKey{}, parentID))
}

func StartGroup(tx *gorm.DB, r *http.Request, action string, resourceType string, resourceID any, newData any) (*models.AuditLog, error) {
	return StartGroupWithData(tx, r, action, resourceType, resourceID, nil, newData)
}

func StartGroupWithData(tx *gorm.DB, r *http.Request, action string, resourceType string, resourceID any, oldData any, newData any) (*models.AuditLog, error) {
	entry := buildEntry(tx, r, action, resourceType, resourceID, oldData, newData)
	entry.ParentID = nil
	entry.GroupStatus = "running"

	if err := tx.Create(&entry).Error; err != nil {
		return nil, err
	}
	if err := maybePrune(tx); err != nil {
		return nil, err
	}

	return &entry, nil
}

func FinalizeGroup(tx *gorm.DB, groupID uint, status string, newData any) error {
	var childCount int64
	if err := tx.Model(&models.AuditLog{}).Where("parent_id = ?", groupID).Count(&childCount).Error; err != nil {
		return err
	}

	result := tx.Model(&models.AuditLog{}).
		Where("id = ? AND parent_id IS NULL", groupID).
		Updates(map[string]any{
			"group_status": status,
			"child_count":  childCount,
			"new_data":     marshalJSON(newData),
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func insert(tx *gorm.DB, r *http.Request, action string, resourceType string, resourceID any, oldData any, newData any) error {
	entry := buildEntry(tx, r, action, resourceType, resourceID, oldData, newData)

	if err := tx.Create(&entry).Error; err != nil {
		return err
	}

	return maybePrune(tx)
}

func buildEntry(tx *gorm.DB, r *http.Request, action string, resourceType string, resourceID any, oldData any, newData any) models.AuditLog {
	entry := models.AuditLog{
		Action:       action,
		ResourceType: resourceType,
		ResourceID:   fmt.Sprint(resourceID),
		RequestPath:  r.URL.Path,
		HTTPMethod:   r.Method,
		OldData:      marshalJSON(oldData),
		NewData:      marshalJSON(newData),
	}

	if parentID, ok := r.Context().Value(parentIDContextKey{}).(uint); ok {
		entry.ParentID = &parentID
	}

	if actorUserID, ok := auth.GetUserIDFromContext(r); ok {
		actorID := uint(actorUserID)
		entry.ActorUserID = &actorID

		var user models.User
		if err := tx.Select("id", "username", "name").First(&user, actorUserID).Error; err == nil {
			entry.ActorUsername = user.Username
			entry.ActorName = user.Name
		}
	}

	return entry
}

func maybePrune(tx *gorm.DB) error {
	now := time.Now().UTC()

	pruneState.Lock()
	if !pruneState.lastRun.IsZero() && now.Sub(pruneState.lastRun) < time.Hour {
		pruneState.Unlock()
		return nil
	}
	pruneState.Unlock()

	cutoff := now.AddDate(0, 0, -getRetentionDays())
	result := tx.Where("created_at < ?", cutoff).Delete(&models.AuditLog{})
	if result.Error != nil {
		return result.Error
	}
	log.Printf("audit retention prune completed: cutoff=%s rows_deleted=%d", cutoff.Format(time.RFC3339), result.RowsAffected)

	pruneState.Lock()
	pruneState.lastRun = now
	pruneState.Unlock()

	return nil
}

func getRetentionDays() int {
	value := os.Getenv("AUDIT_LOG_RETENTION_DAYS")
	if value == "" {
		return defaultRetentionDays
	}

	parsed, err := strconv.Atoi(value)
	if err != nil || parsed < 1 {
		return defaultRetentionDays
	}

	return parsed
}

func marshalJSON(data any) string {
	if data == nil {
		return "null"
	}

	encoded, err := json.Marshal(data)
	if err != nil {
		fallback, _ := json.Marshal(map[string]string{
			"serialization_error": err.Error(),
		})
		return string(fallback)
	}

	return string(encoded)
}
