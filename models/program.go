package models

type Program struct {
	ID   uint   `gorm:"primaryKey;column:id;not null" json:"id"`
	Code string `gorm:"column:code;not null;UNIQUE" json:"code"`
	Name string `gorm:"column:name;not null" json:"name"`
}
