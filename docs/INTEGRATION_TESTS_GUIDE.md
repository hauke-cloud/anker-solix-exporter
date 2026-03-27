# Example: Adding Integration Tests with Testcontainers

This guide shows how to add new integration tests using the modular test helpers in the Anker Solix Exporter project.

## Quick Start with Test Helpers

The easiest way to write integration tests is using our built-in test helpers from `testing.go`:

```go
package database_test

import (
    "testing"
    "github.com/anker-solix-exporter/anker-solix-exporter/internal/database"
)

func TestMyFeature(t *testing.T) {
    skipIfDockerNotAvailable(t)
    
    // Setup test database (container + migrations)
    testDB := database.SetupTestDatabase(t)
    defer testDB.Cleanup(t)
    
    t.Run("MyScenario", func(t *testing.T) {
        testDB.Reset(t) // Clean state for this test
        
        // Create test data
        m := testDB.CreateTestMeasurement(t, func(m *database.Measurement) {
            m.SiteID = "my-site"
            m.SolarPower = 100.0
        })
        
        // Assert results
        testDB.AssertCount(t, 1)
        testDB.AssertCountWhere(t, 1, "site_id = ?", "my-site")
        
        // Get and verify data
        measurements := testDB.GetMeasurementsWhere(t, "solar_power > ?", 50.0)
        if len(measurements) != 1 {
            t.Errorf("expected 1 measurement with high solar power")
        }
    })
}
```

## Available Test Helpers

### Setup and Cleanup

```go
// Setup: Creates container, runs migrations, returns TestDatabase
testDB := database.SetupTestDatabase(t)

// Cleanup: Closes connections and terminates container
defer testDB.Cleanup(t)

// Reset: Truncates all tables and resets sequences
testDB.Reset(t)
```

### Creating Test Data

```go
// Create single measurement with defaults
m := testDB.CreateTestMeasurement(t)

// Create with overrides
m := testDB.CreateTestMeasurement(t, func(m *database.Measurement) {
    m.SiteID = "custom-site"
    m.DeviceSN = "custom-device"
    m.SolarPower = 999.9
})

// Create multiple measurements
measurements := testDB.CreateTestMeasurements(t, 100)

// Create multiple with custom values per item
measurements := testDB.CreateTestMeasurements(t, 50, 
    func(i int, m *database.Measurement) {
        m.SiteID = fmt.Sprintf("site-%d", i%5)
        m.SolarPower = float64(i * 10)
    })
```

### Querying Data

```go
// Get all measurements
all := testDB.GetMeasurements(t)

// Get single measurement by ID
m := testDB.GetMeasurement(t, 1)

// Get with WHERE clause
filtered := testDB.GetMeasurementsWhere(t, "site_id = ?", "site1")
filtered := testDB.GetMeasurementsWhere(t, 
    "solar_power > ? AND battery_soc < ?", 100.0, 50.0)
```

### Counting Data

```go
// Count all
count := testDB.Count(t)

// Count with WHERE clause
count := testDB.CountWhere(t, "device_sn = ?", "device1")
```

### Assertions

```go
// Assert total count
testDB.AssertCount(t, 10)

// Assert conditional count
testDB.AssertCountWhere(t, 5, "site_id = ?", "site1")

// Assert database is empty
testDB.AssertEmpty(t)

// Assert measurement exists and get it
m := testDB.AssertMeasurementExists(t, 1)
```

### Database Inspection

```go
// Check if index exists
exists := testDB.IndexExists(t, "idx_measurements_timestamp")

// Check if index matching pattern exists
hasIndex := testDB.HasIndexLike(t, "%timestamp%")

// Execute raw SQL
testDB.ExecSQL(t, "UPDATE measurements SET solar_power = 0 WHERE site_id = ?", "site1")
```

## Test Patterns

### Pattern 1: Single Container, Multiple Tests (Recommended)

This is the most efficient pattern - start one container and reuse it:

