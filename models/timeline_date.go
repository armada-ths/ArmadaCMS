package models

type TimelineDate struct {
	ID    uint   `gorm:"primaryKey;autoIncrement;column:id;not null" json:"id"`
	Date  string `gorm:"column:date;not null" json:"timeline_date"`
	Title string `gorm:"column:title;not null" json:"timeline_title"`
}
