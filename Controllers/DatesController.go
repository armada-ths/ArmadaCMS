package controllers

import (
	"ArmadaCMS/main/db"
	"ArmadaCMS/main/models"
	"encoding/json"
	"net/http"
)

// GetFairDates returns the fair dates in the nested JSON format expected by
// the public armada.nu frontend. It reads from the database (latest record)
// and falls back to hardcoded defaults if no record exists yet.
func GetFairDates(w http.ResponseWriter, r *http.Request) {
	var config models.FairDateConfig
	result := db.DB.Order("id desc").First(&config)

	var fairDate models.FairDate
	if result.Error != nil {
		// No record in DB yet — return hardcoded defaults for backward compatibility
		fairDate = defaultFairDate()
	} else {
		fairDate = config.ToFairDate()
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(fairDate)
}

// defaultFairDate returns the previously hardcoded values as a fallback.
func defaultFairDate() models.FairDate {
	var fd models.FairDate
	fd.Fair.Description = "THS Armada 2026"
	fd.Fair.Days = []string{"2026-11-17", "2026-11-18"}
	fd.Ticket.End = nil
	fd.IR.Start = "2026-03-01"
	fd.IR.End = "2026-06-01"
	fd.IR.Acceptance = "2026-06-01"
	fd.FR.Start = "2026-08-01"
	fd.FR.End = "2026-09-01"
	fd.Events.Start = "2026-11-02"
	fd.Events.End = ""
	return fd
}
