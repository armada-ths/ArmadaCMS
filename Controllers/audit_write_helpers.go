package controllers

import (
	"ArmadaCMS/main/audit"
	"ArmadaCMS/main/db"
	"ArmadaCMS/main/utils"
	"fmt"
	"net/http"
	"reflect"

	"gorm.io/gorm"
)

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

func writeDeleteResponseWithAudit[T any](w http.ResponseWriter, r *http.Request, resourceType string, id string, notFoundMessage string, buildQuery func(tx *gorm.DB) *gorm.DB, revalidateTags ...string) {
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
