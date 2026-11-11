package models

import "time"

// EventroSyncBlock records the source identity of a manually deleted CMS record.
type EventroSyncBlock struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	Resource   string    `gorm:"not null;uniqueIndex:idx_eventro_sync_block_identity" json:"resource"`
	EventroKey string    `gorm:"not null;uniqueIndex:idx_eventro_sync_block_identity" json:"eventroKey"`
	Label      string    `gorm:"not null" json:"label"`
	DeletedAt  time.Time `gorm:"not null;default:now()" json:"deletedAt"`
}
