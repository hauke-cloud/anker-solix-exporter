-- Create sites table
CREATE TABLE IF NOT EXISTS sites (
    id BIGSERIAL PRIMARY KEY,
    site_id VARCHAR(255) NOT NULL UNIQUE,
    site_name VARCHAR(255) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Create devices table
CREATE TABLE IF NOT EXISTS devices (
    id BIGSERIAL PRIMARY KEY,
    site_id VARCHAR(255) NOT NULL REFERENCES sites(site_id) ON DELETE CASCADE,
    device_sn VARCHAR(255) NOT NULL UNIQUE,
    device_name VARCHAR(255) NOT NULL,
    device_type VARCHAR(100),
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes for devices
CREATE INDEX IF NOT EXISTS idx_devices_site_id ON devices(site_id);
CREATE INDEX IF NOT EXISTS idx_devices_device_sn ON devices(device_sn);

-- Migrate existing data from measurements to sites table
INSERT INTO sites (site_id, site_name, created_at)
SELECT DISTINCT site_id, site_name, MIN(created_at)
FROM measurements
GROUP BY site_id, site_name
ON CONFLICT (site_id) DO NOTHING;

-- Migrate existing data from measurements to devices table
INSERT INTO devices (site_id, device_sn, device_name, device_type, created_at)
SELECT DISTINCT m.site_id, m.device_sn, m.device_name, m.device_type, MIN(m.created_at)
FROM measurements m
GROUP BY m.site_id, m.device_sn, m.device_name, m.device_type
ON CONFLICT (device_sn) DO NOTHING;

-- Create new normalized measurements table (without foreign key initially)
CREATE TABLE IF NOT EXISTS measurements_new (
    id BIGSERIAL NOT NULL,
    timestamp TIMESTAMPTZ NOT NULL,
    device_sn VARCHAR(255) NOT NULL,
    solar_power DOUBLE PRECISION,
    output_power DOUBLE PRECISION,
    grid_power DOUBLE PRECISION,
    battery_power DOUBLE PRECISION,
    battery_soc DOUBLE PRECISION,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (id, timestamp)
);

-- Create indexes for new measurements table
CREATE INDEX IF NOT EXISTS idx_measurements_new_timestamp ON measurements_new(timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_measurements_new_device_sn ON measurements_new(device_sn, timestamp DESC);

-- Enable TimescaleDB hypertable for new measurements table
SELECT create_hypertable('measurements_new', 'timestamp', if_not_exists := TRUE);

-- Migrate data from old measurements table to new one
INSERT INTO measurements_new (id, timestamp, device_sn, solar_power, output_power, grid_power, battery_power, battery_soc, created_at)
SELECT id, timestamp, device_sn, solar_power, output_power, grid_power, battery_power, battery_soc, created_at
FROM measurements;

-- Set up compression policy for new measurements table
ALTER TABLE measurements_new SET (
    timescaledb.compress,
    timescaledb.compress_segmentby = 'device_sn',
    timescaledb.compress_orderby = 'timestamp DESC'
);

SELECT add_compression_policy('measurements_new', INTERVAL '7 days', if_not_exists := TRUE);

-- Set up retention policy for new measurements table (keep data for 2 years)
SELECT add_retention_policy('measurements_new', INTERVAL '2 years', if_not_exists := TRUE);

-- Drop old measurements table
DROP TABLE IF EXISTS measurements CASCADE;

-- Rename new measurements table to measurements
ALTER TABLE measurements_new RENAME TO measurements;

-- Rename indexes to match new table name
ALTER INDEX idx_measurements_new_timestamp RENAME TO idx_measurements_timestamp;
ALTER INDEX idx_measurements_new_device_sn RENAME TO idx_measurements_device_sn;

-- Rename sequence to match new table name
ALTER SEQUENCE measurements_new_id_seq RENAME TO measurements_id_seq;

-- Add foreign key constraint after renaming (to avoid constraint name issues with hypertable chunks)
ALTER TABLE measurements ADD CONSTRAINT measurements_device_sn_fkey 
    FOREIGN KEY (device_sn) REFERENCES devices(device_sn) ON DELETE CASCADE;
