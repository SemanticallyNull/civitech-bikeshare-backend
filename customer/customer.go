package customer

import (
	"database/sql"
	"time"

	"github.com/google/uuid"
)

type Customer struct {
	ID        uuid.UUID
	Auth0ID   string         `db:"auth0_id"`
	StripeID  sql.NullString `db:"stripe_id"`
	Email     sql.NullString `db:"email"`
	Name      sql.NullString `db:"name"`
	CreatedAt time.Time      `db:"created_at"`
}
