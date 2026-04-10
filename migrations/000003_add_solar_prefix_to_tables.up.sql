-- Rename sites table to solar_sites
ALTER TABLE sites RENAME TO solar_sites;

-- Rename devices table to solar_devices
ALTER TABLE devices RENAME TO solar_devices;

-- Rename measurements table to solar_measurements
ALTER TABLE measurements RENAME TO solar_measurements;

-- Rename indexes for sites
ALTER INDEX IF EXISTS sites_pkey RENAME TO solar_sites_pkey;
ALTER INDEX IF EXISTS sites_site_id_key RENAME TO solar_sites_site_id_key;

-- Rename indexes for devices
ALTER INDEX IF EXISTS devices_pkey RENAME TO solar_devices_pkey;
ALTER INDEX IF EXISTS idx_devices_site_id RENAME TO idx_solar_devices_site_id;
ALTER INDEX IF EXISTS idx_devices_device_sn RENAME TO idx_solar_devices_device_sn;

-- Rename indexes for measurements
ALTER INDEX IF EXISTS idx_measurements_timestamp RENAME TO idx_solar_measurements_timestamp;
ALTER INDEX IF EXISTS idx_measurements_device_sn RENAME TO idx_solar_measurements_device_sn;

-- Rename sequences
ALTER SEQUENCE IF EXISTS sites_id_seq RENAME TO solar_sites_id_seq;
ALTER SEQUENCE IF EXISTS devices_id_seq RENAME TO solar_devices_id_seq;
ALTER SEQUENCE IF EXISTS measurements_id_seq RENAME TO solar_measurements_id_seq;

-- Rename foreign key constraint
ALTER TABLE solar_measurements DROP CONSTRAINT IF EXISTS measurements_device_sn_fkey;
ALTER TABLE solar_measurements ADD CONSTRAINT solar_measurements_device_sn_fkey 
    FOREIGN KEY (device_sn) REFERENCES solar_devices(device_sn) ON DELETE CASCADE;

-- Rename foreign key constraint for devices
ALTER TABLE solar_devices DROP CONSTRAINT IF EXISTS devices_site_id_fkey;
ALTER TABLE solar_devices ADD CONSTRAINT solar_devices_site_id_fkey 
    FOREIGN KEY (site_id) REFERENCES solar_sites(site_id) ON DELETE CASCADE;
