package station

import (
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

func (r *Repository) GetStations() ([]Station, error) {
	var stations []Station
	err := r.db.Select(&stations, getStations)
	return stations, err
}

const getStations = `SELECT * FROM stations`
