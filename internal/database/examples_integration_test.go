package database_test

import (
	"context"
	"testing"
	"time"

	"github.com/anker-solix-exporter/anker-solix-exporter/internal/anker"
	"github.com/anker-solix-exporter/anker-solix-exporter/internal/database"
)

// ExampleAdvancedQueries demonstrates complex queries using the test helpers
func TestAdvancedQueries(t *testing.T) {
	skipIfDockerNotAvailable(t)

	testDB := database.SetupTestDatabase(t)
	defer testDB.Cleanup(t)

	t.Run("QueryByTimeRange", func(t *testing.T) {
		testDB.Reset(t)

		baseTime := time.Now().UTC().Truncate(time.Hour)

		// Create measurements across different time periods
		testDB.CreateTestMeasurements(t, 10, func(i int, m *database.Measurement) {
			m.Timestamp = baseTime.Add(time.Duration(i) * time.Hour)
			m.SiteID = "site1"
			m.DeviceSN = "device1"
		})

		// Query measurements in specific time range
		startTime := baseTime.Add(3 * time.Hour)
		endTime := baseTime.Add(7 * time.Hour)

		measurements := testDB.GetMeasurementsWhere(t,
			"timestamp >= ? AND timestamp <= ?",
			startTime, endTime)

		// Should get measurements at hours 3, 4, 5, 6, 7 = 5 measurements
		if len(measurements) != 5 {
			t.Errorf("expected 5 measurements in time range, got %d", len(measurements))
		}
	})

	t.Run("GroupedDeviceData", func(t *testing.T) {
		testDB.Reset(t)

		// Create measurements for multiple devices
		testDB.CreateTestMeasurement(t, func(m *database.Measurement) {
			m.SiteID = "site1"
			m.DeviceSN = "device-a"
			m.SolarPower = 100
		})
		testDB.CreateTestMeasurement(t, func(m *database.Measurement) {
			m.SiteID = "site1"
			m.DeviceSN = "device-a"
			m.SolarPower = 200
		})
		testDB.CreateTestMeasurement(t, func(m *database.Measurement) {
			m.SiteID = "site1"
			m.DeviceSN = "device-b"
			m.SolarPower = 150
		})

		// Count measurements per device
		countA := testDB.CountWhere(t, "device_sn = ?", "device-a")
		countB := testDB.CountWhere(t, "device_sn = ?", "device-b")

		if countA != 2 {
			t.Errorf("expected 2 measurements for device-a, got %d", countA)
		}
		if countB != 1 {
			t.Errorf("expected 1 measurement for device-b, got %d", countB)
		}
	})

	t.Run("LatestMeasurementPerDevice", func(t *testing.T) {
		testDB.Reset(t)

		baseTime := time.Now().UTC()

		// Device 1: older measurement
		testDB.CreateTestMeasurement(t, func(m *database.Measurement) {
			m.DeviceSN = "device1"
			m.Timestamp = baseTime.Add(-2 * time.Hour)
		})

		// Device 2: newer measurement
		testDB.CreateTestMeasurement(t, func(m *database.Measurement) {
			m.DeviceSN = "device2"
			m.Timestamp = baseTime.Add(-1 * time.Hour)
		})

		// Use the Writer's GetLastTimestamp function
		ctx := context.Background()
		lastTime1, _ := testDB.Writer.GetLastTimestamp(ctx, "test-site", "device1")
		lastTime2, _ := testDB.Writer.GetLastTimestamp(ctx, "test-site", "device2")

		if !lastTime1.Before(lastTime2) {
			t.Error("expected device1 timestamp to be before device2")
		}
	})
}

// TestDataIntegrity demonstrates testing data integrity and constraints
func TestDataIntegrity(t *testing.T) {
	skipIfDockerNotAvailable(t)

	testDB := database.SetupTestDatabase(t)
	defer testDB.Cleanup(t)

	t.Run("RequiredFields", func(t *testing.T) {
		testDB.Reset(t)

		// This should succeed - all required fields provided
		m1 := testDB.CreateTestMeasurement(t)
		testDB.AssertMeasurementExists(t, m1.ID)

		// Verify required fields are set
		if m1.SiteID == "" {
			t.Error("SiteID should not be empty")
		}
		if m1.DeviceSN == "" {
			t.Error("DeviceSN should not be empty")
		}
	})

	t.Run("TimestampPrecision", func(t *testing.T) {
		testDB.Reset(t)

		now := time.Now().UTC()
		m := testDB.CreateTestMeasurement(t, func(m *database.Measurement) {
			m.Timestamp = now
		})

		// Retrieve and verify timestamp
		retrieved := testDB.GetMeasurement(t, m.ID)
		if retrieved == nil {
			t.Fatal("failed to retrieve measurement")
		}

		// Database stores with microsecond precision
		diff := retrieved.Timestamp.Sub(now).Abs()
		if diff > time.Microsecond {
			t.Errorf("timestamp difference too large: %v", diff)
		}
	})
}

