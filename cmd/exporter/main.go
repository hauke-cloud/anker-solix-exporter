package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/anker-solix-exporter/anker-solix-exporter/internal/anker"
	"github.com/anker-solix-exporter/anker-solix-exporter/internal/config"
	"github.com/anker-solix-exporter/anker-solix-exporter/internal/database"
	"github.com/anker-solix-exporter/anker-solix-exporter/internal/resume"
)

var (
	configPath = flag.String("config", "/etc/anker-solix-exporter/config.yaml", "Path to config file")
	version    = "dev"
	commit     = "none"
	date       = "unknown"
)

func main() {
	flag.Parse()

	// Load configuration
	cfg, err := config.LoadConfig(*configPath)
	if err != nil {
		panic(err)
	}

	// Setup logger
	logger := setupLogger(cfg.Exporter.LogLevel)
	defer logger.Sync()

	logger.Info("starting anker-solix-exporter",
		zap.String("version", version),
		zap.String("commit", commit),
		zap.String("date", date),
	)

	// Initialize resume state
	resumeState, err := resume.NewState(cfg.Exporter.ResumeFile, logger)
	if err != nil {
		logger.Fatal("failed to initialize resume state", zap.Error(err))
	}

	// Log database connection configuration (without sensitive data)
	logger.Info("database configuration",
		zap.String("host", cfg.Database.Host),
		zap.Int("port", cfg.Database.Port),
		zap.String("user", cfg.Database.User),
		zap.String("database", cfg.Database.Database),
		zap.String("sslmode", cfg.Database.SSLMode),
		zap.String("sslcert", cfg.Database.SSLCert),
		zap.String("sslkey", cfg.Database.SSLKey),
		zap.String("sslrootcert", cfg.Database.SSLRootCert),
	)

	// Initialize database writer
	writer, err := database.NewWriter(cfg.GetDSN(), logger)
	if err != nil {
		logger.Fatal("failed to initialize database writer", zap.Error(err))
	}
	defer writer.Close()

	// Run database migrations
	if err := database.RunMigrations(writer.GetDB(), cfg.Database.MigrationsPath, logger); err != nil {
		logger.Fatal("failed to run database migrations", zap.Error(err))
	}

	// Initialize Anker client with logger
	// Create a child logger for the Anker client with appropriate level
	var ankerLogger *zap.Logger
	if cfg.Anker.Debug {
		// Use debug level for Anker API calls
		ankerLogger = logger.WithOptions(zap.IncreaseLevel(zapcore.DebugLevel))
		logger.Info("Anker API debug logging enabled")
	} else {
		// Use info level for production
		ankerLogger = logger.WithOptions(zap.IncreaseLevel(zapcore.InfoLevel))
	}
	
	client := anker.NewClientWithLogger(cfg.Anker.Email, cfg.Anker.Password, cfg.Anker.Country, ankerLogger)
	
	// Configure rate limiting
	if cfg.Anker.EndpointLimit > 0 {
		client.SetEndpointLimit(cfg.Anker.EndpointLimit)
	}
	if cfg.Anker.RequestDelay > 0 {
		client.SetRequestDelay(time.Duration(cfg.Anker.RequestDelay * float64(time.Second)))
	}
	
	logger.Info("Anker client configured",
		zap.Int("endpoint_limit", cfg.Anker.EndpointLimit),
		zap.Float64("request_delay_seconds", cfg.Anker.RequestDelay),
	)

	// Setup context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		logger.Info("received shutdown signal")
		cancel()
	}()

	// Create exporter
	exp := &Exporter{
		client: client,
		writer: writer,
		state:  resumeState,
		config: cfg,
		logger: logger,
	}

	// Run exporter
	if err := exp.Run(ctx); err != nil {
		logger.Fatal("exporter failed", zap.Error(err))
	}

	logger.Info("exporter stopped gracefully")
}

type Exporter struct {
	client *anker.Client
	writer *database.Writer
	state  *resume.State
	config *config.Config
	logger *zap.Logger
}

