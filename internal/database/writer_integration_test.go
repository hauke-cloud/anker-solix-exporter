package database_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/anker-solix-exporter/anker-solix-exporter/internal/anker"
	"github.com/anker-solix-exporter/anker-solix-exporter/internal/database"
)

func skipIfDockerNotAvailable(t *testing.T) {
	t.Helper()

	// Check if running in short mode
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Try to access Docker - if we can't, skip the test instead of panicking
	// This prevents test failures when Docker is unavailable or user lacks permissions
	defer func() {
		if r := recover(); r != nil {
			t.Skipf("Docker not available or insufficient permissions: %v", r)
		}
	}()

	// Check if Docker is available via DOCKER_HOST or socket
	dockerHost := os.Getenv("DOCKER_HOST")
	if dockerHost == "" {
		// Check for common Docker socket locations
		if _, err := os.Stat("/var/run/docker.sock"); err != nil {
			t.Skip("Docker not available (no socket found)")
		}
		// Check if we can actually access the socket
		if _, err := os.Open("/var/run/docker.sock"); err != nil {
			t.Skipf("Docker socket exists but not accessible: %v", err)
		}
	}
}

// TestWriterIntegration tests all database writer functionality using a shared container
func TestWriterIntegration(t *testing.T) {
	skipIfDockerNotAvailable(t)

	// Setup test database once for all subtests
	testDB := database.SetupTestDatabase(t)
	defer testDB.Cleanup(t)

	ctx := context.Background()

	// Test WriteMeasurements
	t.Run("WriteMeasurements", func(t *testing.T) {
		testDB.Reset(t) // Reset database state

		measurements := []anker.Measurement{
			{
				Timestamp:    time.Now().UTC(),
				SiteID:       "site1",
				SiteName:     "Test Site",
				DeviceSN:     "device1",
				DeviceName:   "Test Device",
				DeviceType:   "solarbank",
				SolarPower:   100.5,
				OutputPower:  50.2,
				GridPower:    10.1,
				BatteryPower: 40.4,
				BatterySoC:   75.5,
			},
		}

		err := testDB.Writer.WriteMeasurements(ctx, measurements)
		if err != nil {
			t.Fatalf("failed to write measurements: %s", err)
		}

		// Verify data was written using helper
		testDB.AssertCount(t, 1)
	})

	// Test GetLastTimestamp
	t.Run("GetLastTimestamp", func(t *testing.T) {
		testDB.Reset(t) // Reset database state

		now := time.Now().UTC().Truncate(time.Second)
		measurements := []anker.Measurement{
			{
				Timestamp:  now.Add(-2 * time.Hour),
				SiteID:     "site2",
				SiteName:   "Test Site 2",
				DeviceSN:   "device2",
				DeviceName: "Test Device 2",
				DeviceType: "solarbank",
			},
			{
				Timestamp:  now.Add(-1 * time.Hour),
				SiteID:     "site2",
				SiteName:   "Test Site 2",
				DeviceSN:   "device2",
				DeviceName: "Test Device 2",
				DeviceType: "solarbank",
			},
		}

		err := testDB.Writer.WriteMeasurements(ctx, measurements)
		if err != nil {
			t.Fatalf("failed to write measurements: %s", err)
		}

		lastTime, err := testDB.Writer.GetLastTimestamp(ctx, "site2", "device2")
		if err != nil {
			t.Fatalf("failed to get last timestamp: %s", err)
		}

		expected := now.Add(-1 * time.Hour)
		if !lastTime.Truncate(time.Second).Equal(expected) {
			t.Errorf("expected last timestamp %v, got %v", expected, lastTime)
		}
	})

	// Test GetLastTimestamp for non-existent device
	t.Run("GetLastTimestampNotFound", func(t *testing.T) {
		testDB.Reset(t) // Reset database state

		lastTime, err := testDB.Writer.GetLastTimestamp(ctx, "nonexistent", "nonexistent")
		if err != nil {
			t.Fatalf("unexpected error: %s", err)
		}

		if !lastTime.IsZero() {
			t.Errorf("expected zero time for non-existent device, got %v", lastTime)
		}
	})

	// Test batch insert
	t.Run("BatchInsert", func(t *testing.T) {
		testDB.Reset(t) // Reset database state

		measurements := make([]anker.Measurement, 250)
		baseTime := time.Now().UTC()

		for i := range measurements {
			measurements[i] = anker.Measurement{
				Timestamp:    baseTime.Add(time.Duration(i) * time.Minute),
				SiteID:       "site3",
				SiteName:     "Batch Site",
				DeviceSN:     "device3",
				DeviceName:   "Batch Device",
				DeviceType:   "solarbank",
				SolarPower:   float64(i),
				OutputPower:  float64(i * 2),
				GridPower:    float64(i * 3),
				BatteryPower: float64(i * 4),
				BatterySoC:   float64(i % 100),
			}
		}

		err := testDB.Writer.WriteMeasurements(ctx, measurements)
		if err != nil {
			t.Fatalf("failed to batch write measurements: %s", err)
		}

		// Verify all were written using helper - query by device_sn only
		testDB.AssertCountWhere(t, 250, "device_sn = ?", "device3")
	})

	// Test empty measurements slice
	t.Run("EmptyMeasurements", func(t *testing.T) {
		testDB.Reset(t) // Reset database state

		err := testDB.Writer.WriteMeasurements(ctx, []anker.Measurement{})
		if err != nil {
			t.Fatalf("unexpected error with empty measurements: %s", err)
		}

		testDB.AssertEmpty(t)
	})
}

