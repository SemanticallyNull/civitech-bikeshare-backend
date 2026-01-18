package customer

import (
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

func (r *Repository) CreateCustomer(auth0ID string) error {
	_, err := r.db.Exec(createCustomerQuery, uuid.New(), auth0ID)
	return err
}

const createCustomerQuery = "INSERT INTO customers (id, auth0_id) VALUES ($1, $2)"

func (r *Repository) AddStripeIDToCustomer(auth0ID, stripeID string) error {
	_, err := r.db.Exec(addStripeIDToCustomerQuery, stripeID, auth0ID)
	return err
}

const addStripeIDToCustomerQuery = "UPDATE customers SET stripe_id = $1 WHERE auth0_id = $2"
