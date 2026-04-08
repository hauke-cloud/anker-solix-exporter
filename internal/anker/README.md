# Anker API Client - Modular Architecture

## Overview

The Anker API client has been refactored from a single 647-line monolithic file into a clean, modular architecture following Go best practices. The code is now organized into focused, single-responsibility modules.

## File Structure

```
internal/anker/
├── auth.go           (1.8K)  - Authentication & session management
├── client.go         (3.0K)  - Core HTTP client & request handling
├── crypto.go         (2.6K)  - Encryption & cryptographic utilities
├── energy.go         (1.6K)  - Historical energy data operations
├── measurements.go   (2.7K)  - Real-time measurement operations
├── sites.go          (3.6K)  - Site & device discovery
├── types.go          (4.0K)  - All data structures & models
├── utils.go          (0.2K)  - Helper functions
└── client_test.go    (2.4K)  - Unit tests
```

## Module Responsibilities

### 1. `types.go` - Data Structures
**Purpose:** Centralized location for all API request/response types and domain models

**Contains:**
- API Request types: `LoginRequest`, `SceneInfoRequest`, `EnergyDataRequest`
- API Response types: `LoginResponse`, `SiteListResponse`, `SceneInfoResponse`, `EnergyDataResponse`
- Domain models: `Site`, `Device`, `PowerData`, `Measurement`

**Benefits:**
- Easy to find and update data structures
- Clear separation between API types and domain models
- Single source of truth for data contracts

### 2. `crypto.go` - Cryptographic Operations
**Purpose:** Handle all encryption and cryptographic operations

**Functions:**
- `generateECDHKeys()` - Generate ECDH key pairs for secure communication
- `encryptPassword()` - Encrypt passwords using AES-256-CBC
- `hashPassword()` - MD5 hash fallback for password encryption
- `pkcs7Pad()` - PKCS7 padding for AES encryption

**Benefits:**
- Security-sensitive code isolated and easy to audit
- Clear cryptographic operations without HTTP logic mixing
- Reusable encryption utilities

### 3. `client.go` - Core HTTP Client
**Purpose:** HTTP communication and request handling

**Responsibilities:**
- Client initialization with `NewClient()`
- HTTP request execution with `doRequest()`
- Header management with `setHeaders()`
- Token generation with `generateGToken()`

**Benefits:**
- Centralized HTTP configuration
- Consistent header handling across all requests
- Easy to add middleware or logging

### 4. `auth.go` - Authentication
**Purpose:** Handle user authentication and session management

**Functions:**
- `Login()` - Authenticate with Anker API and obtain auth token
- `IsAuthenticated()` - Check if client has valid auth token

**Benefits:**
- Authentication logic isolated from other operations
- Easy to add token refresh or session management
- Clear entry point for auth-related changes

### 5. `sites.go` - Site & Device Operations
**Purpose:** Site and device discovery and management

**Functions:**
- `GetSites()` - Retrieve all sites for authenticated user
- `getSceneInfo()` - Fetch detailed device information for a site

**Benefits:**
- Device discovery logic in one place
- Handles complex nested API response structures
- Easy to extend with more site-related operations

### 6. `energy.go` - Historical Data
**Purpose:** Retrieve historical energy data

**Functions:**
- `GetEnergyData()` - Fetch historical power/energy data for devices

**Benefits:**
- Energy data operations isolated
- Easy to add caching or data transformation
- Clear separation from real-time data

### 7. `measurements.go` - Real-time Data
**Purpose:** Collect current/real-time measurements

**Functions:**
- `GetCurrentMeasurements()` - Fetch live power readings from devices

**Benefits:**
- Real-time vs historical data clearly separated
- Easier to optimize polling intervals
- Can add different strategies for live data collection

### 8. `utils.go` - Helper Functions
**Purpose:** Common utility functions

**Functions:**
- `parseFloat()` - Safe string to float64 conversion

**Benefits:**
- Reusable utilities in one place
- Easy to find helper functions
- Can grow as needed without cluttering other files

## Design Principles Applied

