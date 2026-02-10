package models

import "time"

type FeatureFlag struct {
	ID            uint       `gorm:"primaryKey;autoIncrement;column:id;not null" json:"id"`
	Key           string     `gorm:"column:key;uniqueIndex;not null" json:"key"`
	Description   string     `gorm:"column:description;not null" json:"description"`
	Enabled       bool       `gorm:"column:enabled;not null" json:"enabled"`
	AutoValue     *bool      `gorm:"column:auto_value" json:"autoValue,omitempty"`
	AutoUpdatedAt *time.Time `gorm:"column:auto_updated_at" json:"autoUpdatedAt,omitempty"`
}

func boolPtr(value bool) *bool {
	return &value
}

var DefaultFeatureFlags = []FeatureFlag{
	{
		Key:         "EVENT_PAGE",
		Description: "Access to Event Page",
		Enabled:     true,
		AutoValue:   boolPtr(true),
	},
	{
		Key:         "MAP_PAGE",
		Description: "Access to Map Page",
		Enabled:     false,
		AutoValue:   boolPtr(false),
	},
	{
		Key:         "AT_FAIR_PAGE",
		Description: "Access to At the Fair Page",
		Enabled:     true,
		AutoValue:   boolPtr(true),
	},
	{
		Key:         "EXHIBITOR_SIGNUP",
		Description: "Exhibitor signup links. When enabled, links go to Eventro. When disabled, links go to /exhibitor/signup (coming soon page).",
		Enabled:     true,
		AutoValue:   boolPtr(true),
	},
	{
		Key:         "EXHIBITOR_PACKAGES",
		Description: "Exhibitor packages page content",
		Enabled:     false,
		AutoValue:   boolPtr(false),
	},
	{
		Key:         "EXHIBITOR_EVENTS",
		Description: "Exhibitor events page content",
		Enabled:     false,
		AutoValue:   boolPtr(false),
	},
}
