package models

type Tier string

const (
	TierStandard Tier = "Standard"
	TierBronze   Tier = "Bronze"
	TierSilver   Tier = "Silver"
	TierGold     Tier = "Gold"
)

type Exhibitor struct {
	ID              uint    `gorm:"primaryKey;autoIncrement;column:id;not null" json:"id"`
	EventroID       string  `gorm:"column:eventro_id;uniqueIndex" json:"eventroId"`
	Name            string  `gorm:"column:name;not null" json:"name"`
	Type            string  `gorm:"column:type;not null" json:"type"`
	Tier            *Tier   `gorm:"column:tier" json:"tier,omitempty"`
	CompanyWebsite  *string `gorm:"column:company_website" json:"companyWebsite,omitempty"`
	About           *string `gorm:"column:about" json:"about,omitempty"`
	Purpose         *string `gorm:"column:purpose" json:"purpose,omitempty"`
	LogoSquaredUrl  *string `gorm:"column:logo_squared_url" json:"logoSquared,omitempty"`
	LogoFreesizeUrl *string `gorm:"column:logo_freesize_url" json:"logoFreesize,omitempty"`
	MapImg          *string `gorm:"column:map_img" json:"mapImg,omitempty"`

	Industries  []Industry   `gorm:"many2many:exhibitor_industries;" json:"industries,omitempty"`
	Programs    []Program    `gorm:"many2many:exhibitor_programs;" json:"programs,omitempty"`
	Employments []Employment `gorm:"many2many:exhibitor_employments;" json:"emplyoments,omitempty"`

	Cities              *string `gorm:"column:cities" json:"cities,omitempty"`
	FairLocation        string  `gorm:"column:fair_location;not null" json:"fairLocation"`
	VyerPosition        *string `gorm:"column:vyer_position" json:"vyerPosition,omitempty"`
	LocationSpecial     *string `gorm:"column:location_special" json:"locationSpecial,omitempty"`
	ClimateCompensation bool    `gorm:"column:climate_compensation;not null" json:"climateCompensation"`
	Flyer               string  `gorm:"column:flyer;not null" json:"flyer"`
}

// {"id":2,"name":"test","type":"test","tier":"test","companyWebsite":"test","about":"test","purpose":"test","fairLocation":"","climateCompensation":false,"flyer":"","programs":[3,2]}
//   id: number
//   name: string
//   type: string
//   tier?: string
//   company_website?: string
//   about?: string
//   purpose?: string
//   logo_squared?: string
//   logo_freesize?: string
//   map_img?: string
//   industries: Industry[]
//   employments: Employment[]
//   locations: Location[]
//   cities?: string
//   fair_location: string
//   vyer_position?: string
//   location_special?: string
//   climate_compensation: boolean
//   flyer: string
// } (edited)