### Single Responsibility Principle (SRP)
Each file has one clear purpose:
- `auth.go` only handles authentication
- `crypto.go` only handles encryption
- `energy.go` only handles energy data
- etc.

### Separation of Concerns
- Data structures separated from logic (`types.go`)
- HTTP layer separated from business logic (`client.go` vs domain files)
- Crypto operations isolated from API calls

### Maintainability
- Small, focused files (< 200 lines each)
- Clear naming conventions
- Easy to locate specific functionality
- Reduced cognitive load when reading code

### Testability
- Functions are smaller and more testable
- Mock boundaries are clearer
- Crypto can be tested independently of HTTP
- Business logic tested without network calls

### Extensibility
- Easy to add new endpoints (just add to appropriate file)
- Can add new device types without touching unrelated code
- API changes isolated to relevant modules

## Migration from Old Structure

### Before (Monolithic)
```
client.go (647 lines)
  ├── Constants & Types
  ├── Client struct
  ├── ECDH key generation
  ├── Password encryption
  ├── Login
  ├── GetSites
  ├── getSceneInfo
  ├── GetEnergyData
  ├── GetCurrentMeasurements
  ├── doRequest
  ├── Helper functions
  └── ... everything mixed together
```

### After (Modular)
```
types.go      - All data structures
crypto.go     - All encryption logic
auth.go       - Authentication only
sites.go      - Site operations
energy.go     - Energy data
measurements.go - Real-time data
client.go     - HTTP client core
utils.go      - Helpers
```

## Usage Examples

### Authentication
```go
import "github.com/anker-solix-exporter/anker-solix-exporter/internal/anker"

client := anker.NewClient("user@email.com", "password", "DE")
err := client.Login()
if err != nil {
    log.Fatal(err)
}
```

### Get Sites and Devices
```go
sites, err := client.GetSites()
for _, site := range sites {
    fmt.Printf("Site: %s\n", site.SiteName)
    for _, device := range site.DeviceList {
        fmt.Printf("  Device: %s (%s)\n", device.DeviceName, device.DeviceType)
    }
}
```

### Get Real-time Measurements
```go
measurements, err := client.GetCurrentMeasurements(siteID)
for _, m := range measurements {
    fmt.Printf("%s: %.2f W\n", m.DeviceName, m.SolarPower)
}
```

### Get Historical Data
```go
data, err := client.GetEnergyData(siteID, deviceSN, startTime, endTime)
for _, pd := range data {
    fmt.Printf("%s: %s kWh\n", pd.Time, pd.Value)
}
```

## Benefits of Refactoring

1. **Easier to Navigate** - Find code by feature, not by line number
2. **Faster Development** - Know exactly where to add new features
3. **Better Testing** - Test individual modules without dependencies
4. **Code Review** - Reviewers can focus on specific modules
5. **Onboarding** - New developers can understand code faster
6. **Maintenance** - Bug fixes are easier to locate and test
7. **Documentation** - Each file is self-documenting by its purpose

## Future Enhancements

With this modular structure, we can easily add:

1. **Caching Layer** - Add `cache.go` for response caching
2. **Retry Logic** - Add `retry.go` for intelligent request retries
3. **Rate Limiting** - Add `ratelimit.go` for API rate limiting
4. **Metrics** - Add `metrics.go` for observability
5. **Mock Client** - Add `mock.go` for testing consumers
6. **Batch Operations** - Add `batch.go` for bulk operations
7. **Webhooks** - Add `webhook.go` for event notifications

Each addition would be a new file without touching existing code!

## Testing

All existing tests pass:
```bash
$ go test ./internal/anker -v
=== RUN   TestClientInitialization
--- PASS: TestClientInitialization (0.00s)
=== RUN   TestPasswordEncryption
--- PASS: TestPasswordEncryption (0.00s)
=== RUN   TestLoginRequestFormat
--- PASS: TestLoginRequestFormat (0.00s)
PASS
ok      github.com/anker-solix-exporter/internal/anker  0.006s
```

## Conclusion

The refactored code maintains 100% backward compatibility while providing a much cleaner, more maintainable architecture. The modular design makes the codebase more professional and easier to work with for both current and future development.
