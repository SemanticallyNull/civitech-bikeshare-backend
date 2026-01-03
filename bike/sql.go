package bike

import (
	"database/sql"
	"errors"

	"github.com/jmoiron/sqlx"
)

var (
	ErrNotFound = errors.New("not found")
)

type Repository struct {
	db *sqlx.DB
}

func NewRepository(db *sqlx.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) GetBikes() ([]Bike, error) {
	var bikes []Bike
	err := r.db.Select(&bikes, getBikes)
	return bikes, err
}

const getBikes = `SELECT * FROM bikes`

func (r *Repository) GetBike(id string) (Bike, error) {
	var bike Bike

	err := r.db.Get(&bike, getBike, id)
	if errors.Is(err, sql.ErrNoRows) {
		return bike, ErrNotFound
	}

	return bike, err
}

const getBike = `SELECT * FROM bikes WHERE label = $1`
