package models

import "strings"

// FairDateConfig is the GORM database model for fair date configuration.
// It stores all dates as flat fields for easy editing in the admin GUI.
type FairDateConfig struct {
	ID           uint    `gorm:"primaryKey;autoIncrement;column:id;not null" json:"id"`
	Description  string  `gorm:"column:description;not null" json:"description"`
	FairDays     string  `gorm:"column:fair_days;not null" json:"fairDays"` // Comma-separated dates, e.g. "2026-11-17,2026-11-18"
	TicketEnd    *string `gorm:"column:ticket_end" json:"ticketEnd"`        // Nullable
	IRStart      string  `gorm:"column:ir_start;not null" json:"irStart"`
	IREnd        string  `gorm:"column:ir_end;not null" json:"irEnd"`
	IRAcceptance string  `gorm:"column:ir_acceptance;not null" json:"irAcceptance"`
	FRStart      string  `gorm:"column:fr_start;not null" json:"frStart"`
	FREnd        string  `gorm:"column:fr_end;not null" json:"frEnd"`
	EventsStart  string  `gorm:"column:events_start;not null" json:"eventsStart"`
	EventsEnd    string  `gorm:"column:events_end" json:"eventsEnd"`
}

// FairDate is the nested JSON structure returned by the public API.
// This preserves backward compatibility with the armada.nu frontend.
type FairDate struct {
	Fair struct {
		Description string   `json:"description"`
		Days        []string `json:"days"`
	} `json:"fair"`
	Ticket struct {
		End *string `json:"end"`
	} `json:"ticket"`
	IR struct {
		Start      string `json:"start"`
		End        string `json:"end"`
		Acceptance string `json:"acceptance"`
	} `json:"ir"`
	FR struct {
		Start string `json:"start"`
		End   string `json:"end"`
	} `json:"fr"`
	Events struct {
		Start string `json:"start"`
		End   string `json:"end"`
	} `json:"events"`
}

// ToFairDate converts a FairDateConfig (flat DB row) to the nested FairDate
// JSON structure expected by the public API.
func (c *FairDateConfig) ToFairDate() FairDate {
	var fd FairDate

	fd.Fair.Description = c.Description
	fd.Fair.Days = splitDays(c.FairDays)

	fd.Ticket.End = c.TicketEnd

	fd.IR.Start = c.IRStart
	fd.IR.End = c.IREnd
	fd.IR.Acceptance = c.IRAcceptance

	fd.FR.Start = c.FRStart
	fd.FR.End = c.FREnd

	fd.Events.Start = c.EventsStart
	fd.Events.End = c.EventsEnd

	return fd
}

// splitDays splits a comma-separated string of dates into a slice.
func splitDays(s string) []string {
	if s == "" {
		return []string{}
	}
	parts := strings.Split(s, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		trimmed := strings.TrimSpace(p)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}
