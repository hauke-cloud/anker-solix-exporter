package database

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// TestDatabase represents a test database instance with helper methods
type TestDatabase struct {
	Container testcontainers.Container
	Writer    *Writer
	DSN       string
	Logger    *zap.Logger
	ctx       context.Context
}

// SetupTestDatabase creates a new PostgreSQL container and initializes the database
func SetupTestDatabase(t *testing.T) *TestDatabase {
	t.Helper()

	ctx := context.Background()

	// Start PostgreSQL container
	pgContainer, err := postgres.Run(ctx,
		"postgres:16-alpine",
		postgres.WithDatabase("testdb"),
		postgres.WithUsername("testuser"),
		postgres.WithPassword("testpass"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(30*time.Second)),
	)
	if err != nil {
		t.Fatalf("failed to start postgres container: %s", err)
	}

	// Get connection string
	dsn, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		_ = testcontainers.TerminateContainer(pgContainer)
		t.Fatalf("failed to get connection string: %s", err)
	}

	// Create logger
	logger, err := zap.NewDevelopment()
	if err != nil {
		_ = testcontainers.TerminateContainer(pgContainer)
		t.Fatalf("failed to create logger: %s", err)
	}

	// Create writer
	writer, err := NewWriter(dsn, logger)
	if err != nil {
		_ = testcontainers.TerminateContainer(pgContainer)
		t.Fatalf("failed to create writer: %s", err)
	}

	// Run migrations
	if err := RunMigrations(writer.GetDB(), "../../migrations", logger); err != nil {
		writer.Close()
		_ = testcontainers.TerminateContainer(pgContainer)
		t.Fatalf("failed to run migrations: %s", err)
	}

	return &TestDatabase{
		Container: pgContainer,
		Writer:    writer,
		DSN:       dsn,
		Logger:    logger,
		ctx:       ctx,
	}
}

// Cleanup terminates the container and closes connections
func (td *TestDatabase) Cleanup(t *testing.T) {
	t.Helper()

	if td.Writer != nil {
		if err := td.Writer.Close(); err != nil {
			t.Logf("warning: failed to close writer: %s", err)
		}
	}

	if td.Container != nil {
		if err := testcontainers.TerminateContainer(td.Container); err != nil {
			t.Logf("warning: failed to terminate container: %s", err)
		}
	}
}

// Reset clears all data from the database tables
func (td *TestDatabase) Reset(t *testing.T) {
	t.Helper()

	// Truncate all tables to reset state
	if err := td.Writer.GetDB().Exec("TRUNCATE TABLE measurements CASCADE").Error; err != nil {
		t.Fatalf("failed to truncate measurements table: %s", err)
	}

	// Reset any sequences
	if err := td.Writer.GetDB().Exec("ALTER SEQUENCE measurements_id_seq RESTART WITH 1").Error; err != nil {
		t.Logf("warning: failed to reset sequence: %s", err)
	}
}

// Count returns the number of rows in the measurements table
func (td *TestDatabase) Count(t *testing.T) int64 {
	t.Helper()

	var count int64
	if err := td.Writer.GetDB().Model(&Measurement{}).Count(&count).Error; err != nil {
		t.Fatalf("failed to count measurements: %s", err)
	}
	return count
}

// CountWhere returns the number of rows matching the query
func (td *TestDatabase) CountWhere(t *testing.T, query string, args ...interface{}) int64 {
	t.Helper()

	var count int64
	if err := td.Writer.GetDB().Model(&Measurement{}).Where(query, args...).Count(&count).Error; err != nil {
		t.Fatalf("failed to count measurements: %s", err)
	}
	return count
}

// GetMeasurement retrieves a single measurement by ID
func (td *TestDatabase) GetMeasurement(t *testing.T, id uint) *Measurement {
	t.Helper()

	var m Measurement
	if err := td.Writer.GetDB().First(&m, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil
		}
		t.Fatalf("failed to get measurement: %s", err)
	}
	return &m
}

// GetMeasurements retrieves all measurements
func (td *TestDatabase) GetMeasurements(t *testing.T) []Measurement {
	t.Helper()

	var measurements []Measurement
	if err := td.Writer.GetDB().Find(&measurements).Error; err != nil {
		t.Fatalf("failed to get measurements: %s", err)
	}
	return measurements
}

// GetMeasurementsWhere retrieves measurements matching the query
func (td *TestDatabase) GetMeasurementsWhere(t *testing.T, query string, args ...interface{}) []Measurement {
	t.Helper()

	var measurements []Measurement
	if err := td.Writer.GetDB().Where(query, args...).Find(&measurements).Error; err != nil {
		t.Fatalf("failed to get measurements: %s", err)
	}
	return measurements
}

