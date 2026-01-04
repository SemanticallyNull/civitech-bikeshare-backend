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

func (r *Repository) GetStation(id string) (Station, error) {
	var station Station
	err := r.db.Get(&station, getStation, id)
	return station, err
}

const getStation = `SELECT * FROM stations WHERE id = $1`
