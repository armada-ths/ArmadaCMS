package models

type TimelineEntry struct {
	ID        uint        `gorm:"primaryKey;autoIncrement;column:id;not null" json:"id"`
	Title     string      `gorm:"column:title;not null" json:"title"`
	Body      string      `gorm:"column:body;not null" json:"body"`
	EraID     uint        `gorm:"column:era_id;not null" json:"eraId"`
	EraRecord TimelineEra `gorm:"foreignKey:EraID" json:"-"`
	SortOrder int         `gorm:"column:sort_order;not null" json:"sortOrder"`
	EraTitle  string      `gorm:"-" json:"eraTitle"`
}

func (entry *TimelineEntry) SetEraPresentation() {
	entry.EraTitle = entry.EraRecord.Title
}
