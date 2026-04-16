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
		Description: "Show the student events page",
		Enabled:     true,
	},
	{
		Key:         "MAP_PAGE",
		Description: "Show the fair map page",
		Enabled:     false,
	},
	{
		Key:         "AT_FAIR_PAGE",
		Description: "Show the at-the-fair student page",
		Enabled:     true,
	},
	{
		Key:         "EXHIBITOR_PACKAGES",
		Description: "Show the exhibitor packages page",
		Enabled:     false,
	},
	{
		Key:         "EXHIBITOR_EVENTS",
		Description: "Show the exhibitor events page",
		Enabled:     false,
	},
	{
		Key:         "EXHIBITOR_PAGE",
		Description: "Show the student exhibitors/companies page",
		Enabled:     true,
	},
}
