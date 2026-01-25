// Package bike
package bike

import (
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

// Bike represents a bike which can used as part of a booking.
type Bike struct {
	// ID is an internal identifier for a bike
	ID uuid.UUID
	// Label is a physical label which is on the bike. It should be scannable (e.g. "CARGO-123")
	// in QR Code or Code-128 format.
	Label string
	// IMEI is the identifier of the SIM card used in the bike. This is what is transmitted by the lock
	IMEI string

	Location pgtype.Point

	BatteryVoltage int

	Available bool

	StationID   *uuid.UUID `db:"station_id"`
	StationName *string    `db:"station_name"`

	// DisplayName is a user-friendly name for the bike type (e.g., "Bergamont Cargoville LJ")
	DisplayName *string `db:"display_name"`
	// ImageURL is a URL to an image of the bike
	ImageURL *string `db:"image_url"`
}