func (e *Exporter) Run(ctx context.Context) error {
	pollInterval := e.config.GetPollInterval()
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	e.logger.Info("exporter started",
		zap.Duration("poll_interval", pollInterval),
	)

	// Run first poll immediately
	e.logger.Debug("running initial poll")
	if err := e.poll(ctx); err != nil {
		e.logger.Error("initial poll failed", zap.Error(err))
	}

	e.logger.Debug("entering main poll loop",
		zap.Duration("poll_interval", pollInterval),
	)

	pollCount := 1 // Already did initial poll
	for {
		select {
		case <-ctx.Done():
			e.logger.Info("shutdown signal received",
				zap.Int("total_polls_completed", pollCount),
			)
			// Save state before exit
			e.logger.Debug("saving final state before shutdown")
			if err := e.state.Save(); err != nil {
				e.logger.Error("failed to save state", zap.Error(err))
			} else {
				e.logger.Debug("final state saved successfully")
			}
			return nil
		case tickTime := <-ticker.C:
			pollCount++
			e.logger.Debug("poll cycle triggered",
				zap.Time("tick_time", tickTime),
				zap.Int("poll_number", pollCount),
			)
			if err := e.poll(ctx); err != nil {
				e.logger.Error("poll failed",
					zap.Int("poll_number", pollCount),
					zap.Error(err),
				)
			}
			nextPoll := time.Now().Add(pollInterval)
			e.logger.Debug("poll cycle completed, waiting for next",
				zap.Int("poll_number", pollCount),
				zap.Time("next_poll_at", nextPoll),
				zap.Duration("wait_duration", pollInterval),
			)
		}
	}
}

func (e *Exporter) poll(ctx context.Context) error {
	e.logger.Debug("starting poll cycle")

	// Login if not authenticated
	if !e.client.IsAuthenticated() {
		e.logger.Info("authenticating with Anker API")
		if err := e.client.Login(); err != nil {
			e.logger.Error("authentication failed", zap.Error(err))
			return err
		}
		e.logger.Info("successfully authenticated")
	} else {
		e.logger.Debug("already authenticated, skipping login")
	}

	// Get sites
	e.logger.Debug("fetching sites from Anker API")
	sites, err := e.client.GetSites()
	if err != nil {
		// Token might have expired, try to re-login
		e.logger.Warn("failed to get sites, attempting re-login", zap.Error(err))
		if err := e.client.Login(); err != nil {
			e.logger.Error("re-authentication failed", zap.Error(err))
			return err
		}
		e.logger.Debug("re-authentication successful, retrying GetSites")
		sites, err = e.client.GetSites()
		if err != nil {
			e.logger.Error("failed to get sites after re-authentication", zap.Error(err))
			return err
		}
	}

	e.logger.Info("fetched sites", zap.Int("count", len(sites)))

	// Log details of each site
	for i, site := range sites {
		e.logger.Debug("site details",
			zap.Int("site_index", i),
			zap.String("site_id", site.SiteID),
			zap.String("site_name", site.SiteName),
			zap.Int("device_count", len(site.DeviceList)),
		)
	}

	// Process each site and device
	processedDevices := 0
	skippedDevices := 0
	failedDevices := 0
	totalMeasurements := 0

	for siteIdx, site := range sites {
		e.logger.Debug("processing site",
			zap.Int("site_index", siteIdx),
			zap.String("site_id", site.SiteID),
			zap.String("site_name", site.SiteName),
			zap.Int("devices_to_process", len(site.DeviceList)),
		)

		// First, get current measurements for the site
		e.logger.Debug("fetching current measurements for site",
			zap.String("site_id", site.SiteID),
		)
		currentMeasurements, err := e.client.GetCurrentMeasurements(site.SiteID)
		if err != nil {
			e.logger.Warn("failed to get current measurements for site",
				zap.String("site_id", site.SiteID),
				zap.Error(err),
			)
		} else if len(currentMeasurements) > 0 {
			e.logger.Info("fetched current measurements",
				zap.String("site_id", site.SiteID),
				zap.Int("count", len(currentMeasurements)),
			)
			
			// Write current measurements to database
			if err := e.writer.WriteMeasurements(ctx, currentMeasurements); err != nil {
				e.logger.Error("failed to write current measurements",
					zap.String("site_id", site.SiteID),
					zap.Error(err),
				)
			} else {
				totalMeasurements += len(currentMeasurements)
				e.logger.Debug("wrote current measurements to database",
					zap.String("site_id", site.SiteID),
					zap.Int("count", len(currentMeasurements)),
				)
			}
		}

		for deviceIdx, device := range site.DeviceList {
			e.logger.Debug("processing device",
				zap.Int("site_index", siteIdx),
				zap.Int("device_index", deviceIdx),
				zap.String("site_id", site.SiteID),
				zap.String("device_sn", device.DeviceSN),
				zap.String("device_name", device.DeviceName),
				zap.String("device_type", device.DeviceType),
			)

			if err := e.processDevice(ctx, site, device); err != nil {
				e.logger.Error("failed to process device",
					zap.String("site_id", site.SiteID),
					zap.String("site_name", site.SiteName),
					zap.String("device_sn", device.DeviceSN),
					zap.String("device_name", device.DeviceName),
					zap.Error(err),
				)
				failedDevices++
				// Continue with other devices
				continue
			}

			processedDevices++

			// Rate limiting: small delay between devices to respect API limits
			e.logger.Debug("rate limiting delay before next device",
				zap.Duration("delay", 2*time.Second),
			)
			time.Sleep(2 * time.Second)
		}
	}

	e.logger.Info("poll cycle completed",
		zap.Int("total_sites", len(sites)),
		zap.Int("processed_devices", processedDevices),
		zap.Int("skipped_devices", skippedDevices),
		zap.Int("failed_devices", failedDevices),
		zap.Int("current_measurements", totalMeasurements),
	)

	// Update last poll time
	e.logger.Debug("updating last poll time")
	e.state.UpdatePollTime()

	// Save state after successful poll
	e.logger.Debug("saving state to disk")
	if err := e.state.Save(); err != nil {
		e.logger.Error("failed to save state", zap.Error(err))
	} else {
		e.logger.Debug("state saved successfully")
	}

	return nil
}

