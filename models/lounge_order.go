package models

import "time"

type LoungeCategory struct {
	ID        uint    `gorm:"primaryKey" json:"id"`
	Name      string  `json:"name"`
	NameSv    *string `json:"nameSv,omitempty"`
	SortOrder int64   `json:"sortOrder"`
}

type LoungeOrder struct {
	ID                     uint              `gorm:"primaryKey" json:"id"`
	CreatedAt              time.Time         `json:"createdAt"`
	CompanyID              uint              `json:"companyId"`
	CompanyName            string            `json:"companyName"`
	IsTest                 bool              `json:"isTest"`
	Status                 string            `json:"status"`
	IdempotencyKey         string            `json:"-"`
	Fingerprint            string            `json:"-"`
	DeliveryStatus         string            `json:"deliveryStatus"`
	DeliveryRetryRequested bool              `json:"deliveryRetryRequested"`
	DeliveryError          string            `json:"deliveryError"`
	ClaimToken             string            `json:"-"`
	ClaimExpiresAt         *time.Time        `json:"-"`
	Lines                  []LoungeOrderLine `gorm:"foreignKey:OrderID" json:"lines"`
}

type LoungeOrderLine struct {
	ID             uint    `gorm:"primaryKey" json:"id"`
	OrderID        uint    `json:"orderId"`
	ItemID         uint    `json:"itemId"`
	CategoryID     uint    `json:"categoryId"`
	ItemName       string  `json:"itemName"`
	ItemNameSv     *string `json:"itemNameSv,omitempty"`
	CategoryName   string  `json:"categoryName"`
	CategoryNameSv *string `json:"categoryNameSv,omitempty"`
	Quantity       int64   `json:"quantity"`
	Position       int     `json:"position"`
}