func TestWriterConnectionFailure(t *testing.T) {
	skipIfDockerNotAvailable(t)

	// Try to connect to non-existent database
	testDB := database.SetupTestDatabase(t)
	defer testDB.Cleanup(t)

	// Create a writer with invalid connection (this will fail during NewWriter)
	_, err := database.NewWriter("postgresql://invalid:invalid@localhost:54321/invalid?sslmode=disable", testDB.Logger)
	if err == nil {
		t.Fatal("expected error when connecting to invalid database, got nil")
	}
}

func TestMeasurementModel(t *testing.T) {
	skipIfDockerNotAvailable(t)

	// Setup test database once for all subtests
	testDB := database.SetupTestDatabase(t)
	defer testDB.Cleanup(t)

	// Test table name
	t.Run("TableName", func(t *testing.T) {
		m := database.Measurement{}
		if m.TableName() != "solar_measurements" {
			t.Errorf("expected table name 'solar_measurements', got '%s'", m.TableName())
		}
	})

	// Test BeforeCreate hook
	t.Run("BeforeCreateHook", func(t *testing.T) {
		testDB.Reset(t)

		// Create site and device first
		site := database.Site{SiteID: "site1", SiteName: "Test Site"}
		testDB.Writer.GetDB().Create(&site)

		device := database.Device{SiteID: "site1", DeviceSN: "device1", DeviceName: "Test Device", DeviceType: "solarbank"}
		testDB.Writer.GetDB().Create(&device)

		m := &database.Measurement{
			DeviceSN: "device1",
		}

		// Create without setting Timestamp
		result := testDB.Writer.GetDB().Create(m)
		if result.Error != nil {
			t.Fatalf("failed to create measurement: %s", result.Error)
		}

		// Timestamp should be set automatically
		if m.Timestamp.IsZero() {
			t.Error("expected Timestamp to be set by BeforeCreate hook")
		}
	})

	// Test indexes
	t.Run("Indexes", func(t *testing.T) {
		// Check timestamp index
		if !testDB.HasIndexLike(t, "%timestamp%") {
			t.Error("expected timestamp index to exist")
		}

		// Check device_sn index
		if !testDB.HasIndexLike(t, "%device_sn%") {
			t.Error("expected device_sn index to exist")
		}
	})
}

