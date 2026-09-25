package models

import "time"

type PhotoEvent struct {
	ID                  uint64     `gorm:"primaryKey;autoIncrement;column:id" json:"id"`
	Name                string     `gorm:"column:name" json:"name"`
	Slug                string     `gorm:"column:slug" json:"slug"`
	Description         string     `gorm:"column:description" json:"description"`
	UploadsOpenAt       time.Time  `gorm:"column:uploads_open_at" json:"uploads_open_at"`
	UploadsCloseAt      time.Time  `gorm:"column:uploads_close_at" json:"uploads_close_at"`
	GalleryCloseAt      time.Time  `gorm:"column:gallery_close_at" json:"gallery_close_at"`
	DeletionRequestedAt *time.Time `gorm:"column:deletion_requested_at" json:"deletion_requested_at"`
	DeletionCompletedAt *time.Time `gorm:"column:deletion_completed_at" json:"deletion_completed_at"`
	Active              bool       `gorm:"column:active" json:"active"`
	PrivacyURL          string     `gorm:"column:privacy_url" json:"privacy_url"`
	MaxPhotosPerGuest   int        `gorm:"column:max_photos_per_guest;default:25" json:"max_photos_per_guest"`
	TokenVersion        int        `gorm:"column:token_version;default:1" json:"token_version"`
	CreatedAt           time.Time  `gorm:"column:created_at" json:"created_at"`
	UpdatedAt           time.Time  `gorm:"column:updated_at" json:"updated_at"`
}

type EventPhoto struct {
	ID          uint64     `gorm:"primaryKey;autoIncrement;column:id" json:"id"`
	EventID     uint64     `gorm:"column:event_id" json:"event_id"`
	ObjectKey   *string    `gorm:"column:object_key" json:"-"`
	Status      string     `gorm:"column:status" json:"status"`
	GuestHash   string     `gorm:"column:guest_hash" json:"-"`
	ByteSize    int64      `gorm:"column:byte_size" json:"byte_size"`
	Width       int        `gorm:"column:width" json:"width"`
	Height      int        `gorm:"column:height" json:"height"`
	UploadedAt  time.Time  `gorm:"column:uploaded_at" json:"uploaded_at"`
	ModeratedAt *time.Time `gorm:"column:moderated_at" json:"moderated_at"`
	ModeratedBy *uint      `gorm:"column:moderated_by" json:"moderated_by"`
}

type PhotoExport struct {
	ID          uint64     `gorm:"primaryKey;autoIncrement;column:id" json:"id"`
	EventID     uint64     `gorm:"column:event_id" json:"event_id"`
	Status      string     `gorm:"column:status" json:"status"`
	ObjectKey   *string    `gorm:"column:object_key" json:"-"`
	RequestedBy *uint      `gorm:"column:requested_by" json:"requested_by"`
	CreatedAt   time.Time  `gorm:"column:created_at" json:"created_at"`
	StartedAt   *time.Time `gorm:"column:started_at" json:"started_at"`
	FinishedAt  *time.Time `gorm:"column:finished_at" json:"finished_at"`
	ExpiresAt   *time.Time `gorm:"column:expires_at" json:"expires_at"`
	Error       *string    `gorm:"column:error" json:"error"`
}
