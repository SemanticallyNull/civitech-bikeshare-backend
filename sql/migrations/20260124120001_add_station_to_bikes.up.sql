ALTER TABLE bikes ADD COLUMN station_id uuid REFERENCES stations(id);
CREATE INDEX bikes_station_id_idx ON bikes (station_id);
