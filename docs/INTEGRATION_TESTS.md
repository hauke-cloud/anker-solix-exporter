# Integration Tests with Testcontainers

This project includes integration tests that use [Testcontainers](https://testcontainers.com/) to test database interactions with a real PostgreSQL instance running in Docker.

## Overview

The integration tests are located in:
- `internal/database/writer_integration_test.go` - Core writer functionality tests
- `internal/database/examples_integration_test.go` - Advanced examples and scenarios
- `internal/database/testing.go` - Reusable test helpers and utilities

These tests verify the database writer functionality by:
- Creating a PostgreSQL container
- Running migrations
- Testing write operations
- Testing query operations
- Verifying indexes and constraints

## Modular Test Helpers

The project includes a comprehensive set of test helpers in `testing.go` that make it easy to write integration tests:

### Setup and Cleanup

```go
// Setup a test database (container + migrations)
testDB := database.SetupTestDatabase(t)
defer testDB.Cleanup(t)

// Reset database between tests
testDB.Reset(t)
```

### Assertion Helpers

```go
// Assert counts
testDB.AssertCount(t, 10)
testDB.AssertCountWhere(t, 5, "site_id = ?", "site1")
testDB.AssertEmpty(t)

// Assert existence
measurement := testDB.AssertMeasurementExists(t, 1)
```

### Data Creation Helpers

```go
// Create single measurement with defaults
m := testDB.CreateTestMeasurement(t)

// Create with custom values
m := testDB.CreateTestMeasurement(t, func(m *database.Measurement) {
    m.SiteID = "custom-site"
    m.SolarPower = 999.9
})

// Create multiple measurements
measurements := testDB.CreateTestMeasurements(t, 100)

// Create multiple with custom values
measurements := testDB.CreateTestMeasurements(t, 50, func(i int, m *database.Measurement) {
    m.SiteID = fmt.Sprintf("site-%d", i)
})
```

### Query Helpers

```go
// Get all measurements
measurements := testDB.GetMeasurements(t)

// Get with conditions
measurements := testDB.GetMeasurementsWhere(t, "site_id = ?", "site1")

// Count records
count := testDB.Count(t)
count := testDB.CountWhere(t, "device_sn = ?", "device1")
```

### Database Helpers

```go
// Check indexes
exists := testDB.IndexExists(t, "idx_measurements_timestamp")
hasIndex := testDB.HasIndexLike(t, "%timestamp%")

// Execute raw SQL
testDB.ExecSQL(t, "UPDATE measurements SET solar_power = 0")
```

## Writing Tests

### Basic Test Pattern

```go
func TestYourFeature(t *testing.T) {
    skipIfDockerNotAvailable(t)
    
    // Setup once
    testDB := database.SetupTestDatabase(t)
    defer testDB.Cleanup(t)
    
    t.Run("Scenario1", func(t *testing.T) {
        testDB.Reset(t) // Clean state
        
        // Your test logic
        m := testDB.CreateTestMeasurement(t)
        testDB.AssertCount(t, 1)
    })
    
    t.Run("Scenario2", func(t *testing.T) {
        testDB.Reset(t) // Clean state again
        
        // Your test logic
        testDB.CreateTestMeasurements(t, 10)
        testDB.AssertCount(t, 10)
    })
}
```

### Container Reuse Pattern

The modular approach allows for efficient container reuse:

```go
func TestMultipleScenarios(t *testing.T) {
    skipIfDockerNotAvailable(t)
    
    // Container starts ONCE
    testDB := database.SetupTestDatabase(t)
    defer testDB.Cleanup(t)
    
    // Multiple subtests reuse the same container
    t.Run("Test1", func(t *testing.T) {
        testDB.Reset(t)
        // ... test logic
    })
    
    t.Run("Test2", func(t *testing.T) {
        testDB.Reset(t)
        // ... test logic
    })
    
    // Container terminates ONCE at the end
}
```

This is much faster than creating a new container for each test!

## Prerequisites

### Local Development

To run integration tests locally, you need:
- Docker installed and running
- Docker socket accessible at `/var/run/docker.sock` or `DOCKER_HOST` environment variable set

### GitHub Actions

The tests run automatically in GitHub Actions on:
- Pull requests to `main` branch
- Pushes to `main` branch
- Tagged releases

GitHub Actions provides Docker by default, so no additional configuration is needed.

## Running Tests

### Run All Tests (including integration tests)

```bash
make test
# or
go test -v ./...
```

### Run Only Unit Tests (skip integration tests)

```bash
make test-unit
# or
go test -v -short ./...
```

### Run Only Integration Tests

```bash
make test-integration
# or
go test -v ./...
```

### Run Specific Integration Test

```bash
go test -v ./internal/database/ -run TestWriterIntegration
```

## Test Behavior

The integration tests will:
- **Skip** when run with `-short` flag (unit test mode)
- **Skip** when Docker is not available
- **Run** when Docker is available and not in short mode

This ensures that:
- Unit tests can run quickly without Docker
- Integration tests run in CI/CD environments
- Developers can run integration tests locally if they have Docker

## GitHub Actions Configuration

The GitHub Actions workflow (`.github/workflows/build.yml`) includes two test jobs:

1. **test**: Runs unit tests with `-short` flag
2. **test-integration**: Runs all tests including integration tests

Both jobs must pass before the build job runs.

## Test Coverage

The integration tests cover:
- Database connection establishment
- Migration execution
- Writing single measurements
- Batch writing measurements
- Querying last timestamp
- Handling empty data
- Model hooks (BeforeCreate)
- Index verification
- Error handling

## Troubleshooting

### Tests Skip Locally

If integration tests skip on your local machine:

1. Check if Docker is running:
   ```bash
   docker info
   ```

2. Check if Docker socket is accessible:
   ```bash
   ls -la /var/run/docker.sock
   ```

3. If using Docker Desktop or remote Docker, set DOCKER_HOST:
   ```bash
   export DOCKER_HOST=unix:///var/run/docker.sock
   ```

### Tests Fail in GitHub Actions

If tests fail in GitHub Actions:

1. Check the workflow logs for specific error messages
2. Verify Docker is available in the runner (it should be by default)
3. Check for network or resource constraints
4. Ensure testcontainers dependencies are up to date

## Adding New Integration Tests

When adding new integration tests:

1. Place them in the appropriate `*_integration_test.go` file
2. Use the `skipIfDockerNotAvailable(t)` helper at the start of the test
3. Clean up containers with `defer testcontainers.TerminateContainer(container)`
4. Use `context.Background()` or a context with timeout
5. Add descriptive test names and subtests

Example:

```go
func TestNewFeatureIntegration(t *testing.T) {
    skipIfDockerNotAvailable(t)
    
    ctx := context.Background()
    
    // Start container
    container, err := postgres.Run(ctx, "postgres:16-alpine", ...)
    if err != nil {
        t.Fatalf("failed to start container: %s", err)
    }
    defer testcontainers.TerminateContainer(container)
    
    // Your test code here
}
```

## Performance Considerations

Integration tests are slower than unit tests because they:
- Pull Docker images (first run only, then cached)
- Start containers
- Initialize databases
- Run migrations

Typical execution time:
- First run: 30-60 seconds (image download)
- Subsequent runs: 5-15 seconds (using cached image)

This is why unit tests (`-short` mode) are recommended for rapid development cycles.
