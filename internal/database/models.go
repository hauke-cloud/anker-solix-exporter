package database

import (
	"time"

	"gorm.io/gorm"
)

// Site represents a site location
type Site struct {
	ID        uint      `gorm:"primaryKey;autoIncrement"`
	SiteID    string    `gorm:"type:varchar(255);not null;uniqueIndex"`
	SiteName  string    `gorm:"type:varchar(255);not null"`
	CreatedAt time.Time `gorm:"type:timestamptz;not null;default:current_timestamp"`
	UpdatedAt time.Time `gorm:"type:timestamptz;not null;default:current_timestamp"`
	Devices   []Device  `gorm:"foreignKey:SiteID;references:SiteID"`
}

// TableName overrides the table name
func (Site) TableName() string {
	return "solar_sites"
}

// Device represents a device at a site
type Device struct {
	ID         uint      `gorm:"primaryKey;autoIncrement"`
	SiteID     string    `gorm:"type:varchar(255);not null;index:idx_devices_site_id"`
	DeviceSN   string    `gorm:"type:varchar(255);not null;uniqueIndex:idx_devices_device_sn"`
	DeviceName string    `gorm:"type:varchar(255);not null"`
	DeviceType string    `gorm:"type:varchar(100)"`
	CreatedAt  time.Time `gorm:"type:timestamptz;not null;default:current_timestamp"`
	UpdatedAt  time.Time `gorm:"type:timestamptz;not null;default:current_timestamp"`
	Site       Site      `gorm:"foreignKey:SiteID;references:SiteID"`
}

// TableName overrides the table name
func (Device) TableName() string {
	return "solar_devices"
}

// Measurement represents a single energy measurement from Anker Solix
type Measurement struct {
	ID           uint      `gorm:"primaryKey;autoIncrement"`
	Timestamp    time.Time `gorm:"type:timestamptz;not null;primaryKey;index:idx_measurements_timestamp"`
	DeviceSN     string    `gorm:"type:varchar(255);not null;index:idx_measurements_device_sn"`
	SolarPower   float64   `gorm:"type:double precision"`
	OutputPower  float64   `gorm:"type:double precision"`
	GridPower    float64   `gorm:"type:double precision"`
	BatteryPower float64   `gorm:"type:double precision"`
	BatterySoC   float64   `gorm:"column:battery_soc;type:double precision"`
	CreatedAt    time.Time `gorm:"type:timestamptz;not null;default:current_timestamp"`
	Device       Device    `gorm:"foreignKey:DeviceSN;references:DeviceSN"`
}

// TableName overrides the table name
func (Measurement) TableName() string {
	return "solar_measurements"
}

// BeforeCreate hook to ensure timestamp is set
func (m *Measurement) BeforeCreate(tx *gorm.DB) error {
	if m.Timestamp.IsZero() {
		m.Timestamp = time.Now()
	}
	return nil
}
