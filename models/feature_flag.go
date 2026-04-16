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
		Enabled:     true,
	},
	{
		Key:         "AT_FAIR_PAGE",
		Description: "Show the at-the-fair student page",
		Enabled:     true,
	},
	{
		Key:         "EXHIBITOR_PACKAGES",
		Description: "Show the exhibitor packages page",
		Enabled:     true,
	},
	{
		Key:         "EXHIBITOR_EVENTS",
		Description: "Show the exhibitor events page",
		Enabled:     true,
	},
	{
		Key:         "EXHIBITOR_PAGE",
		Description: "Show the student exhibitors/companies page",
		Enabled:     true,
	},
	{
		Key:         "STUDENT_RECRUITMENT_PAGE",
		Description: "Show the student recruitment page",
		Enabled:     true,
	},
	{
		Key:         "EXHIBITOR_MAIN_PAGE",
		Description: "Show the exhibitor main/why armada page",
		Enabled:     true,
	},
	{
		Key:         "EXHIBITOR_TIMELINE_PAGE",
		Description: "Show the exhibitor timeline page",
		Enabled:     true,
	},
	{
		Key:         "EXHIBITOR_SIGNUP_PAGE",
		Description: "Show the exhibitor signup/registration page (only controls the topnav link, the page itself is controlled by the exhibitor timeline).",
		Enabled:     true,
	},
	{
		Key:         "ABOUT_PAGE",
		Description: "Show the about armada page",
		Enabled:     true,
	},
	{
		Key:         "ABOUT_TEAM_PAGE",
		Description: "Show the about team page",
		Enabled:     true,
	},
	{
		Key:         "ARMADA_BLOG_PAGE",
		Description: "Show the armada blog page",
		Enabled:     false,
	},
}
