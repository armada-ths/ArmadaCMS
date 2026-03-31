package audit

import (
	"ArmadaCMS/main/auth"
	"ArmadaCMS/main/models"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"

	"gorm.io/gorm"
)

const defaultRetentionDays = 7

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

func insert(tx *gorm.DB, r *http.Request, action string, resourceType string, resourceID any, oldData any, newData any) error {
	entry := models.AuditLog{
		Action:       action,
		ResourceType: resourceType,
		ResourceID:   fmt.Sprint(resourceID),
		RequestPath:  r.URL.Path,
		HTTPMethod:   r.Method,
		OldData:      marshalJSON(oldData),
		NewData:      marshalJSON(newData),
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

	if err := tx.Create(&entry).Error; err != nil {
		return err
	}

	return maybePrune(tx)
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
	if err := tx.Where("created_at < ?", cutoff).Delete(&models.AuditLog{}).Error; err != nil {
		return err
	}

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