```go
func TestMultipleScenarios(t *testing.T) {
    skipIfDockerNotAvailable(t)
    
    // Container starts ONCE
    testDB := database.SetupTestDatabase(t)
    defer testDB.Cleanup(t)
    
    t.Run("Scenario1", func(t *testing.T) {
        testDB.Reset(t) // Clean state
        
        testDB.CreateTestMeasurement(t)
        testDB.AssertCount(t, 1)
    })
    
    t.Run("Scenario2", func(t *testing.T) {
        testDB.Reset(t) // Clean state
        
        testDB.CreateTestMeasurements(t, 10)
        testDB.AssertCount(t, 10)
    })
    
    // Container terminates ONCE
}
```

**Benefits:**
- Fast: Container starts once
- Isolated: Each subtest gets clean state via Reset()
- Resource efficient: No container proliferation

### Pattern 2: Testing Complex Queries

```go
func TestComplexQueries(t *testing.T) {
    skipIfDockerNotAvailable(t)
    
    testDB := database.SetupTestDatabase(t)
    defer testDB.Cleanup(t)
    
    t.Run("TimeRangeQuery", func(t *testing.T) {
        testDB.Reset(t)
        
        baseTime := time.Now().UTC()
        
        // Create data across different times
        testDB.CreateTestMeasurements(t, 24, func(i int, m *database.Measurement) {
            m.Timestamp = baseTime.Add(time.Duration(i) * time.Hour)
            m.SiteID = "site1"
        })
        
        // Query specific time range
        measurements := testDB.GetMeasurementsWhere(t,
            "timestamp >= ? AND timestamp <= ?",
            baseTime.Add(5*time.Hour),
            baseTime.Add(10*time.Hour))
        
        if len(measurements) != 6 {
            t.Errorf("expected 6 measurements in range, got %d", len(measurements))
        }
    })
}
```

### Pattern 3: Testing with Real Writer Functions

```go
func TestWriterFunctions(t *testing.T) {
    skipIfDockerNotAvailable(t)
    
    testDB := database.SetupTestDatabase(t)
    defer testDB.Cleanup(t)
    
    ctx := context.Background()
    
    t.Run("GetLastTimestamp", func(t *testing.T) {
        testDB.Reset(t)
        
        now := time.Now().UTC()
        
        // Create measurements using helper
        testDB.CreateTestMeasurement(t, func(m *database.Measurement) {
            m.Timestamp = now.Add(-2 * time.Hour)
            m.SiteID = "site1"
            m.DeviceSN = "device1"
        })
        
        testDB.CreateTestMeasurement(t, func(m *database.Measurement) {
            m.Timestamp = now.Add(-1 * time.Hour)
            m.SiteID = "site1"
            m.DeviceSN = "device1"
        })
        
        // Test the actual Writer function
        lastTime, err := testDB.Writer.GetLastTimestamp(ctx, "site1", "device1")
        if err != nil {
            t.Fatalf("failed to get last timestamp: %v", err)
        }
        
        expected := now.Add(-1 * time.Hour)
        if !lastTime.Truncate(time.Second).Equal(expected.Truncate(time.Second)) {
            t.Errorf("expected %v, got %v", expected, lastTime)
        }
    })
}
```

### Pattern 4: Concurrent Operations

```go
func TestConcurrency(t *testing.T) {
    skipIfDockerNotAvailable(t)
    
    testDB := database.SetupTestDatabase(t)
    defer testDB.Cleanup(t)
    
    t.Run("ParallelWrites", func(t *testing.T) {
        testDB.Reset(t)
        
        ctx := context.Background()
        done := make(chan bool, 5)
        
        // 5 concurrent goroutines writing data
        for i := 0; i < 5; i++ {
            go func(id int) {
                measurements := []anker.Measurement{
                    {
                        Timestamp: time.Now().UTC(),
                        SiteID: fmt.Sprintf("site-%d", id),
                        SiteName: "Test",
                        DeviceSN: "device",
                        DeviceName: "Device",
                        DeviceType: "solarbank",
                    },
                }
                
                if err := testDB.Writer.WriteMeasurements(ctx, measurements); err != nil {
                    t.Errorf("goroutine %d failed: %v", id, err)
                }
                done <- true
            }(i)
        }
        
        // Wait for all
        for i := 0; i < 5; i++ {
            <-done
        }
        
        testDB.AssertCount(t, 5)
    })
}
```

## Using Other Testcontainers Modules

Testcontainers supports many popular services. Here are some examples:

### Redis

