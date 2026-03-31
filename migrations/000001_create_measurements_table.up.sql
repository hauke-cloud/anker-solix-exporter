-- Create measurements table
CREATE TABLE IF NOT EXISTS measurements (
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

-- Create indexes
CREATE INDEX IF NOT EXISTS idx_measurements_timestamp ON measurements(timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_measurements_site_device ON measurements(site_id, device_sn, timestamp DESC);

-- Enable TimescaleDB hypertable
SELECT create_hypertable('measurements', 'timestamp', if_not_exists => TRUE);

-- Set up compression policy (compress data older than 7 days)
ALTER TABLE measurements SET (
    timescaledb.compress,
    timescaledb.compress_segmentby = 'site_id,device_sn',
    timescaledb.compress_orderby = 'timestamp DESC'
);

SELECT add_compression_policy('measurements', INTERVAL '7 days', if_not_exists => TRUE);

-- Set up retention policy (keep data for 2 years)
SELECT add_retention_policy('measurements', INTERVAL '2 years', if_not_exists => TRUE);
