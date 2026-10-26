package models

type TimelineEra struct {
	ID        uint    `gorm:"primaryKey;autoIncrement;column:id;not null" json:"id"`
	Title     string  `gorm:"column:title;not null" json:"title"`
	TitleSv   *string `gorm:"column:title_sv" json:"titleSv,omitempty"`
	SortOrder int     `gorm:"column:sort_order;not null" json:"sortOrder"`
}
