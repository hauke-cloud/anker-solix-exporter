# Database Migrations

This directory contains database migrations for the Anker Solix Exporter TimescaleDB schema.

## Overview

Migrations are automatically run on application startup using golang-migrate. The migrations create and configure a normalized TimescaleDB schema with separate sites, devices, and measurements tables, all prefixed with "solar_".

## Migration Files

- `000001_create_measurements_table.up.sql` - Creates the initial measurements table with TimescaleDB hypertable, indexes, compression, and retention policies
- `000001_create_measurements_table.down.sql` - Removes the measurements table and all policies
- `000002_normalize_sites_devices.up.sql` - Normalizes the database by adding sites and devices tables, migrating data, and updating the measurements table structure
- `000002_normalize_sites_devices.down.sql` - Reverts to the denormalized structure
- `000003_add_solar_prefix_to_tables.up.sql` - Adds "solar_" prefix to all tables (including schema_migrations), indexes, and sequences
- `000003_add_solar_prefix_to_tables.down.sql` - Removes "solar_" prefix from tables, indexes, and sequences

## Schema Structure

After running all migrations, the database will have the following normalized structure:

### solar_sites Table
Stores information about site locations:
- `id` - Auto-incrementing primary key
- `site_id` - Unique site identifier from Anker API
- `site_name` - Human-readable site name
- `created_at`, `updated_at` - Timestamps

### solar_devices Table
Stores information about devices at sites:
- `id` - Auto-incrementing primary key
- `site_id` - Foreign key to solar_sites table
- `device_sn` - Unique device serial number
- `device_name` - Human-readable device name
- `device_type` - Type of device (e.g., "solarbank", "solar")
- `created_at`, `updated_at` - Timestamps

### solar_measurements Table
Stores time-series energy measurements:
- `id` - Auto-incrementing primary key
- `timestamp` - Measurement timestamp (hypertable partition key)
- `device_sn` - Foreign key to solar_devices table
- `solar_power`, `output_power`, `grid_power`, `battery_power`, `battery_soc` - Energy metrics
- `created_at` - Record creation timestamp

### solar_schema_migrations Table
Tracks which database migrations have been applied (managed by golang-migrate):
- `version` - Migration version number
- `dirty` - Whether the migration is in an incomplete state

## Features

The migration sets up:

1. **Normalized Schema**: Eliminates data redundancy by separating sites, devices, and measurements
2. **TimescaleDB Hypertable**: Optimized for time-series data storage on the measurements table
3. **Foreign Key Constraints**: Maintains referential integrity between tables
4. **Indexes**: 
   - Time-based index on measurements for efficient range queries
   - Device index on measurements for fast device-specific queries
   - Unique constraints on site_id and device_sn
5. **Compression**: Automatic compression of measurement data older than 7 days
6. **Retention**: Automatic deletion of measurement data older than 2 years
7. **Data Migration**: Existing data is automatically migrated to the new normalized structure

## Manual Migration

If you need to run migrations manually:

```bash
# Set your database connection string
export DATABASE_URL="postgres://user:password@localhost:5432/anker_solix?sslmode=disable"

# Install golang-migrate CLI
go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest

# Run migrations
migrate -database "${DATABASE_URL}" -path migrations up

# Rollback one migration
migrate -database "${DATABASE_URL}" -path migrations down 1

# Rollback all migrations
migrate -database "${DATABASE_URL}" -path migrations down
```

## Creating New Migrations

To create a new migration:

```bash
migrate create -ext sql -dir migrations -seq migration_name
```

This will create two files:
- `NNNNNN_migration_name.up.sql` - For applying the migration
- `NNNNNN_migration_name.down.sql` - For rolling back the migration
