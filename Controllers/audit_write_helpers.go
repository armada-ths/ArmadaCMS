package controllers

import (
	"ArmadaCMS/main/audit"
	"ArmadaCMS/main/db"
	"ArmadaCMS/main/models"
	"ArmadaCMS/main/utils"
	"fmt"
	"net/http"
	"reflect"

	"gorm.io/gorm"
)

func startAuditGroup(r *http.Request, action string, resourceType string, resourceID any, data any) (*models.AuditLog, error) {
	var group *models.AuditLog
	err := db.DB.Transaction(func(tx *gorm.DB) error {
		var err error
		group, err = audit.StartGroup(tx, r, action, resourceType, resourceID, data)
		return err
	})
	return group, err
}

func finalizeAuditGroup(groupID uint, status string, data any) error {
	return db.DB.Transaction(func(tx *gorm.DB) error {
		return audit.FinalizeGroup(tx, groupID, status, data)
	})
}

func auditGroupStatus(successful, failed int) string {
	if failed == 0 {
		return "completed"
	}
	if successful == 0 {
		return "failed"
	}
	return "partial"
}

func createWithAudit[T any](r *http.Request, resourceType string, entity *T, persist func(tx *gorm.DB) error, reload func(tx *gorm.DB) error, revalidateTags ...string) error {
	err := db.DB.Transaction(func(tx *gorm.DB) error {
		if err := persist(tx); err != nil {
			return err
		}
		if reload != nil {
			if err := reload(tx); err != nil {
				return err
			}
		}
		return audit.LogCreate(tx, r, resourceType, getResourceID(entity), entity)
	})
	if err == nil {
		for _, tag := range revalidateTags {
			go utils.RevalidateTag(tag)
		}
	}
	return err
}

func updateWithAudit[T any](r *http.Request, resourceType string, resourceID any, before any, after *T, mutate func(tx *gorm.DB) error, reload func(tx *gorm.DB) error, revalidateTags ...string) error {
	err := db.DB.Transaction(func(tx *gorm.DB) error {
		if err := mutate(tx); err != nil {
			return err
		}
		if reload != nil {
			if err := reload(tx); err != nil {
				return err
			}
		}
		return audit.LogUpdate(tx, r, resourceType, resourceID, before, after)
	})
	if err == nil {
		for _, tag := range revalidateTags {
			go utils.RevalidateTag(tag)
		}
	}
	return err
}

// updateWithGroupedAudit records one parent update while retaining detailed
// child audit entries for related mutations. The audit log list hides child
// entries by default and exposes them through the parent's Related count.
func updateWithGroupedAudit[T any](r *http.Request, resourceType string, resourceID any, before any, after *T, mutate func(tx *gorm.DB, childRequest *http.Request) error, reload func(tx *gorm.DB) error, revalidateTags ...string) error {
	err := db.DB.Transaction(func(tx *gorm.DB) error {
		group, err := audit.StartGroupWithData(tx, r, "update", resourceType, resourceID, before, after)
		if err != nil {
			return err
		}

		if err := mutate(tx, audit.WithParent(r, group.ID)); err != nil {
			return err
		}
		if reload != nil {
			if err := reload(tx); err != nil {
				return err
			}
		}
		return audit.FinalizeGroup(tx, group.ID, "completed", after)
	})
	if err == nil {
		for _, tag := range revalidateTags {
			go utils.RevalidateTag(tag)
		}
	}
	return err
}

func writeDeleteResponseWithAudit[T any](w http.ResponseWriter, r *http.Request, resourceType string, id string, notFoundMessage string, buildQuery func(tx *gorm.DB) *gorm.DB, beforeDelete func(tx *gorm.DB, entity *T) error, revalidateTags ...string) {
	var entity T
	query := db.DB
	if buildQuery != nil {
		query = buildQuery(query)
	}

	if err := query.First(&entity, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			http.Error(w, notFoundMessage, http.StatusNotFound)
			return
		}
		http.Error(w, "Delete failed", http.StatusInternalServerError)
		return
	}

	if err := db.DB.Transaction(func(tx *gorm.DB) error {
		if beforeDelete != nil {
			if err := beforeDelete(tx, &entity); err != nil {
				return err
			}
		}
		result := tx.Delete(&entity)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return gorm.ErrRecordNotFound
		}
		return audit.LogDelete(tx, r, resourceType, id, entity)
	}); err != nil {
		if err == gorm.ErrRecordNotFound {
			http.Error(w, notFoundMessage, http.StatusNotFound)
			return
		}
		http.Error(w, "Delete failed", http.StatusInternalServerError)
		return
	}

	for _, tag := range revalidateTags {
		go utils.RevalidateTag(tag)
	}

	w.WriteHeader(http.StatusNoContent)
}

func writeGroupedDeleteResponseWithAudit[T any](w http.ResponseWriter, r *http.Request, resourceType string, id string, notFoundMessage string, buildQuery func(tx *gorm.DB) *gorm.DB, beforeDelete func(tx *gorm.DB, entity *T, childRequest *http.Request) error, summaryData any, revalidateTags ...string) {
	var entity T
	query := db.DB
	if buildQuery != nil {
		query = buildQuery(query)
	}

	if err := query.First(&entity, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			http.Error(w, notFoundMessage, http.StatusNotFound)
			return
		}
		http.Error(w, "Delete failed", http.StatusInternalServerError)
		return
	}

	if err := db.DB.Transaction(func(tx *gorm.DB) error {
		group, err := audit.StartGroupWithData(tx, r, "delete", resourceType, id, entity, summaryData)
		if err != nil {
			return err
		}
		childRequest := audit.WithParent(r, group.ID)

		if beforeDelete != nil {
			if err := beforeDelete(tx, &entity, childRequest); err != nil {
				return err
			}
		}
		result := tx.Delete(&entity)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return gorm.ErrRecordNotFound
		}
		return audit.FinalizeGroup(tx, group.ID, "completed", summaryData)
	}); err != nil {
		if err == gorm.ErrRecordNotFound {
			http.Error(w, notFoundMessage, http.StatusNotFound)
			return
		}
		http.Error(w, "Delete failed", http.StatusInternalServerError)
		return
	}

	for _, tag := range revalidateTags {
		go utils.RevalidateTag(tag)
	}

	w.WriteHeader(http.StatusNoContent)
}

func removeExhibitorAssociationWithAudit(tx *gorm.DB, childRequest *http.Request, exhibitors []models.Exhibitor, association string, value any) error {
	for i := range exhibitors {
		before := exhibitors[i]
		if err := tx.Model(&exhibitors[i]).Association(association).Delete(value); err != nil {
			return err
		}
		if err := tx.Preload("Industries").Preload("Programs").Preload("Employments").First(&exhibitors[i], exhibitors[i].ID).Error; err != nil {
			return err
		}
		if err := audit.LogUpdate(tx, childRequest, "exhibitors", exhibitors[i].ID, before, exhibitors[i]); err != nil {
			return err
		}
	}
	return nil
}

func getResourceID(entity any) string {
	value := reflect.ValueOf(entity)
	if value.Kind() == reflect.Pointer {
		if value.IsNil() {
			return ""
		}
		value = value.Elem()
	}

	if value.Kind() != reflect.Struct {
		return fmt.Sprint(entity)
	}

	field := value.FieldByName("ID")
	if !field.IsValid() {
		return ""
	}

	return fmt.Sprint(field.Interface())
}
