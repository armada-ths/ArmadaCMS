package models

import "time"

type TimelineDate struct {
	ID      uint      `gorm:"primaryKey;autoIncrement;column:id;not null" json:"id"`
	Date    time.Time `gorm:"column:date;not null" json:"date"`
	Title   string    `gorm:"column:title;not null" json:"title"`
	Details string    `gorm:"column:details" json:"details"`
}
