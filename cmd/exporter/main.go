package main

import (
	"context"
	"flag"
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

	// Initialize Anker client
	client := anker.NewClient(cfg.Anker.Email, cfg.Anker.Password, cfg.Anker.Country)

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
		client:      client,
		writer:      writer,
		state:       resumeState,
		config:      cfg,
		logger:      logger,
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

	// Run first poll immediately
	if err := e.poll(ctx); err != nil {
		e.logger.Error("initial poll failed", zap.Error(err))
	}

	for {
		select {
		case <-ctx.Done():
			// Save state before exit
			if err := e.state.Save(); err != nil {
				e.logger.Error("failed to save state", zap.Error(err))
			}
			return nil
		case <-ticker.C:
			if err := e.poll(ctx); err != nil {
				e.logger.Error("poll failed", zap.Error(err))
			}
		}
	}
}

func (e *Exporter) poll(ctx context.Context) error {
	// Login if not authenticated
	if !e.client.IsAuthenticated() {
		e.logger.Info("authenticating with Anker API")
		if err := e.client.Login(); err != nil {
			return err
		}
		e.logger.Info("successfully authenticated")
	}

	// Get sites
	sites, err := e.client.GetSites()
	if err != nil {
		// Token might have expired, try to re-login
		e.logger.Warn("failed to get sites, attempting re-login", zap.Error(err))
		if err := e.client.Login(); err != nil {
			return err
		}
		sites, err = e.client.GetSites()
		if err != nil {
			return err
		}
	}

	e.logger.Info("fetched sites", zap.Int("count", len(sites)))

	// Process each site and device
	for _, site := range sites {
		for _, device := range site.DeviceList {
			if err := e.processDevice(ctx, site, device); err != nil {
				e.logger.Error("failed to process device",
					zap.String("site_id", site.SiteID),
					zap.String("device_sn", device.DeviceSN),
					zap.Error(err),
				)
				// Continue with other devices
				continue
			}
			
			// Rate limiting: small delay between devices to respect API limits
			time.Sleep(2 * time.Second)
		}
	}

	// Update last poll time
	e.state.UpdatePollTime()
	
	// Save state after successful poll
	if err := e.state.Save(); err != nil {
		e.logger.Error("failed to save state", zap.Error(err))
	}

	return nil
}

func (e *Exporter) processDevice(ctx context.Context, site anker.Site, device anker.Device) error {
	deviceKey := site.SiteID + ":" + device.DeviceSN
	
	// Determine time range for data fetch
	// Use resume functionality to avoid duplicate imports
	defaultLookback := 7 * 24 * time.Hour // Default: look back 7 days on first run
	startTime := e.state.GetResumeTime(deviceKey, defaultLookback)
	endTime := time.Now()

	// If we already have recent data, skip if less than poll interval has passed
	if state, exists := e.state.GetDeviceState(deviceKey); exists {
		timeSinceLastExport := time.Since(state.LastExportTime)
		if timeSinceLastExport < e.config.GetPollInterval() {
			e.logger.Debug("skipping device, too soon since last export",
				zap.String("device_sn", device.DeviceSN),
				zap.Duration("time_since_last", timeSinceLastExport),
			)
			return nil
		}
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
	powerData, err := e.client.GetEnergyData(site.SiteID, device.DeviceSN, startTime, endTime)
	if err != nil {
		return err
	}

	if len(powerData) == 0 {
		e.logger.Debug("no new data for device",
			zap.String("device_sn", device.DeviceSN),
		)
		return nil
	}

	// Convert to measurements
	measurements := make([]anker.Measurement, 0, len(powerData))
	var latestDataTime time.Time
	
	for _, pd := range powerData {
		timestamp := time.Unix(pd.Time, 0)
		
		// Skip data we've already exported
		if !startTime.IsZero() && !timestamp.After(startTime) {
			continue
		}
		
		measurement := anker.Measurement{
			Timestamp:    timestamp,
			SiteID:       site.SiteID,
			SiteName:     site.SiteName,
			DeviceSN:     device.DeviceSN,
			DeviceName:   device.DeviceName,
			DeviceType:   device.DeviceType,
			SolarPower:   pd.Solar,
			OutputPower:  pd.Output,
			GridPower:    pd.Grid,
			BatteryPower: pd.Battery,
			BatterySoC:   pd.BatterySoC,
		}
		measurements = append(measurements, measurement)
		
		if timestamp.After(latestDataTime) {
			latestDataTime = timestamp
		}
	}

	if len(measurements) == 0 {
		e.logger.Debug("no new measurements after filtering",
			zap.String("device_sn", device.DeviceSN),
		)
		return nil
	}

	// Write to database
	if err := e.writer.WriteMeasurements(ctx, measurements); err != nil {
		return err
	}

	e.logger.Info("exported measurements",
		zap.String("device_sn", device.DeviceSN),
		zap.Int("count", len(measurements)),
		zap.Time("latest_data", latestDataTime),
	)

	// Update resume state
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
