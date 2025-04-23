package models

import "time"

type Blogpost struct {
	ID        uint      `gorm:"primaryKey;autoIncrement;column:id;not null" json:"id"`
	UserID    int64     `gorm:"column:user_id;not null" json:"user_id"`
	Text      string    `gorm:"column:text" json:"text"`
	Title     string    `gorm:"column:title" json:"title"`
	Author    string    `gorm:"column:author" json:"author"`
	CreatedAt time.Time `gorm:"column:created_at;default:CURRENT_TIMESTAMP" json:"created_at"`
}
