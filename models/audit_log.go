package models

import "time"

type AuditLog struct {
	ID            uint      `gorm:"primaryKey;autoIncrement;column:id;not null" json:"id"`
	ActorUserID   *uint     `gorm:"column:actor_user_id;index" json:"actor_user_id,omitempty"`
	ActorUsername string    `gorm:"column:actor_username" json:"actor_username,omitempty"`
	ActorName     string    `gorm:"column:actor_name" json:"actor_name,omitempty"`
	Action        string    `gorm:"column:action;not null;index" json:"action"`
	ResourceType  string    `gorm:"column:resource_type;not null;index" json:"resource_type"`
	ResourceID    string    `gorm:"column:resource_id;not null;index" json:"resource_id"`
	RequestPath   string    `gorm:"column:request_path;not null" json:"request_path"`
	HTTPMethod    string    `gorm:"column:http_method;not null" json:"http_method"`
	OldData       string    `gorm:"column:old_data;type:jsonb;not null;default:'null'" json:"old_data"`
	NewData       string    `gorm:"column:new_data;type:jsonb;not null;default:'null'" json:"new_data"`
	CreatedAt     time.Time `gorm:"column:created_at;default:now();index" json:"created_at"`
}
