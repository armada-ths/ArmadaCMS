package models

import "time"

type Blogpost struct {
	ID              uint      `gorm:"primaryKey;autoIncrement;column:id;not null" json:"id"`
	UserID          int64     `gorm:"column:user_id;not null" json:"userId"`
	Text            string    `gorm:"column:text;type:text" json:"text"`
	TextSv          *string   `gorm:"column:text_sv;type:text" json:"textSv,omitempty"` // Nullable: Swedish translation
	Title           string    `gorm:"column:title" json:"title"`
	TitleSv         *string   `gorm:"column:title_sv" json:"titleSv,omitempty"` // Nullable: Swedish translation
	Author          string    `gorm:"column:author" json:"author"`
	Published       bool      `gorm:"column:published;not null" json:"published"`
	ImageURL        *string   `gorm:"column:image_url" json:"imageUrl,omitempty"`
	ImageURLs       []string  `gorm:"column:image_urls;type:jsonb;serializer:json;not null;default:'[]'" json:"imageUrls"`
	ShowCoverInPost bool      `gorm:"column:show_cover_in_post" json:"showCoverInPost"`
	CreatedAt       time.Time `gorm:"column:created_at;default:CURRENT_TIMESTAMP" json:"createdAt"`
}