// TestConcurrentWrites demonstrates testing concurrent database operations
func TestConcurrentWrites(t *testing.T) {
	skipIfDockerNotAvailable(t)

	testDB := database.SetupTestDatabase(t)
	defer testDB.Cleanup(t)

	t.Run("ParallelInserts", func(t *testing.T) {
		testDB.Reset(t)

		ctx := context.Background()
		done := make(chan bool, 3)

		// Simulate 3 concurrent writers
		for i := 0; i < 3; i++ {
			go func(id int) {
				measurements := make([]anker.Measurement, 10)
				baseTime := time.Now().UTC()

				for j := range measurements {
					measurements[j] = anker.Measurement{
						Timestamp:  baseTime.Add(time.Duration(j) * time.Second),
						SiteID:     "concurrent-test",
						SiteName:   "Test Site",
						DeviceSN:   "device-concurrent",
						DeviceName: "Device",
						DeviceType: "solarbank",
					}
				}

				if err := testDB.Writer.WriteMeasurements(ctx, measurements); err != nil {
					t.Errorf("goroutine %d failed to write: %v", id, err)
				}
				done <- true
			}(i)
		}

		// Wait for all goroutines
		for i := 0; i < 3; i++ {
			<-done
		}

		// Should have 30 measurements total
		testDB.AssertCount(t, 30)
	})
}

// TestResetBehavior verifies that Reset properly cleans the database
func TestResetBehavior(t *testing.T) {
	skipIfDockerNotAvailable(t)

	testDB := database.SetupTestDatabase(t)
	defer testDB.Cleanup(t)

	// Create some data
	testDB.CreateTestMeasurements(t, 100)
	testDB.AssertCount(t, 100)

	// Reset
	testDB.Reset(t)
	testDB.AssertEmpty(t)

	// Create new data - IDs should restart from 1
	m := testDB.CreateTestMeasurement(t)
	if m.ID != 1 {
		t.Errorf("expected ID to be 1 after reset, got %d", m.ID)
	}
}

// TestComplexScenario demonstrates a realistic multi-step test scenario
func TestComplexScenario(t *testing.T) {
	skipIfDockerNotAvailable(t)

	testDB := database.SetupTestDatabase(t)
	defer testDB.Cleanup(t)

	t.Run("MultiSiteMultiDeviceScenario", func(t *testing.T) {
		testDB.Reset(t)

		ctx := context.Background()
		baseTime := time.Now().UTC().Truncate(time.Hour)

		// Scenario: 2 sites, each with 2 devices, collecting data over 24 hours
		sites := []string{"site-home", "site-office"}
		devices := []string{"solarbank-1", "solarbank-2"}

		for _, siteID := range sites {
			for _, deviceSN := range devices {
				// Each device reports every hour for 24 hours
				measurements := make([]anker.Measurement, 24)
				for hour := 0; hour < 24; hour++ {
					measurements[hour] = anker.Measurement{
						Timestamp:    baseTime.Add(time.Duration(hour) * time.Hour),
						SiteID:       siteID,
						SiteName:     "Site " + siteID,
						DeviceSN:     deviceSN,
						DeviceName:   "Device " + deviceSN,
						DeviceType:   "solarbank",
						SolarPower:   float64(hour * 10),
						OutputPower:  float64(hour * 5),
						BatteryPower: float64(hour * 3),
						BatterySoC:   float64((hour * 4) % 100),
					}
				}

				if err := testDB.Writer.WriteMeasurements(ctx, measurements); err != nil {
					t.Fatalf("failed to write measurements: %v", err)
				}
			}
		}

		// Verify total count: 2 sites * 2 devices * 24 hours = 96 measurements
		testDB.AssertCount(t, 96)

		// Verify per-site counts
		testDB.AssertCountWhere(t, 48, "site_id = ?", "site-home")
		testDB.AssertCountWhere(t, 48, "site_id = ?", "site-office")

		// Verify per-device counts
		testDB.AssertCountWhere(t, 48, "device_sn = ?", "solarbank-1")
		testDB.AssertCountWhere(t, 48, "device_sn = ?", "solarbank-2")

		// Verify specific device on specific site
		testDB.AssertCountWhere(t, 24, "site_id = ? AND device_sn = ?", "site-home", "solarbank-1")

		// Get last timestamp for a specific device
		lastTime, err := testDB.Writer.GetLastTimestamp(ctx, "site-home", "solarbank-1")
		if err != nil {
			t.Fatalf("failed to get last timestamp: %v", err)
		}

		expectedLast := baseTime.Add(23 * time.Hour)
		if !lastTime.Equal(expectedLast) {
			t.Errorf("expected last timestamp %v, got %v", expectedLast, lastTime)
		}

		// Query peak solar production (hour 23 should have max)
		measurements := testDB.GetMeasurementsWhere(t,
			"site_id = ? AND device_sn = ? AND solar_power >= ?",
			"site-home", "solarbank-1", 200.0)

		if len(measurements) < 1 {
			t.Error("expected to find measurements with high solar power")
		}
	})
}
