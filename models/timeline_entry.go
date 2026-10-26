package models

type TimelineEntry struct {
	ID         uint        `gorm:"primaryKey;autoIncrement;column:id;not null" json:"id"`
	Title      string      `gorm:"column:title;not null" json:"title"`
	TitleSv    *string     `gorm:"column:title_sv" json:"titleSv,omitempty"`
	Body       string      `gorm:"column:body;not null" json:"body"`
	BodySv     *string     `gorm:"column:body_sv" json:"bodySv,omitempty"`
	EraID      uint        `gorm:"column:era_id;not null" json:"eraId"`
	EraRecord  TimelineEra `gorm:"foreignKey:EraID" json:"-"`
	SortOrder  int         `gorm:"column:sort_order;not null" json:"sortOrder"`
	EraTitleSv *string     `gorm:"-" json:"eraTitleSv,omitempty"`
	EraTitle   string      `gorm:"-" json:"eraTitle"`
	ImageUrl   *string     `gorm:"column:image_url" json:"imageUrl"`
}

func (entry *TimelineEntry) SetEraPresentation() {
	entry.EraTitle = entry.EraRecord.Title
	entry.EraTitleSv = entry.EraRecord.TitleSv
}
