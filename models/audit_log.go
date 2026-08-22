package models

import "time"

type AuditLog struct {
	ID            uint      `gorm:"primaryKey;autoIncrement;column:id;not null" json:"id"`
	ParentID      *uint     `gorm:"column:parent_id;index" json:"parent_id,omitempty"`
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
	GroupStatus   string    `gorm:"column:group_status;not null;default:'';index" json:"group_status,omitempty"`
	ChildCount    int       `gorm:"column:child_count;not null;default:0" json:"child_count"`
	CreatedAt     time.Time `gorm:"column:created_at;default:now();index" json:"created_at"`
}
