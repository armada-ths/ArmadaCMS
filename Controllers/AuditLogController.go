package controllers

import (
	"ArmadaCMS/main/db"
	"ArmadaCMS/main/models"
	"ArmadaCMS/main/utils"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
)

// GetAuditLogs returns a paginated list of audit log entries.
// @Summary List audit logs
// @Tags audit-logs
// @Produce json
// @Param range query string false "Pagination range, e.g. [0,24]"
// @Param sort query string false "Sort, e.g. [\"created_at\",\"DESC\"]"
// @Param filter query string false "Filter. Supported keys: action, resource_type, resource_id, actor_username, http_method, q (full-text)"
// @Success 200 {array} models.AuditLog
// @Header 200 {string} Content-Range "auditlogs 0-24/1000"
// @Security BearerAuth
// @Router /auditlogs [get]
func GetAuditLogs(w http.ResponseWriter, r *http.Request) {
	params, _ := utils.ParseListParams(r.URL.Query())

	var logs []models.AuditLog
	query := db.DB.Model(&models.AuditLog{})

	for k, v := range params.Filter {
		switch k {
		case "action", "resource_type", "resource_id", "actor_username", "http_method":
			query = query.Where(k+" = ?", v)
		case "q":
			like := "%" + v + "%"
			query = query.Where(
				"actor_username ILIKE ? OR actor_name ILIKE ? OR resource_type ILIKE ? OR resource_id ILIKE ? OR request_path ILIKE ?",
				like, like, like, like, like,
			)
		}
	}

	if len(params.Sort) == 2 {
		query = query.Order(params.Sort[0] + " " + params.Sort[1])
	} else {
		query = query.Order("created_at DESC")
	}

	start, end := params.Range[0], params.Range[1]
	limit := end - start + 1

	var total int64
	query.Count(&total)

	query = query.Offset(start).Limit(limit)
	query.Find(&logs)

	w.Header().Set("Access-Control-Expose-Headers", "Content-Range")
	w.Header().Set("Content-Range", fmt.Sprintf("auditlogs %d-%d/%d", start, end, total))
	writeJSONResponse(w, http.StatusOK, logs)
}

// GetAuditLogByID returns a single audit log entry by ID.
// @Summary Get audit log by ID
// @Tags audit-logs
// @Produce json
// @Param id path int true "AuditLog ID"
// @Success 200 {object} models.AuditLog
// @Failure 404 {string} string "Audit log entry not found"
// @Security BearerAuth
// @Router /auditlogs/{id} [get]
func GetAuditLogByID(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	var entry models.AuditLog
	if err := db.DB.First(&entry, id).Error; err != nil {
		http.Error(w, "Audit log entry not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(entry)
}
