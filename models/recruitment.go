package models

import "time"

type RecruitmentPeriod struct {
	ID        uint              `gorm:"primaryKey;autoIncrement;column:id;not null" json:"id"`
	EventroID *string           `gorm:"column:eventro_id;uniqueIndex" json:"eventroId,omitempty"`
	Name      string            `gorm:"column:name;not null;default:''" json:"name"`
	Link      string            `gorm:"column:link;not null;default:''" json:"link"`
	StartDate *time.Time        `gorm:"column:start_date" json:"startDate,omitempty"`
	EndDate   *time.Time        `gorm:"column:end_date" json:"endDate,omitempty"`
	Roles     []RecruitmentRole `gorm:"foreignKey:RecruitmentID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"roles,omitempty"`
}

type RecruitmentRole struct {
	ID            uint               `gorm:"primaryKey;autoIncrement;column:id;not null" json:"id"`
	EventroRoleID *string            `gorm:"column:eventro_role_id;uniqueIndex" json:"eventroRoleId,omitempty"`
	RecruitmentID uint               `gorm:"column:recruitment_id;not null;index" json:"recruitmentId"`
	Recruitment   *RecruitmentPeriod `gorm:"foreignKey:RecruitmentID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"recruitment,omitempty"`
	TeamID        *uint              `gorm:"column:team_id;index" json:"team_id,omitempty"`
	Team          *Team              `gorm:"foreignKey:TeamID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL;" json:"team,omitempty"`
	Name          string             `gorm:"column:name;not null;default:''" json:"name"`
	Description   string             `gorm:"column:description;not null;default:''" json:"description"`
}
