package models

import "time"

type User struct {
	ID        uint      `gorm:"primaryKey;autoIncrement;column:id;not null" json:"id"`
	Username  string    `gorm:"column:username;not null" json:"username"`
	Password  string    `gorm:"column:password;not null" json:"-"`
	Name      string    `gorm:"column:name;not null" json:"name"`
	Avatar    string    `gorm:"column:avatar" json:"avatar"`
	CreatedAt time.Time `gorm:"column:created_at;default:now()" json:"created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at;default:now()" json:"updated_at"`
}
