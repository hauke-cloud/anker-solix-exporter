-- Create old measurements table structure
CREATE TABLE IF NOT EXISTS measurements_old (
    id BIGSERIAL NOT NULL,
    timestamp TIMESTAMPTZ NOT NULL,
    site_id VARCHAR(255) NOT NULL,
    site_name VARCHAR(255) NOT NULL,
    device_sn VARCHAR(255) NOT NULL,
    device_name VARCHAR(255) NOT NULL,
    device_type VARCHAR(100),
    solar_power DOUBLE PRECISION,
    output_power DOUBLE PRECISION,
    grid_power DOUBLE PRECISION,
    battery_power DOUBLE PRECISION,
    battery_soc DOUBLE PRECISION,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (id, timestamp)
);

-- Create indexes for old measurements table
CREATE INDEX IF NOT EXISTS idx_measurements_old_timestamp ON measurements_old(timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_measurements_old_site_device ON measurements_old(site_id, device_sn, timestamp DESC);

-- Enable TimescaleDB hypertable for old measurements table
SELECT create_hypertable('measurements_old', 'timestamp', if_not_exists => TRUE);

-- Migrate data back from normalized structure
INSERT INTO measurements_old (id, timestamp, site_id, site_name, device_sn, device_name, device_type, solar_power, output_power, grid_power, battery_power, battery_soc, created_at)
SELECT m.id, m.timestamp, s.site_id, s.site_name, d.device_sn, d.device_name, d.device_type, m.solar_power, m.output_power, m.grid_power, m.battery_power, m.battery_soc, m.created_at
FROM measurements m
JOIN devices d ON m.device_sn = d.device_sn
JOIN sites s ON d.site_id = s.site_id;

-- Set up compression policy for old measurements table
ALTER TABLE measurements_old SET (
    timescaledb.compress,
    timescaledb.compress_segmentby = 'site_id,device_sn',
    timescaledb.compress_orderby = 'timestamp DESC'
);

SELECT add_compression_policy('measurements_old', INTERVAL '7 days', if_not_exists => TRUE);

-- Set up retention policy for old measurements table
SELECT add_retention_policy('measurements_old', INTERVAL '2 years', if_not_exists => TRUE);

-- Remove retention policy from current measurements table
SELECT remove_retention_policy('measurements', if_exists => TRUE);

-- Remove compression policy from current measurements table
SELECT remove_compression_policy('measurements', if_exists => TRUE);

-- Drop current measurements table
DROP TABLE IF EXISTS measurements CASCADE;

-- Rename old measurements table back
ALTER TABLE measurements_old RENAME TO measurements;

-- Rename indexes
ALTER INDEX idx_measurements_old_timestamp RENAME TO idx_measurements_timestamp;
ALTER INDEX idx_measurements_old_site_device RENAME TO idx_measurements_site_device;

-- Rename sequence to match old table name
ALTER SEQUENCE measurements_old_id_seq RENAME TO measurements_id_seq;

-- Drop normalized tables
DROP TABLE IF EXISTS devices CASCADE;
DROP TABLE IF EXISTS sites CASCADE;
