package models

type FeatureFlag struct {
	ID          uint   `gorm:"primaryKey;autoIncrement;column:id;not null" json:"id"`
	Key         string `gorm:"column:key;uniqueIndex;not null" json:"key"`
	Description string `gorm:"column:description;not null" json:"description"`
	Enabled     bool   `gorm:"column:enabled;not null" json:"enabled"`
}

var DefaultFeatureFlags = []FeatureFlag{
	{
		Key:         "EVENT_PAGE",
		Description: "Access to Event Page",
		Enabled:     true,
	},
	{
		Key:         "MAP_PAGE",
		Description: "Access to Map Page",
		Enabled:     false,
	},
	{
		Key:         "AT_FAIR_PAGE",
		Description: "Access to At the Fair Page",
		Enabled:     true,
	},
	{
		Key:         "EXHIBITOR_PACKAGES",
		Description: "Exhibitor packages page content",
		Enabled:     false,
	},
	{
		Key:         "EXHIBITOR_EVENTS",
		Description: "Exhibitor events page content",
		Enabled:     false,
	},
}
