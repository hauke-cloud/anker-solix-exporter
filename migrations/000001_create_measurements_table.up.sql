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
-- Note: Using DO block to make this idempotent
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM timescaledb_information.hypertables
        WHERE hypertable_name = 'measurements'
    ) THEN
        PERFORM create_hypertable('measurements', 'timestamp');
    END IF;
END $$;
