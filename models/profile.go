package models

type Profile struct {
	ID         uint    `gorm:"primaryKey;autoIncrement;column:id;not null" json:"id"`
	EventroKey *string `gorm:"column:eventro_key;uniqueIndex" json:"eventro_key,omitempty"`
	Name       string  `gorm:"column:name;not null" json:"name"`
	TeamID     *int32  `gorm:"column:team_id" json:"team_id,omitempty"`
	Team       *Team   `gorm:"foreignKey:TeamID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL;" json:"team,omitempty"`
	Linkedin   string  `gorm:"column:linkedin" json:"linkedin"`
	Email      string  `gorm:"column:email" json:"email"`
	Photo      string  `gorm:"column:photo" json:"photo"`
	Rank       string  `gorm:"column:rank" json:"rank"`
	Title      string  `gorm:"column:title" json:"title"`
}
