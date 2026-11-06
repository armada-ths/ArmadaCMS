package models

import "encoding/json"

// EventroSyncState is private import metadata, never part of public exhibitors.
type EventroSyncState struct {
	Resource         string                     `gorm:"primaryKey" json:"resource"`
	RecordID         uint                       `gorm:"primaryKey;autoIncrement:false" json:"recordId"`
	FairID           string                     `json:"fairId"`
	Source           map[string]json.RawMessage `gorm:"serializer:json;type:jsonb" json:"source"`
	Overrides        map[string]bool            `gorm:"serializer:json;type:jsonb" json:"overrides"`
	Conflicts        map[string]bool            `gorm:"serializer:json;type:jsonb" json:"conflicts"`
	UnmappedProducts []EventroOrderedProduct    `gorm:"serializer:json;type:jsonb" json:"unmappedProducts"`
}

type EventroOrderedProduct struct {
	ProductID   string `json:"productId"`
	Name        string `json:"name"`
	DisplayName string `json:"displayName"`
}
