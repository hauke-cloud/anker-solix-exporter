-- Rename solar_schema_migrations back to schema_migrations first
-- This must be done BEFORE other changes so the migration system can track the rollback
ALTER TABLE solar_schema_migrations RENAME TO schema_migrations;

-- Rename solar_measurements table back to measurements
ALTER TABLE solar_measurements RENAME TO measurements;

-- Rename solar_devices table back to devices
ALTER TABLE solar_devices RENAME TO devices;

-- Rename solar_sites table back to sites
ALTER TABLE solar_sites RENAME TO sites;

-- Rename indexes for sites
ALTER INDEX IF EXISTS solar_sites_pkey RENAME TO sites_pkey;
ALTER INDEX IF EXISTS solar_sites_site_id_key RENAME TO sites_site_id_key;

-- Rename indexes for devices
ALTER INDEX IF EXISTS solar_devices_pkey RENAME TO devices_pkey;
ALTER INDEX IF EXISTS idx_solar_devices_site_id RENAME TO idx_devices_site_id;
ALTER INDEX IF EXISTS idx_solar_devices_device_sn RENAME TO idx_devices_device_sn;

-- Rename indexes for measurements
ALTER INDEX IF EXISTS idx_solar_measurements_timestamp RENAME TO idx_measurements_timestamp;
ALTER INDEX IF EXISTS idx_solar_measurements_device_sn RENAME TO idx_measurements_device_sn;

-- Rename sequences
ALTER SEQUENCE IF EXISTS solar_sites_id_seq RENAME TO sites_id_seq;
ALTER SEQUENCE IF EXISTS solar_devices_id_seq RENAME TO devices_id_seq;
ALTER SEQUENCE IF EXISTS solar_measurements_id_seq RENAME TO measurements_id_seq;

-- Rename foreign key constraint
ALTER TABLE measurements DROP CONSTRAINT IF EXISTS solar_measurements_device_sn_fkey;
ALTER TABLE measurements ADD CONSTRAINT measurements_device_sn_fkey 
    FOREIGN KEY (device_sn) REFERENCES devices(device_sn) ON DELETE CASCADE;

-- Rename foreign key constraint for devices
ALTER TABLE devices DROP CONSTRAINT IF EXISTS solar_devices_site_id_fkey;
ALTER TABLE devices ADD CONSTRAINT devices_site_id_fkey 
    FOREIGN KEY (site_id) REFERENCES sites(site_id) ON DELETE CASCADE;
