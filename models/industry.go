package models

type Industry struct {
	ID   uint   `gorm:"primaryKey;autoIncrement;column:id;not null" json:"id"`
	Name string `gorm:"column:name;uniqueIndex;not null" json:"name"`
}
