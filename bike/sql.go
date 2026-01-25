package bike

import (
	"context"
	"database/sql"
	"errors"

	"github.com/jmoiron/sqlx"
)

var (
	ErrNotFound     = errors.New("not found")
	ErrNotAvailable = errors.New("bike not available")
)

type Repository struct {
	db *sqlx.DB
}

func NewRepository(db *sqlx.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) GetBikes(ctx context.Context) ([]Bike, error) {
	var bikes []Bike
	err := r.db.SelectContext(ctx, &bikes, getBikes)
	return bikes, err
}

const getBikes = `SELECT * FROM bikes`

func (r *Repository) GetBike(ctx context.Context, label string) (Bike, error) {
	var bike Bike

	err := r.db.GetContext(ctx, &bike, getBike, label)
	if errors.Is(err, sql.ErrNoRows) {
		return bike, ErrNotFound
	}

	return bike, err
}

const getBike = `SELECT b.*, COALESCE((bk.start_time >= NOW() AND bk.end_time <= NOW()), true) AS available, s.name as station_name
					FROM bikes b
					LEFT OUTER JOIN bookings bk ON b.id = bk.bike_id
					LEFT OUTER JOIN stations s ON b.station_id = s.id
					WHERE b.label = $1`

// BikeWithStation represents a bike with its station info for availability queries.
type BikeWithStation struct {
	Bike
	StationName string `db:"station_name"`
}

// GetBikesWithStations fetches all bikes with their station info.
func (r *Repository) GetBikesWithStations(ctx context.Context, stationID *string) ([]BikeWithStation, error) {
	var bikes []BikeWithStation
	var err error
	if stationID != nil {
		err = r.db.SelectContext(ctx, &bikes, getBikesWithStationsByStation, *stationID)
	} else {
		err = r.db.SelectContext(ctx, &bikes, getBikesWithStations)
	}
	return bikes, err
}

const getBikesWithStations = `
SELECT b.*, COALESCE(s.name, '') as station_name
FROM bikes b
LEFT JOIN stations s ON b.station_id = s.id
`

const getBikesWithStationsByStation = `
SELECT b.*, COALESCE(s.name, '') as station_name
FROM bikes b
LEFT JOIN stations s ON b.station_id = s.id
WHERE b.station_id = $1
`
