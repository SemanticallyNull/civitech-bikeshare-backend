DROP INDEX IF EXISTS bikes_station_id_idx;
ALTER TABLE bikes DROP COLUMN IF EXISTS station_id;
