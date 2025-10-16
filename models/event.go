package models

import (
	"time"
)

type Event struct {
	ID                   uint       `gorm:"primaryKey;autoIncrement;column:id;not null" json:"id"`
	EventroID            string     `gorm:"column:eventro_id;uniqueIndex" json:"eventroId"`
	Name                 string     `gorm:"column:name;not null" json:"name"`
	Description          *string    `gorm:"column:description" json:"description,omitempty"`
	Location             string     `gorm:"column:location;not null" json:"location"`
	Food                 *string    `gorm:"column:food" json:"food,omitempty"`
	EventStart           time.Time  `gorm:"column:event_start;not null" json:"eventStart"`
	EventEnd             time.Time  `gorm:"column:event_end;not null" json:"eventEnd"`
	RegistrationEnd      *time.Time `gorm:"column:registration_end" json:"registrationEnd,omitempty"`
	ImageURL             *string    `gorm:"column:image_url" json:"imageUrl,omitempty"`
	Fee                  *string    `gorm:"column:fee" json:"fee,omitempty"`
	RegistrationRequired bool       `gorm:"column:registration_required;not null" json:"registrationRequired"`
	SignupLink           *string    `gorm:"column:signup_link" json:"signupLink,omitempty"`
	EventMaxCapacity     *int       `gorm:"column:event_max_capacity" json:"eventMaxCapacity,omitempty"`
	Show                 bool       `gorm:"column:show;not null;default:true" json:"show"`
}
