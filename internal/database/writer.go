package database

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/anker-solix-exporter/anker-solix-exporter/internal/anker"
)

// Writer handles writing measurements to TimescaleDB via GORM
type Writer struct {
	db     *gorm.DB
	logger *zap.Logger
}

// NewWriter creates a new database writer
func NewWriter(dsn string, zapLogger *zap.Logger) (*Writer, error) {
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: newZapGormLogger(zapLogger),
		NowFunc: func() time.Time {
			return time.Now().UTC()
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Get underlying SQL DB for connection pool settings
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get database instance: %w", err)
	}

	// Configure connection pool
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetConnMaxLifetime(time.Hour)
	sqlDB.SetConnMaxIdleTime(10 * time.Minute) // Close idle connections after 10 minutes

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := sqlDB.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	zapLogger.Info("connected to TimescaleDB")

	return &Writer{
		db:     db,
		logger: zapLogger,
	}, nil
}

// WriteMeasurements writes measurements to the database
func (w *Writer) WriteMeasurements(ctx context.Context, measurements []anker.Measurement) error {
	if len(measurements) == 0 {
		return nil
	}

	// First, ensure sites and devices exist
	if err := w.ensureSitesAndDevices(ctx, measurements); err != nil {
		return fmt.Errorf("failed to ensure sites and devices: %w", err)
	}

	// Convert to database models
	dbMeasurements := make([]Measurement, len(measurements))
	for i, m := range measurements {
		dbMeasurements[i] = Measurement{
			Timestamp:    m.Timestamp,
			DeviceSN:     m.DeviceSN,
			SolarPower:   m.SolarPower,
			OutputPower:  m.OutputPower,
			GridPower:    m.GridPower,
			BatteryPower: m.BatteryPower,
			BatterySoC:   m.BatterySoC,
		}
	}

	// Batch insert
	result := w.db.WithContext(ctx).CreateInBatches(dbMeasurements, 100)
	if result.Error != nil {
		return fmt.Errorf("failed to insert measurements: %w", result.Error)
	}

	w.logger.Debug("wrote measurements to database",
		zap.Int("count", len(measurements)),
	)

	return nil
}

// ensureSitesAndDevices ensures that sites and devices exist in the database
func (w *Writer) ensureSitesAndDevices(ctx context.Context, measurements []anker.Measurement) error {
	// Collect unique sites and devices
	sitesMap := make(map[string]string) // siteID -> siteName
	devicesMap := make(map[string]struct {
		siteID     string
		deviceName string
		deviceType string
	})

	for _, m := range measurements {
		sitesMap[m.SiteID] = m.SiteName
		devicesMap[m.DeviceSN] = struct {
			siteID     string
			deviceName string
			deviceType string
		}{
			siteID:     m.SiteID,
			deviceName: m.DeviceName,
			deviceType: m.DeviceType,
		}
	}

	// Upsert sites
	for siteID, siteName := range sitesMap {
		site := Site{
			SiteID:   siteID,
			SiteName: siteName,
		}
		result := w.db.WithContext(ctx).
			Where(Site{SiteID: siteID}).
			Assign(Site{SiteName: siteName}).
			FirstOrCreate(&site)
		if result.Error != nil {
			return fmt.Errorf("failed to upsert site %s: %w", siteID, result.Error)
		}
	}

	// Upsert devices
	for deviceSN, deviceInfo := range devicesMap {
		device := Device{
			SiteID:     deviceInfo.siteID,
			DeviceSN:   deviceSN,
			DeviceName: deviceInfo.deviceName,
			DeviceType: deviceInfo.deviceType,
		}
		result := w.db.WithContext(ctx).
			Where(Device{DeviceSN: deviceSN}).
			Assign(Device{
				SiteID:     deviceInfo.siteID,
				DeviceName: deviceInfo.deviceName,
				DeviceType: deviceInfo.deviceType,
			}).
			FirstOrCreate(&device)
		if result.Error != nil {
			return fmt.Errorf("failed to upsert device %s: %w", deviceSN, result.Error)
		}
	}

	return nil
}

// GetLastTimestamp retrieves the latest timestamp for a specific device
func (w *Writer) GetLastTimestamp(ctx context.Context, siteID, deviceSN string) (time.Time, error) {
	var measurement Measurement

	result := w.db.WithContext(ctx).
		Where("device_sn = ?", deviceSN).
		Order("timestamp DESC").
		First(&measurement)

	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			// No data found, return zero time
			return time.Time{}, nil
		}
		return time.Time{}, fmt.Errorf("failed to query last timestamp: %w", result.Error)
	}

	return measurement.Timestamp, nil
}

// Close closes the database connection
func (w *Writer) Close() error {
	sqlDB, err := w.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

// GetDB returns the underlying GORM DB instance
func (w *Writer) GetDB() *gorm.DB {
	return w.db
}

// zapGormLogger is a GORM logger that uses zap
type zapGormLogger struct {
	zap    *zap.Logger
	config logger.Config
}

func newZapGormLogger(zapLogger *zap.Logger) logger.Interface {
	return &zapGormLogger{
		zap: zapLogger.Named("gorm"),
		config: logger.Config{
			SlowThreshold:             200 * time.Millisecond,
			LogLevel:                  logger.Warn,
			IgnoreRecordNotFoundError: true,
			Colorful:                  false,
		},
	}
}

func (l *zapGormLogger) LogMode(level logger.LogLevel) logger.Interface {
	newLogger := *l
	newLogger.config.LogLevel = level
	return &newLogger
}

func (l *zapGormLogger) Info(ctx context.Context, msg string, data ...interface{}) {
	if l.config.LogLevel >= logger.Info {
		l.zap.Sugar().Infof(msg, data...)
	}
}

func (l *zapGormLogger) Warn(ctx context.Context, msg string, data ...interface{}) {
	if l.config.LogLevel >= logger.Warn {
		l.zap.Sugar().Warnf(msg, data...)
	}
}

func (l *zapGormLogger) Error(ctx context.Context, msg string, data ...interface{}) {
	if l.config.LogLevel >= logger.Error {
		l.zap.Sugar().Errorf(msg, data...)
	}
}

func (l *zapGormLogger) Trace(ctx context.Context, begin time.Time, fc func() (string, int64), err error) {
	if l.config.LogLevel <= logger.Silent {
		return
	}

	elapsed := time.Since(begin)
	sql, rows := fc()

	switch {
	case err != nil && l.config.LogLevel >= logger.Error && (!l.config.IgnoreRecordNotFoundError || err != gorm.ErrRecordNotFound):
		l.zap.Error("database error",
			zap.Error(err),
			zap.Duration("elapsed", elapsed),
			zap.Int64("rows", rows),
			zap.String("sql", sql),
		)
	case elapsed > l.config.SlowThreshold && l.config.SlowThreshold != 0 && l.config.LogLevel >= logger.Warn:
		l.zap.Warn("slow query",
			zap.Duration("elapsed", elapsed),
			zap.Duration("threshold", l.config.SlowThreshold),
			zap.Int64("rows", rows),
			zap.String("sql", sql),
		)
	case l.config.LogLevel >= logger.Info:
		l.zap.Debug("database query",
			zap.Duration("elapsed", elapsed),
			zap.Int64("rows", rows),
			zap.String("sql", sql),
		)
	}
}
