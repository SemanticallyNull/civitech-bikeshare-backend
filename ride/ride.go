package ride

import (
	"database/sql"
	"time"

	"github.com/google/uuid"
)

type Ride struct {
	ID              uuid.UUID
	BikeID          uuid.UUID
	CustomerID      uuid.UUID
	StartedAt       time.Time
	EndedAt         sql.NullTime
	ChargeCreatedAt sql.NullTime
}

