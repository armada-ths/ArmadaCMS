package models

type Profile struct {
	ID       uint   `gorm:"primaryKey;autoIncrement;column:id;not null" json:"id"`
	Name     string `gorm:"column:name;not null" json:"name"`
	TeamID   int32  `gorm:"column:team_id;not null" json:"team_id"`
	Team     Team   `gorm:"foreignKey:TeamID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL;" json:"team,omitempty"`
	Linkedin string `gorm:"column:linkedin" json:"linkedin"`
	Email    string `gorm:"column:email" json:"email"`
	Photo    string `gorm:"column:photo" json:"photo"`
	Title    string `gorm:"column:title" json:"title"`
}
