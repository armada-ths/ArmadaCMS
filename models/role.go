package models

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
)

// Permissions is a custom type that stores a JSON array of permission strings in Postgres.
type Permissions []string

func (p Permissions) Value() (driver.Value, error) {
	if p == nil {
		return "[]", nil
	}
	b, err := json.Marshal(p)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal permissions: %w", err)
	}
	return string(b), nil
}

func (p *Permissions) Scan(value interface{}) error {
	if value == nil {
		*p = Permissions{}
		return nil
	}
	var bytes []byte
	switch v := value.(type) {
	case string:
		bytes = []byte(v)
	case []byte:
		bytes = v
	default:
		return fmt.Errorf("unsupported type for Permissions: %T", value)
	}
	return json.Unmarshal(bytes, p)
}

// Role defines a named set of permissions assignable to users.
//
// Permission strings follow the format "resource.action":
//
//	resource = the API resource name (e.g. "profiles", "teams", "exhibitors")
//	action   = list | show | create | edit | delete
//
// The special permission "*" grants access to everything (admin).
//
// Examples:
//
//	["*"]                                                   → full admin
//	["profiles.list", "profiles.show", "profiles.create", "profiles.edit"] → profiles CRUD (no delete)
type Role struct {
	ID          uint        `gorm:"primaryKey;autoIncrement;column:id;not null" json:"id"`
	Name        string      `gorm:"column:name;uniqueIndex;not null" json:"name"`
	Permissions Permissions `gorm:"column:permissions;type:text;not null;default:'[]'" json:"permissions"`
}
