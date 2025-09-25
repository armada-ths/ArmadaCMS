package models

type Employment struct {
	ID   uint   `gorm:"primaryKey;autoIncrement;column:id;not null" json:"id"`
	Name string `gorm:"column:name;not null" json:"name"`
}
