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

func (r *Repository) GetBike(ctx context.Context, id string) (Bike, error) {
	var bike Bike

	err := r.db.GetContext(ctx, &bike, getBike, id)
	if errors.Is(err, sql.ErrNoRows) {
		return bike, ErrNotFound
	}

	return bike, err
}

const getBike = `SELECT * FROM bikes WHERE label = $1`

func (r *Repository) ReserveBike(ctx context.Context, id string) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var available bool
	err = tx.GetContext(ctx, &available, reserveBike_checkAvailable, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNotFound
		}
		return err
	}
	if !available {
		return ErrNotAvailable
	}

	_, err = tx.Exec(reserveBike, id)
	if err != nil {
		return err
	}

	return tx.Commit()
}

const reserveBike_checkAvailable = `SELECT available FROM bikes WHERE label = $1 FOR UPDATE`
const reserveBike = `UPDATE bikes SET available = false WHERE label = $1`
