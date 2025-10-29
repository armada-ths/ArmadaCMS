package models

type Program struct {
	ID   uint   `gorm:"primaryKey;column:id;not null" json:"id"`
	Name string `gorm:"column:name;uniqueIndex;not null" json:"name"`
}
