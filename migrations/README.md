# Database Migrations

This directory contains database migrations for the Anker Solix Exporter TimescaleDB schema.

## Overview

Migrations are automatically run on application startup using golang-migrate. The migrations create and configure a TimescaleDB hypertable for storing energy measurements.

## Migration Files

- `000001_create_measurements_table.up.sql` - Creates the measurements table with TimescaleDB hypertable, indexes, compression, and retention policies
- `000001_create_measurements_table.down.sql` - Removes the measurements table and all policies

## Features

The migration sets up:

1. **Measurements Table**: Stores energy data from Anker Solix devices
2. **TimescaleDB Hypertable**: Optimized for time-series data storage
3. **Indexes**: 
   - Time-based index for efficient range queries
   - Composite index on site_id, device_sn, and timestamp
4. **Compression**: Automatic compression of data older than 7 days
5. **Retention**: Automatic deletion of data older than 2 years

## Manual Migration

If you need to run migrations manually:

```bash
# Set your database connection string
export DATABASE_URL="postgres://user:password@localhost:5432/anker_solix?sslmode=disable"

# Install golang-migrate CLI
go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest

# Run migrations
migrate -database "${DATABASE_URL}" -path migrations up

# Rollback
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
