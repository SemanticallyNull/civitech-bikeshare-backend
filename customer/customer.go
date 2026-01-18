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
	CreatedAt time.Time      `db:"created_at"`
}
