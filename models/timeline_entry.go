package models

type TimelineEntry struct {
	ID        uint   `gorm:"primaryKey;autoIncrement;column:id;not null" json:"id"`
	Title     string `gorm:"column:title;not null" json:"title"`
	Body      string `gorm:"column:body;not null" json:"body"`
	Era       string `gorm:"column:era;not null" json:"era"`
	EraTitle  string `gorm:"column:era_title;not null" json:"eraTitle"`
	SortOrder int    `gorm:"column:sort_order;not null;default:0" json:"sortOrder"`
}
