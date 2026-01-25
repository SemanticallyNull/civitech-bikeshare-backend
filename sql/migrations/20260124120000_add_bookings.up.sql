CREATE TABLE bookings (
    id           uuid                     NOT NULL PRIMARY KEY,
    bike_id      uuid                     NOT NULL REFERENCES bikes(id),
    user_id      text                     NOT NULL,
    start_time   timestamp with time zone NOT NULL,
    end_time     timestamp with time zone NOT NULL,
    cancelled_at timestamp with time zone,
    total_cost   integer,
    created_at   timestamp with time zone NOT NULL DEFAULT now()
);

CREATE INDEX bookings_bike_id_idx ON bookings (bike_id);
CREATE INDEX bookings_user_id_idx ON bookings (user_id);