```go
import "github.com/testcontainers/testcontainers-go/modules/redis"

redisContainer, err := redis.Run(ctx, "redis:7-alpine")
if err != nil {
    t.Fatalf("failed to start redis: %s", err)
}
defer testcontainers.TerminateContainer(redisContainer)

endpoint, err := redisContainer.Endpoint(ctx, "")
if err != nil {
    t.Fatalf("failed to get endpoint: %s", err)
}
```

### MySQL

```go
import "github.com/testcontainers/testcontainers-go/modules/mysql"

mysqlContainer, err := mysql.Run(ctx,
    "mysql:8",
    mysql.WithDatabase("testdb"),
    mysql.WithUsername("root"),
    mysql.WithPassword("password"),
)
if err != nil {
    t.Fatalf("failed to start mysql: %s", err)
}
defer testcontainers.TerminateContainer(mysqlContainer)
```

### MongoDB

```go
import "github.com/testcontainers/testcontainers-go/modules/mongodb"

mongoContainer, err := mongodb.Run(ctx, "mongo:7")
if err != nil {
    t.Fatalf("failed to start mongodb: %s", err)
}
defer testcontainers.TerminateContainer(mongoContainer)

endpoint, err := mongoContainer.Endpoint(ctx, "mongodb")
if err != nil {
    t.Fatalf("failed to get endpoint: %s", err)
}
```

## Best Practices

### 1. Always Clean Up Containers

```go
defer func() {
    if err := testcontainers.TerminateContainer(container); err != nil {
        t.Logf("failed to terminate container: %s", err)
    }
}()
```

### 2. Use Contexts with Timeouts for Long Operations

```go
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
defer cancel()
```

### 3. Use Subtests for Better Organization

```go
func TestFeature(t *testing.T) {
    // Setup container once
    
    t.Run("Scenario1", func(t *testing.T) {
        // Test scenario 1
    })
    
    t.Run("Scenario2", func(t *testing.T) {
        // Test scenario 2
    })
}
```

### 4. Reuse Containers When Possible

For tests that don't modify state, you can reuse the same container:

```go
func TestMultipleReads(t *testing.T) {
    skipIfDockerNotAvailable(t)
    
    // Start container once
    container := setupContainer(t)
    defer cleanup(container, t)
    
    t.Run("Read1", func(t *testing.T) {
        // Read operation
    })
    
    t.Run("Read2", func(t *testing.T) {
        // Another read operation
    })
}
```

### 5. Use Table-Driven Tests

```go
func TestVariousInputs(t *testing.T) {
    skipIfDockerNotAvailable(t)
    
    container := setupContainer(t)
    defer cleanup(container, t)
    
    tests := []struct {
        name     string
        input    string
        expected string
    }{
        {"case1", "input1", "output1"},
        {"case2", "input2", "output2"},
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Test with tt.input and tt.expected
        })
    }
}
```

### 6. Check Container Logs for Debugging

```go
if t.Failed() {
    logs, err := container.Logs(ctx)
    if err == nil {
        logContent, _ := io.ReadAll(logs)
        t.Logf("Container logs:\n%s", logContent)
    }
}
```

## Running Tests in CI/CD

The tests are automatically run in GitHub Actions. The workflow:

1. **Unit Tests** (`test` job): Run with `-short` flag, skip integration tests
2. **Integration Tests** (`test-integration` job): Run all tests including integration tests
3. **Build** job: Only runs if both test jobs pass

## Troubleshooting

### Container Startup Timeout

If containers take too long to start, increase the timeout:

```go
testcontainers.WithWaitStrategy(
    wait.ForLog("ready").
        WithOccurrence(1).
        WithStartupTimeout(60*time.Second)), // Increased timeout
```

### Port Conflicts

Testcontainers automatically assigns random ports to avoid conflicts. Get the actual port:

```go
port, err := container.MappedPort(ctx, "5432/tcp")
if err != nil {
    t.Fatalf("failed to get mapped port: %s", err)
}
```

### Image Pull Issues in CI

If CI fails to pull images, ensure:
- The image name and tag are correct
- The CI environment has internet access
- The image is publicly accessible

## Resources

- [Testcontainers Documentation](https://testcontainers.com/)
- [Testcontainers Go](https://golang.testcontainers.org/)
- [Available Modules](https://golang.testcontainers.org/modules/)
