package models

import "time"

type Tokens struct {
	RefreshToken string `json:"refreshToken"`
	AccessToken  string `json:"accessToken"`
}
type RefreshToken struct {
	ID           uint      `gorm:"primaryKey;autoIncrement;column:id;not null" json:"id"`
	RefreshToken string    `gorm:"column:refresh_token;type:text;not null" json:"-"`
	ValidFrom    time.Time `gorm:"column:valid_from;not null"`
	ValidTo      time.Time `gorm:"column:valid_to;not null"`
	UserID       uint      `gorm:"column:user_id;not null"`
	User         User      `gorm:"foreignKey:UserID"`
	Enabled      bool      `gorm:"column:enabled;default:true"`
}
