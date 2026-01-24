package customer

import (
	"context"
	"database/sql"
	"errors"
	"time"

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

var ErrNotFound = errors.New("customer not found")

func (r *Repository) GetCustomerByAuth0ID(auth0ID string) (*Customer, error) {
	var customer Customer
	err := r.db.Get(&customer, getCustomerByAuth0IDQuery, auth0ID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}

		return nil, err
	}
	return &customer, nil
}

const getCustomerByAuth0IDQuery = "SELECT * FROM customers WHERE auth0_id = $1"

func (r *Repository) CreateCustomer(auth0ID string) (*Customer, error) {
	var customer Customer
	err := r.db.Get(&customer, createCustomerQuery, uuid.New(), auth0ID)
	return &customer, err
}

const createCustomerQuery = "INSERT INTO customers (id, auth0_id) VALUES ($1, $2) RETURNING *"

func (r *Repository) AddStripeIDToCustomer(auth0ID, stripeID string) error {
	_, err := r.db.Exec(addStripeIDToCustomerQuery, stripeID, auth0ID)
	return err
}

const addStripeIDToCustomerQuery = "UPDATE customers SET stripe_id = $1 WHERE auth0_id = $2"

var ErrNoRideInProgress = errors.New("no rides in progress")

type CurrentRideResult struct {
	BikeID    string    `db:"label"`
	StartedAt time.Time `db:"started_at"`
}

func (r *Repository) CurrentRide(customerID uuid.UUID) (CurrentRideResult, error) {
	var result CurrentRideResult
	err := r.db.Get(&result, getCurrentRideQuery, customerID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return CurrentRideResult{}, ErrNoRideInProgress
		}
		return CurrentRideResult{}, err
	}
	return result, err
}

const getCurrentRideQuery = `SELECT b.label, r.started_at FROM rides r JOIN bikes b ON bike_id = b.id WHERE r.customer_id = $1
                                                             AND r.ended_at IS NULL`

func (r *Repository) UpdateProfile(ctx context.Context, auth0ID, email, name string) error {
	_, err := r.db.ExecContext(ctx, updateProfileQuery, email, name, auth0ID)
	return err
}

const updateProfileQuery = `UPDATE customers SET email = NULLIF($1, ''), name = NULLIF($2, '') WHERE auth0_id = $3`
