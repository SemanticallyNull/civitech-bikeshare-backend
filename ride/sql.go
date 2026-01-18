package ride

import (
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type Repository struct {
	db *sqlx.DB
}

func NewRepository(db *sqlx.DB) *Repository {
	return &Repository{
		db: db,
	}
}

func (r *Repository) StartRide(bikeID, customerID uuid.UUID) (*Ride, error) {
	var ride *Ride
	err := r.db.Get(&ride, startRideQuery, uuid.New(), bikeID, customerID)
	return ride, err
}

const startRideQuery = `
INSERT INTO rides (id, bike_id, customer_id, started_at)
VALUES ($1, $2, $3, now())
RETURNING id, started_at
`