func (e *Exporter) processDevice(ctx context.Context, site anker.Site, device anker.Device) error {
	deviceKey := site.SiteID + ":" + device.DeviceSN

	e.logger.Debug("processing device start",
		zap.String("device_key", deviceKey),
		zap.String("site_id", site.SiteID),
		zap.String("device_sn", device.DeviceSN),
	)

	// Determine time range for data fetch
	// Use resume functionality to avoid duplicate imports
	defaultLookback := 7 * 24 * time.Hour // Default: look back 7 days on first run
	startTime := e.state.GetResumeTime(deviceKey, defaultLookback)
	endTime := time.Now()

	e.logger.Debug("calculated time range for data fetch",
		zap.String("device_key", deviceKey),
		zap.Time("start_time", startTime),
		zap.Time("end_time", endTime),
		zap.Duration("time_range", endTime.Sub(startTime)),
	)

	// If we already have recent data, skip if less than poll interval has passed
	if state, exists := e.state.GetDeviceState(deviceKey); exists {
		timeSinceLastExport := time.Since(state.LastExportTime)
		pollInterval := e.config.GetPollInterval()

		e.logger.Debug("found existing device state",
			zap.String("device_key", deviceKey),
			zap.Time("last_export_time", state.LastExportTime),
			zap.Duration("time_since_last_export", timeSinceLastExport),
			zap.Duration("poll_interval", pollInterval),
			zap.Bool("should_skip", timeSinceLastExport < pollInterval),
		)

		if timeSinceLastExport < pollInterval {
			e.logger.Debug("skipping device, too soon since last export",
				zap.String("device_sn", device.DeviceSN),
				zap.String("device_name", device.DeviceName),
				zap.Duration("time_since_last", timeSinceLastExport),
				zap.Duration("required_interval", pollInterval),
			)
			return nil
		}
	} else {
		e.logger.Debug("no existing device state found, first time processing",
			zap.String("device_key", deviceKey),
			zap.Duration("default_lookback", defaultLookback),
		)
	}

	e.logger.Info("fetching energy data",
		zap.String("site_id", site.SiteID),
		zap.String("site_name", site.SiteName),
		zap.String("device_sn", device.DeviceSN),
		zap.String("device_name", device.DeviceName),
		zap.Time("start_time", startTime),
		zap.Time("end_time", endTime),
	)

	// Fetch energy data
	e.logger.Debug("calling Anker API GetEnergyData",
		zap.String("device_sn", device.DeviceSN),
	)
	powerData, err := e.client.GetEnergyData(site.SiteID, device.DeviceSN, startTime, endTime)
	if err != nil {
		e.logger.Error("failed to fetch energy data from Anker API",
			zap.String("device_sn", device.DeviceSN),
			zap.Error(err),
		)
		return err
	}

	e.logger.Debug("received power data from API",
		zap.String("device_sn", device.DeviceSN),
		zap.Int("raw_data_points", len(powerData)),
	)

	if len(powerData) == 0 {
		e.logger.Debug("no new data for device",
			zap.String("device_sn", device.DeviceSN),
			zap.String("device_name", device.DeviceName),
		)
		return nil
	}

	// Convert to measurements
	measurements := make([]anker.Measurement, 0, len(powerData))
	var latestDataTime time.Time
	var earliestDataTime time.Time
	skippedCount := 0

	e.logger.Debug("converting power data to measurements",
		zap.String("device_sn", device.DeviceSN),
		zap.Int("power_data_count", len(powerData)),
	)

	for i, pd := range powerData {
		// Parse date string to time
		timestamp, err := time.Parse("2006-01-02", pd.Time)
		if err != nil {
			e.logger.Warn("failed to parse timestamp",
				zap.String("device_sn", device.DeviceSN),
				zap.String("time_string", pd.Time),
				zap.Error(err),
			)
			continue
		}

		// Skip data we've already exported
		if !startTime.IsZero() && !timestamp.After(startTime) {
			e.logger.Debug("skipping already exported data point",
				zap.String("device_sn", device.DeviceSN),
				zap.Int("data_point_index", i),
				zap.Time("timestamp", timestamp),
				zap.Time("resume_time", startTime),
			)
			skippedCount++
			continue
		}

		// Parse the value string to float64
		value := 0.0
		if pd.Value != "" {
			if val, err := parseFloat(pd.Value); err == nil {
				value = val
			}
		}

		measurement := anker.Measurement{
			Timestamp:    timestamp,
			SiteID:       site.SiteID,
			SiteName:     site.SiteName,
			DeviceSN:     device.DeviceSN,
			DeviceName:   device.DeviceName,
			DeviceType:   device.DeviceType,
			SolarPower:   value, // The value represents energy for the day
			OutputPower:  0,
			GridPower:    0,
			BatteryPower: 0,
			BatterySoC:   0,
		}
		measurements = append(measurements, measurement)

		if timestamp.After(latestDataTime) {
			latestDataTime = timestamp
		}
		if earliestDataTime.IsZero() || timestamp.Before(earliestDataTime) {
			earliestDataTime = timestamp
		}

		// Log first and last measurements at debug level
		if i == 0 || i == len(powerData)-1 {
			e.logger.Debug("measurement data point",
				zap.String("device_sn", device.DeviceSN),
				zap.Int("index", i),
				zap.Time("timestamp", timestamp),
				zap.String("raw_value", pd.Value),
				zap.Float64("parsed_value", value),
			)
		}
	}

	e.logger.Debug("measurement conversion complete",
		zap.String("device_sn", device.DeviceSN),
		zap.Int("total_power_data", len(powerData)),
		zap.Int("skipped_data_points", skippedCount),
		zap.Int("new_measurements", len(measurements)),
	)

	if len(measurements) == 0 {
		e.logger.Debug("no new measurements after filtering",
			zap.String("device_sn", device.DeviceSN),
			zap.String("device_name", device.DeviceName),
			zap.Int("total_received", len(powerData)),
			zap.Int("skipped", skippedCount),
		)
		return nil
	}

	// Write to database
	e.logger.Debug("writing measurements to database",
		zap.String("device_sn", device.DeviceSN),
		zap.Int("measurement_count", len(measurements)),
		zap.Time("earliest_timestamp", earliestDataTime),
		zap.Time("latest_timestamp", latestDataTime),
	)

	if err := e.writer.WriteMeasurements(ctx, measurements); err != nil {
		e.logger.Error("failed to write measurements to database",
			zap.String("device_sn", device.DeviceSN),
			zap.Int("measurement_count", len(measurements)),
			zap.Error(err),
		)
		return err
	}

	e.logger.Info("exported measurements",
		zap.String("device_sn", device.DeviceSN),
		zap.String("device_name", device.DeviceName),
		zap.Int("count", len(measurements)),
		zap.Time("earliest_data", earliestDataTime),
		zap.Time("latest_data", latestDataTime),
	)

	// Update resume state
	e.logger.Debug("updating resume state",
		zap.String("device_key", deviceKey),
		zap.Time("new_resume_time", latestDataTime),
	)
	e.state.UpdateDeviceState(deviceKey, latestDataTime)

	return nil
}

func setupLogger(level string) *zap.Logger {
	var zapLevel zapcore.Level
	switch level {
	case "debug":
		zapLevel = zapcore.DebugLevel
	case "info":
		zapLevel = zapcore.InfoLevel
	case "warn":
		zapLevel = zapcore.WarnLevel
	case "error":
		zapLevel = zapcore.ErrorLevel
	default:
		zapLevel = zapcore.InfoLevel
	}

	config := zap.NewProductionConfig()
	config.Level = zap.NewAtomicLevelAt(zapLevel)
	config.EncoderConfig.TimeKey = "timestamp"
	config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder

	logger, err := config.Build()
	if err != nil {
		panic(err)
	}

	return logger
}

// parseFloat safely converts a string to float64
func parseFloat(s string) (float64, error) {
	if s == "" {
		return 0, nil
	}
	var val float64
	_, err := fmt.Sscanf(s, "%f", &val)
	return val, err
}
