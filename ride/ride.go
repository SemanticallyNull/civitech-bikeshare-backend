package ride

import (
	"database/sql"
	"time"

	"github.com/google/uuid"
)

type Ride struct {
	ID              uuid.UUID     `db:"id"`
	BikeID          uuid.UUID     `db:"bike_id"`
	CustomerID      uuid.UUID     `db:"customer_id"`
	StartedAt       time.Time     `db:"started_at"`
	EndedAt         sql.NullTime  `db:"ended_at"`
	ChargeCreatedAt sql.NullTime  `db:"charge_created_at"`
	LockUserID      sql.NullInt64 `db:"lock_user_id"`
}
