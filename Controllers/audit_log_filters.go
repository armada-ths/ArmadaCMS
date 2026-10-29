package controllers

import (
	"ArmadaCMS/main/utils"

	"gorm.io/gorm"
)

// auditLogQuery applies list filters before counting or paginating audit entries.
func auditLogQuery(query *gorm.DB, filters map[string]string) *gorm.DB {
	query = utils.ApplySearch(query, filters["q"], "actor_username", "actor_name", "resource_type", "resource_id", "request_path", "group_status", "action", "http_method")
	hasParentFilter := false

	for k, v := range filters {
		switch k {
		case "action", "resource_type", "resource_id", "actor_username", "http_method", "group_status":
			query = query.Where(k+" = ?", v)
		case "parent_id":
			hasParentFilter = true
			query = query.Where("parent_id = ?", v)
		}
	}
	if !hasParentFilter {
		query = query.Where("parent_id IS NULL")
		if filters["include_auth"] != "true" {
			query = query.Where("request_path NOT IN ?", []string{
				"/api/v1/login",
				"/api/v1/refreshAccessToken",
				"/api/v1/me/password",
			})
		}
		if filters["include_guest_photos"] != "true" {
			query = query.Where("resource_type <> ?", "eventphotos")
		}
	}

	return query
}
