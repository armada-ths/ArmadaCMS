package models

// HighlightCard is the GORM database model for a highlight card on the landing page.
// It stores all fields as flat fields for easy editing in the admin GUI.
type HighlightCard struct {
	ID            uint    `gorm:"primaryKey;autoIncrement;column:id;not null" json:"id"`
	Title         string  `gorm:"column:title;not null" json:"title"`
	TitleSv       *string `gorm:"column:title_sv" json:"titleSv,omitempty"`             // Nullable: Swedish translation
	Subtitle      string  `gorm:"column:subtitle;not null" json:"subtitle"`
	SubtitleSv    *string `gorm:"column:subtitle_sv" json:"subtitleSv,omitempty"`       // Nullable: Swedish translation
	Description   string  `gorm:"column:description;not null;type:text" json:"description"`
	DescriptionSv *string `gorm:"column:description_sv;type:text" json:"descriptionSv,omitempty"` // Nullable: Swedish translation
	Brand         *string `gorm:"column:brand;default:'ARMADA'" json:"brand"` // Nullable with default
	LinkText      *string `gorm:"column:link_text" json:"linkText"`           // Nullable
	LinkUrl       *string `gorm:"column:link_url" json:"linkUrl"`             // Nullable
	CtaEventName  *string `gorm:"column:cta_event_name" json:"ctaEventName"`  // Nullable: event name for tracking
}
