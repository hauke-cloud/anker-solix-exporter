-- Remove retention policy
SELECT remove_retention_policy('measurements', if_exists => TRUE);

-- Remove compression policy
SELECT remove_compression_policy('measurements', if_exists => TRUE);

-- Drop hypertable (this will also drop the table)
DROP TABLE IF EXISTS measurements CASCADE;
