package translations

import "time"

type Job struct {
	Outdated        bool       `gorm:"-" json:"outdated"`
	ID              uint64     `gorm:"primaryKey" json:"id"`
	Resource        string     `json:"resource"`
	RecordID        uint64     `json:"recordId"`
	RequestedBy     int        `json:"requestedBy"`
	Status          string     `json:"status"`
	DedupKey        string     `json:"-"`
	SourceJSON      string     `gorm:"type:jsonb" json:"-"`
	FieldIDs        string     `gorm:"type:jsonb" json:"-"`
	Fingerprint     string     `json:"fingerprint"`
	Model           string     `json:"model"`
	Prompt          string     `json:"-"`
	Glossary        string     `gorm:"type:jsonb" json:"-"`
	PromptVersion   string     `json:"promptVersion"`
	GlossaryVersion string     `json:"glossaryVersion"`
	Attempts        int        `json:"attempts"`
	Error           string     `json:"error,omitempty"`
	Slot            *int       `json:"-"`
	LeaseUntil      *time.Time `json:"-"`
	NotBefore       time.Time  `json:"notBefore"`
	ReservedTokens  int64      `json:"-"`
	ReservationDay  string     `json:"-"`
	CreatedAt       time.Time  `json:"createdAt"`
	UpdatedAt       time.Time  `json:"updatedAt"`
	FinishedAt      *time.Time `json:"finishedAt,omitempty"`
}

func (Job) TableName() string { return "translation_jobs" }

type Suggestion struct {
	Model             string     `json:"model"`
	PromptVersion     string     `json:"promptVersion"`
	GlossaryVersion   string     `json:"glossaryVersion"`
	ID                uint64     `gorm:"primaryKey" json:"id"`
	Resource          string     `json:"resource"`
	RecordID          uint64     `json:"recordId"`
	FieldID           string     `json:"fieldId"`
	JobID             *uint64    `json:"jobId,omitempty"`
	Version           int        `json:"version"`
	SourceFingerprint string     `json:"sourceFingerprint"`
	TargetBefore      *string    `json:"targetBefore"`
	Generated         string     `json:"generated"`
	Draft             string     `json:"draft"`
	Status            string     `json:"status"`
	Warnings          string     `gorm:"type:jsonb" json:"-"`
	PublishedText     *string    `json:"publishedText,omitempty"`
	ReviewedBy        *int       `json:"reviewedBy,omitempty"`
	ReviewedAt        *time.Time `json:"reviewedAt,omitempty"`
	CreatedAt         time.Time  `json:"createdAt"`
	UpdatedAt         time.Time  `json:"updatedAt"`
}

func (Suggestion) TableName() string { return "translation_suggestions" }

type Usage struct {
	Day      string `gorm:"primaryKey" json:"day"`
	Used     int64  `json:"used"`
	Reserved int64  `json:"reserved"`
}

func (Usage) TableName() string { return "translation_usage_daily" }
