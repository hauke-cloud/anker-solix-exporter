package database

import (
	"time"

	"gorm.io/gorm"
)

// Measurement represents a single energy measurement from Anker Solix
type Measurement struct {
	ID           uint      `gorm:"primarykey"`
	Timestamp    time.Time `gorm:"type:timestamptz;not null;index:idx_measurements_timestamp"`
	SiteID       string    `gorm:"type:varchar(255);not null;index:idx_measurements_site_device"`
	SiteName     string    `gorm:"type:varchar(255);not null"`
	DeviceSN     string    `gorm:"type:varchar(255);not null;index:idx_measurements_site_device"`
	DeviceName   string    `gorm:"type:varchar(255);not null"`
	DeviceType   string    `gorm:"type:varchar(100)"`
	SolarPower   float64   `gorm:"type:double precision"`
	OutputPower  float64   `gorm:"type:double precision"`
	GridPower    float64   `gorm:"type:double precision"`
	BatteryPower float64   `gorm:"type:double precision"`
	BatterySoC   float64   `gorm:"type:double precision"`
	CreatedAt    time.Time `gorm:"type:timestamptz;not null;default:current_timestamp"`
}

// TableName overrides the table name
func (Measurement) TableName() string {
	return "measurements"
}

// BeforeCreate hook to ensure timestamp is set
func (m *Measurement) BeforeCreate(tx *gorm.DB) error {
	if m.Timestamp.IsZero() {
		m.Timestamp = time.Now()
	}
	return nil
}
