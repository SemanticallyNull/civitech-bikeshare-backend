package ride

import (
	"context"
	"database/sql"
	"errors"

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

var ErrRideInProgress = errors.New("ride in progress")

func (r *Repository) StartRide(ctx context.Context, bikeID, customerID uuid.UUID) (Ride, error) {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return Ride{}, err
	}
	defer tx.Rollback()

	var rides uuid.UUID
	err = tx.Get(&rides, verifyNoRides, bikeID)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return Ride{}, err
	}

	if rides != uuid.Nil {
		return Ride{}, &rideInProgressError{customerID: customerID}
	}

	var ride Ride
	err = r.db.Get(&ride, startRideQuery, uuid.New(), bikeID, customerID)
	if err != nil {
		return Ride{}, err
	}

	err = tx.Commit()
	return ride, err
}

const verifyNoRides = `SELECT customer_id FROM rides WHERE bike_id = $1 AND ended_at IS NULL`

const startRideQuery = `
INSERT INTO rides (id, bike_id, customer_id, started_at)
VALUES ($1, $2, $3, now())
RETURNING *
`

func (r *Repository) EndRide(ctx context.Context, userID uuid.UUID) (int, error) {
	var i int
	err := r.db.GetContext(ctx, &i, endRideQuery, userID)
	return i, err
}

const endRideQuery = `UPDATE rides SET ended_at = now() WHERE customer_id = $1 AND ended_at IS NULL RETURNING ceil(extract(epoch FROM (ended_at - started_at))/60)::int as diff`

type rideInProgressError struct {
	customerID uuid.UUID
}

func (e *rideInProgressError) Error() string {
	return "ride in progress for customer " + e.customerID.String()
}

func CustomerFromRideInProgressError(err error) (uuid.UUID, bool) {
	riperr, ok := err.(*rideInProgressError)
	if ok {
		return riperr.customerID, ok
	}
	return uuid.UUID{}, false
}
