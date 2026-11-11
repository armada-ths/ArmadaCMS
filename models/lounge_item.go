package models

type LoungeItem struct {
	ID            uint    `gorm:"primaryKey;autoIncrement;column:id;not null" json:"id"`
	CategoryID    uint    `gorm:"column:category_id;not null" json:"categoryId"`
	Name          string  `gorm:"column:name;not null" json:"name"`
	NameSv        *string `gorm:"column:name_sv" json:"nameSv,omitempty"`
	Description   *string `gorm:"column:description" json:"description,omitempty"`
	DescriptionSv *string `gorm:"column:description_sv" json:"descriptionSv,omitempty"`
	SortOrder     int64   `gorm:"column:sort_order;not null" json:"sortOrder"`
	MaxPerOrder   *int64  `gorm:"column:max_per_order" json:"maxPerOrder"`
	SoldOut       bool    `gorm:"column:sold_out;not null" json:"soldOut"`
}