// TestHelperFunctions demonstrates and tests the helper functions
func TestHelperFunctions(t *testing.T) {
	skipIfDockerNotAvailable(t)

	testDB := database.SetupTestDatabase(t)
	defer testDB.Cleanup(t)

	t.Run("CreateTestMeasurement", func(t *testing.T) {
		testDB.Reset(t)

		// Create a test measurement with defaults
		m1 := testDB.CreateTestMeasurement(t)
		if m1.ID == 0 {
			t.Error("expected ID to be set")
		}

		// Create with overrides
		m2 := testDB.CreateTestMeasurement(t, func(m *database.Measurement) {
			m.DeviceSN = "custom-device"
			m.SolarPower = 999.9
		})

		// Need to create the custom device first
		testDB.Writer.GetDB().Create(&database.Device{SiteID: "test-site", DeviceSN: "custom-device", DeviceName: "Custom Device", DeviceType: "solarbank"})

		if m2.DeviceSN != "custom-device" {
			t.Errorf("expected DeviceSN to be 'custom-device', got '%s'", m2.DeviceSN)
		}
		if m2.SolarPower != 999.9 {
			t.Errorf("expected SolarPower to be 999.9, got %f", m2.SolarPower)
		}

		testDB.AssertCount(t, 2)
	})

	t.Run("CreateTestMeasurements", func(t *testing.T) {
		testDB.Reset(t)

		// Create 10 measurements
		measurements := testDB.CreateTestMeasurements(t, 10)
		if len(measurements) != 10 {
			t.Errorf("expected 10 measurements, got %d", len(measurements))
		}

		testDB.AssertCount(t, 10)

		// Create with overrides
		testDB.Reset(t)
		
		// Create a site and device for the test
		testDB.Writer.GetDB().Create(&database.Site{SiteID: "same-site", SiteName: "Same Site"})
		testDB.Writer.GetDB().Create(&database.Device{SiteID: "same-site", DeviceSN: "same-device", DeviceName: "Same Device", DeviceType: "solarbank"})
		
		testDB.CreateTestMeasurements(t, 5, func(i int, m *database.Measurement) {
			m.DeviceSN = "same-device"
		})

		testDB.AssertCountWhere(t, 5, "device_sn = ?", "same-device")
	})

	t.Run("GetMeasurements", func(t *testing.T) {
		testDB.Reset(t)

		testDB.CreateTestMeasurements(t, 3)
		measurements := testDB.GetMeasurements(t)

		if len(measurements) != 3 {
			t.Errorf("expected 3 measurements, got %d", len(measurements))
		}
	})

	t.Run("GetMeasurementsWhere", func(t *testing.T) {
		testDB.Reset(t)

		// Create sites first
		testDB.Writer.GetDB().Create(&database.Site{SiteID: "site-a", SiteName: "Site A"})
		testDB.Writer.GetDB().Create(&database.Site{SiteID: "site-b", SiteName: "Site B"})

		// Create devices
		testDB.Writer.GetDB().Create(&database.Device{SiteID: "site-a", DeviceSN: "device-a1", DeviceName: "Device A1", DeviceType: "solarbank"})
		testDB.Writer.GetDB().Create(&database.Device{SiteID: "site-b", DeviceSN: "device-b1", DeviceName: "Device B1", DeviceType: "solarbank"})
		testDB.Writer.GetDB().Create(&database.Device{SiteID: "site-a", DeviceSN: "device-a2", DeviceName: "Device A2", DeviceType: "solarbank"})

		testDB.CreateTestMeasurement(t, func(m *database.Measurement) {
			m.DeviceSN = "device-a1"
		})
		testDB.CreateTestMeasurement(t, func(m *database.Measurement) {
			m.DeviceSN = "device-b1"
		})
		testDB.CreateTestMeasurement(t, func(m *database.Measurement) {
			m.DeviceSN = "device-a2"
		})

		// Query by joining with devices table
		var measurements []database.Measurement
		testDB.Writer.GetDB().Joins("JOIN solar_devices ON solar_devices.device_sn = solar_measurements.device_sn").
			Where("solar_devices.site_id = ?", "site-a").
			Find(&measurements)

		if len(measurements) != 2 {
			t.Errorf("expected 2 measurements for site-a, got %d", len(measurements))
		}
	})

	t.Run("Reset", func(t *testing.T) {
		testDB.Reset(t)

		testDB.CreateTestMeasurements(t, 5)
		testDB.AssertCount(t, 5)

		testDB.Reset(t)
		testDB.AssertEmpty(t)
	})
}