// IndexExists checks if an index exists on the table
func (td *TestDatabase) IndexExists(t *testing.T, indexName string) bool {
	t.Helper()

	var count int64
	query := `
		SELECT COUNT(*) FROM pg_indexes 
		WHERE tablename = 'measurements' 
		AND indexname = $1
	`
	if err := td.Writer.GetDB().Raw(query, indexName).Scan(&count).Error; err != nil {
		t.Fatalf("failed to check index: %s", err)
	}
	return count > 0
}

// HasIndexLike checks if an index matching the pattern exists
func (td *TestDatabase) HasIndexLike(t *testing.T, pattern string) bool {
	t.Helper()

	var count int64
	query := `
		SELECT COUNT(*) FROM pg_indexes 
		WHERE tablename = 'measurements' 
		AND indexname LIKE $1
	`
	if err := td.Writer.GetDB().Raw(query, pattern).Scan(&count).Error; err != nil {
		t.Fatalf("failed to check index: %s", err)
	}
	return count > 0
}

// ExecSQL executes raw SQL (useful for testing)
func (td *TestDatabase) ExecSQL(t *testing.T, sql string, args ...interface{}) {
	t.Helper()

	if err := td.Writer.GetDB().Exec(sql, args...).Error; err != nil {
		t.Fatalf("failed to execute SQL: %s", err)
	}
}

// AssertCount asserts the total number of measurements
func (td *TestDatabase) AssertCount(t *testing.T, expected int64) {
	t.Helper()

	count := td.Count(t)
	if count != expected {
		t.Errorf("expected %d measurements, got %d", expected, count)
	}
}

// AssertCountWhere asserts the number of measurements matching the query
func (td *TestDatabase) AssertCountWhere(t *testing.T, expected int64, query string, args ...interface{}) {
	t.Helper()

	count := td.CountWhere(t, query, args...)
	if count != expected {
		t.Errorf("expected %d measurements matching query '%s', got %d", expected, query, count)
	}
}

// AssertMeasurementExists asserts that a measurement with the given ID exists
func (td *TestDatabase) AssertMeasurementExists(t *testing.T, id uint) *Measurement {
	t.Helper()

	m := td.GetMeasurement(t, id)
	if m == nil {
		t.Errorf("expected measurement with ID %d to exist", id)
		return nil
	}
	return m
}

// AssertEmpty asserts that the database is empty
func (td *TestDatabase) AssertEmpty(t *testing.T) {
	t.Helper()

	count := td.Count(t)
	if count != 0 {
		t.Errorf("expected database to be empty, found %d measurements", count)
	}
}

// CreateTestMeasurement creates a single test measurement
func (td *TestDatabase) CreateTestMeasurement(t *testing.T, overrides ...func(*Measurement)) *Measurement {
	t.Helper()

	m := &Measurement{
		Timestamp:    time.Now().UTC(),
		SiteID:       "test-site",
		SiteName:     "Test Site",
		DeviceSN:     "test-device",
		DeviceName:   "Test Device",
		DeviceType:   "solarbank",
		SolarPower:   100.0,
		OutputPower:  50.0,
		GridPower:    10.0,
		BatteryPower: 40.0,
		BatterySoC:   75.0,
	}

	// Apply overrides
	for _, override := range overrides {
		override(m)
	}

	if err := td.Writer.GetDB().Create(m).Error; err != nil {
		t.Fatalf("failed to create test measurement: %s", err)
	}

	return m
}

// CreateTestMeasurements creates multiple test measurements
func (td *TestDatabase) CreateTestMeasurements(t *testing.T, count int, overrides ...func(int, *Measurement)) []Measurement {
	t.Helper()

	measurements := make([]Measurement, count)
	baseTime := time.Now().UTC()

	for i := 0; i < count; i++ {
		measurements[i] = Measurement{
			Timestamp:    baseTime.Add(time.Duration(i) * time.Minute),
			SiteID:       fmt.Sprintf("site-%d", i%3),
			SiteName:     fmt.Sprintf("Site %d", i%3),
			DeviceSN:     fmt.Sprintf("device-%d", i%5),
			DeviceName:   fmt.Sprintf("Device %d", i%5),
			DeviceType:   "solarbank",
			SolarPower:   float64(i * 10),
			OutputPower:  float64(i * 5),
			GridPower:    float64(i * 2),
			BatteryPower: float64(i * 3),
			BatterySoC:   float64(i % 100),
		}

		// Apply overrides
		for _, override := range overrides {
			override(i, &measurements[i])
		}
	}

	if err := td.Writer.GetDB().Create(&measurements).Error; err != nil {
		t.Fatalf("failed to create test measurements: %s", err)
	}

	return measurements
}
