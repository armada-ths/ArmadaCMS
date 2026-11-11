package models

import (
	"strings"
	"time"
)

const DefaultLoungeOrderCloseMinutes = 30
const MaxLoungeOrderCloseMinutes = 1440

// FairDateConfig is the GORM database model for fair date configuration.
// It stores all dates as flat fields for easy editing in the admin GUI.
type FairDateConfig struct {
	ID                      uint    `gorm:"primaryKey;autoIncrement;column:id;not null" json:"id"`
	DayOneOpensAt           string  `gorm:"not null;default:10:00" json:"dayOneOpensAt"`
	DayOneClosesAt          string  `gorm:"not null;default:16:00" json:"dayOneClosesAt"`
	DayTwoOpensAt           string  `gorm:"not null;default:10:00" json:"dayTwoOpensAt"`
	DayTwoClosesAt          string  `gorm:"not null;default:15:00" json:"dayTwoClosesAt"`
	LoungeOrderCloseMinutes *int    `gorm:"not null;default:30" json:"loungeOrderCloseMinutes"`
	EventroID               *string `gorm:"column:eventro_id;uniqueIndex:fair_date_configs_eventro_id_key" json:"eventroId"`
	Description             string  `gorm:"column:description;not null" json:"description"`
	DescriptionSv           *string `gorm:"column:description_sv" json:"descriptionSv,omitempty"` // Nullable: Swedish translation
	FairDays                string  `gorm:"column:fair_days;not null" json:"fairDays"`            // Comma-separated dates, e.g. "2026-11-17,2026-11-18"
	IRStart                 string  `gorm:"column:ir_start;not null" json:"irStart"`
	IREnd                   string  `gorm:"column:ir_end;not null" json:"irEnd"`
	IRAcceptance            string  `gorm:"column:ir_acceptance;not null" json:"irAcceptance"`
	FRStart                 string  `gorm:"column:fr_start;not null" json:"frStart"`
	FREnd                   string  `gorm:"column:fr_end;not null" json:"frEnd"`
	EventsStart             string  `gorm:"column:events_start;not null" json:"eventsStart"`
}

func (c FairDateConfig) LoungeClosingLeadMinutes() int {
	if c.LoungeOrderCloseMinutes == nil {
		return DefaultLoungeOrderCloseMinutes
	}
	return *c.LoungeOrderCloseMinutes
}

type FairOpeningHours struct {
	OpensAt  string `json:"opensAt"`
	ClosesAt string `json:"closesAt"`
}

func (hours FairOpeningHours) Valid() bool {
	open, openErr := time.Parse("15:04", hours.OpensAt)
	close, closeErr := time.Parse("15:04", hours.ClosesAt)
	return openErr == nil && closeErr == nil &&
		open.Format("15:04") == hours.OpensAt && close.Format("15:04") == hours.ClosesAt && close.After(open)
}

func (c FairDateConfig) OpeningHours() [2]FairOpeningHours {
	return [2]FairOpeningHours{
		{OpensAt: c.DayOneOpensAt, ClosesAt: c.DayOneClosesAt},
		{OpensAt: c.DayTwoOpensAt, ClosesAt: c.DayTwoClosesAt},
	}
}

// FairDate is the nested JSON structure returned by the public API.
// This preserves backward compatibility with the armada.nu frontend.
type FairDate struct {
	Fair struct {
		Description   string              `json:"description"`
		DescriptionSv *string             `json:"descriptionSv,omitempty"`
		Days          []string            `json:"days"`
		OpeningHours  [2]FairOpeningHours `json:"openingHours"`
	} `json:"fair"`
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
	} `json:"events"`
}

// ToFairDate converts a FairDateConfig (flat DB row) to the nested FairDate
// JSON structure expected by the public API.
func (c *FairDateConfig) ToFairDate() FairDate {
	var fd FairDate

	fd.Fair.Description = c.Description
	fd.Fair.DescriptionSv = c.DescriptionSv
	fd.Fair.Days = splitDays(c.FairDays)
	fd.Fair.OpeningHours = c.OpeningHours()

	fd.IR.Start = c.IRStart
	fd.IR.End = c.IREnd
	fd.IR.Acceptance = c.IRAcceptance

	fd.FR.Start = c.FRStart
	fd.FR.End = c.FREnd

	fd.Events.Start = c.EventsStart

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
