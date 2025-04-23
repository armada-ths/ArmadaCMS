package models

type Team struct {
	ID       uint   `gorm:"primaryKey;autoIncrement;column:id;not null" json:"id"`
	TeamName string `gorm:"column:team_name;not null" json:"team_name"`
}
