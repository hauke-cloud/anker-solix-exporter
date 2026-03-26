# Migration from InfluxDB to TimescaleDB - Summary

This document summarizes the changes made to migrate the Anker Solix Exporter from InfluxDB to TimescaleDB with GORM and golang-migrate.

## Changes Made

### 1. New Database Package (`internal/database/`)

Created three new files:

- **`models.go`**: Defines the `Measurement` model with GORM tags for TimescaleDB
  - Includes proper indexes for time-series queries
  - Composite index on site_id, device_sn, and timestamp
  - Auto-timestamp handling with BeforeCreate hook

- **`writer.go`**: Database writer implementation
  - Replaces the InfluxDB writer
  - Uses GORM for database operations
  - Batch insert support (100 records per batch)
  - Custom zap logger integration for GORM
  - Connection pooling configuration
  - `WriteMeasurements()` and `GetLastTimestamp()` methods

- **`migrations.go`**: Migration runner
  - Uses golang-migrate for database schema management
  - Automatic migration execution on startup
  - Handles dirty state recovery

### 2. Database Migrations (`migrations/`)

Created migration files with TimescaleDB optimizations:

- **`000001_create_measurements_table.up.sql`**:
  - Creates measurements table with proper data types
  - Converts to TimescaleDB hypertable
  - Sets up compression policy (7 days)
  - Sets up retention policy (2 years)
  - Creates performance indexes

- **`000001_create_measurements_table.down.sql`**:
  - Rollback support for removing the table and policies

- **`README.md`**: Migration documentation

### 3. Configuration Changes

Updated `internal/config/config.go`:

- Replaced `InfluxDBConfig` with `DatabaseConfig`
- New fields:
  - `Host`, `Port`, `User`, `Password`, `Database`
  - `SSLMode` for PostgreSQL connection security
  - `MigrationsPath` for migration files location
- Added `GetDSN()` method to generate PostgreSQL connection string
- Updated validation logic
- Updated environment variable mappings (DB_* instead of INFLUXDB_*)

Updated `config.yaml.example`:
- Removed InfluxDB configuration section
- Added database configuration section with TimescaleDB settings

### 4. Main Application Changes

Updated `cmd/exporter/main.go`:

- Changed import from `internal/influxdb` to `internal/database`
- Initialize database writer with DSN instead of InfluxDB credentials
- Added automatic migration execution on startup
- Updated Exporter struct to use `*database.Writer`
- Updated comments to reflect database usage

### 5. Test Updates

Updated `internal/config/config_test.go`:

- Replaced InfluxDB test configurations with database configurations
- Updated environment variables in tests
- All tests passing

### 6. Docker Configuration

Updated `Dockerfile`:

- Added migration files copy to `/etc/anker-solix-exporter/migrations`

Updated `docker-compose.yml`:

- Replaced `influxdb` service with `timescaledb` service
  - Using `timescale/timescaledb:latest-pg16` image
  - Port 5432 exposed
  - Health check configured
  - Volume for data persistence
- Updated exporter service:
  - Changed dependency to TimescaleDB with health check
  - Updated environment variables (DB_* instead of INFLUXDB_*)
  - Removed InfluxDB specific variables
- Grafana still included but now points to TimescaleDB

### 7. Helm Chart Updates

Updated deployment files in `deployments/helm/anker-solix-exporter/`:

- **`values.yaml`**:
  - Replaced `influxdb` section with `database` section
  - Added database connection parameters
  - Updated example config

- **`values-production.yaml`**:
  - Updated with production-ready database settings
  - Changed sslmode to "require" for production
  - Removed InfluxDB references

- **`templates/deployment.yaml`**:
  - Updated environment variables (DB_* instead of INFLUXDB_*)
  - Changed secret references

- **`templates/secret.yaml`**:
  - Replaced INFLUXDB_TOKEN with DB_PASSWORD
  - Updated conditional logic for database secret

- **`templates/_helpers.tpl`**:
  - Replaced `influxdbSecretName` with `databaseSecretName` helper

### 8. Cleanup

Removed:
- `internal/influxdb/` directory and all its contents
- InfluxDB client dependencies from `go.mod`

## New Dependencies

Added to `go.mod`:
- `gorm.io/gorm` - ORM for database operations
- `gorm.io/driver/postgres` - PostgreSQL driver for GORM
- `github.com/golang-migrate/migrate/v4` - Database migration tool
- Associated PostgreSQL libraries (pgx, lib/pq)

Removed:
- `github.com/influxdata/influxdb-client-go/v2`
- All InfluxDB related dependencies

## Configuration Migration Guide

### Old InfluxDB Configuration
```yaml
influxdb:
  url: "http://influxdb:8086"
  token: "your-token"
  org: "my-org"
  bucket: "solar"
  measurement: "solix_energy"
```

### New TimescaleDB Configuration
```yaml
database:
  host: "localhost"
  port: 5432
  user: "anker_exporter"
  password: "your-password"
  database: "anker_solix"
  sslmode: "disable"
  migrations_path: "/etc/anker-solix-exporter/migrations"
```

### Environment Variables

Old:
- `INFLUXDB_URL`
- `INFLUXDB_TOKEN`
- `INFLUXDB_ORG`
- `INFLUXDB_BUCKET`
- `INFLUXDB_MEASUREMENT`

New:
- `DB_HOST`
- `DB_PORT`
- `DB_USER`
- `DB_PASSWORD`
- `DB_NAME`
- `DB_SSLMODE`
- `DB_MIGRATIONS_PATH`

## Benefits of TimescaleDB Migration

1. **Better SQL Support**: Full PostgreSQL compatibility with standard SQL queries
2. **ACID Compliance**: Strong consistency guarantees
3. **Automatic Compression**: TimescaleDB compresses old data automatically
4. **Retention Policies**: Automatic data cleanup based on age
5. **Better Integration**: Works with standard PostgreSQL tools and ORMs
6. **Cost Effective**: Open-source with no licensing restrictions
7. **Type Safety**: GORM provides compile-time type checking
8. **Migration Support**: Built-in schema versioning with golang-migrate
9. **Enhanced Security**: Support for certificate-based authentication (mTLS)

## Security Features

### Certificate-Based Authentication

Optional support for mutual TLS (mTLS) authentication using X.509 certificates:

- **Client Certificate Authentication**: Authenticate without passwords
- **Server Certificate Verification**: Verify database server identity
- **CA Certificate Support**: Trust chain validation
- **Multiple SSL Modes**: disable, require, verify-ca, verify-full

Configuration:
```yaml
database:
  sslmode: "verify-full"
  sslcert: "/path/to/client-cert.pem"
  sslkey: "/path/to/client-key.pem"
  sslrootcert: "/path/to/ca-cert.pem"
```

See [docs/CERTIFICATE_AUTH.md](docs/CERTIFICATE_AUTH.md) for detailed setup instructions.

## Next Steps

1. Deploy TimescaleDB instance
2. Update configuration with database credentials
3. Deploy the updated exporter (migrations run automatically)
4. Update Grafana datasource to use PostgreSQL/TimescaleDB
5. Optionally migrate historical data from InfluxDB to TimescaleDB

## Testing

All existing tests pass:
```bash
go test ./...
go build ./cmd/exporter
```

The application compiles successfully and is ready for deployment.
