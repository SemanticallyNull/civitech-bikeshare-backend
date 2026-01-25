package booking

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

var (
	ErrNotFound        = errors.New("booking not found")
	ErrOverlap         = errors.New("booking overlaps with existing booking")
	ErrInvalidDuration = errors.New("invalid booking duration")
	ErrCannotCancel    = errors.New("cannot cancel booking that has already started")
	ErrNotAuthorized   = errors.New("not authorized to modify this booking")
)

type Repository struct {
	db *sqlx.DB
}

func NewRepository(db *sqlx.DB) *Repository {
	return &Repository{db: db}
}

// GetByID fetches a single booking by its ID.
func (r *Repository) GetByID(ctx context.Context, id uuid.UUID) (Booking, error) {
	var b Booking
	err := r.db.GetContext(ctx, &b, getByIDQuery, id)
	if errors.Is(err, sql.ErrNoRows) {
		return Booking{}, ErrNotFound
	}
	return b, err
}

const getByIDQuery = `SELECT * FROM bookings WHERE id = $1`

// GetByUserID fetches all bookings for a user, optionally filtered by status.
// Results are sorted by start_time ASC.
func (r *Repository) GetByUserID(ctx context.Context, userID uuid.UUID, status *BookingStatus) ([]Booking, error) {
	var bookings []Booking
	err := r.db.SelectContext(ctx, &bookings, getByUserIDQuery, userID)
	if err != nil {
		return nil, err
	}

	if status == nil {
		return bookings, nil
	}

	// Filter by derived status in Go
	filtered := make([]Booking, 0, len(bookings))
	now := time.Now()
	for _, b := range bookings {
		if b.StatusAt(now) == *status {
			filtered = append(filtered, b)
		}
	}
	return filtered, nil
}

const getByUserIDQuery = `SELECT bk.*, bikes.label as bike_label, bikes.display_name as bike_name FROM bookings bk JOIN bikes 
    ON bk.bike_id = bikes.id WHERE user_id = $1 ORDER BY start_time ASC`

// GetCurrentByUserID fetches the currently active booking for a user.
func (r *Repository) GetCurrentByUserID(ctx context.Context, userID string) (*Booking, error) {
	var b Booking
	err := r.db.GetContext(ctx, &b, getCurrentByUserIDQuery, userID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &b, nil
}

const getCurrentByUserIDQuery = `
SELECT * FROM bookings
WHERE user_id = $1
  AND cancelled_at IS NULL
  AND start_time <= now()
  AND end_time >= now()
`

// Create inserts a new booking after checking for overlaps.
func (r *Repository) Create(ctx context.Context, booking *Booking) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Check for overlapping bookings using FOR UPDATE to prevent race conditions
	var overlappingIDs []uuid.UUID
	err = tx.SelectContext(ctx, &overlappingIDs, checkOverlapQuery, booking.BikeID, booking.StartTime, booking.EndTime)
	if err != nil {
		return err
	}
	if len(overlappingIDs) > 0 {
		return ErrOverlap
	}

	// Insert the booking
	err = tx.GetContext(ctx, booking, createBookingQuery,
		booking.ID, booking.BikeID, booking.UserID, booking.StartTime, booking.EndTime, booking.TotalCost)
	if err != nil {
		return err
	}

	return tx.Commit()
}

const checkOverlapQuery = `
SELECT id FROM bookings
WHERE bike_id = $1
  AND cancelled_at IS NULL
  AND start_time < $3
  AND end_time > $2
FOR UPDATE
`

const createBookingQuery = `
INSERT INTO bookings (id, bike_id, user_id, start_time, end_time, total_cost, created_at)
VALUES ($1, $2, $3, $4, $5, $6, now())
RETURNING *
`

// Cancel sets cancelled_at on a booking after verifying ownership and that it hasn't started.
func (r *Repository) Cancel(ctx context.Context, id uuid.UUID, userID uuid.UUID) (Booking, error) {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return Booking{}, err
	}
	defer tx.Rollback()

	// Fetch the booking with FOR UPDATE
	var b Booking
	err = tx.GetContext(ctx, &b, getBookingForUpdateQuery, id)
	if errors.Is(err, sql.ErrNoRows) {
		return Booking{}, ErrNotFound
	}
	if err != nil {
		return Booking{}, err
	}

	// Check ownership
	if b.UserID != userID {
		return Booking{}, ErrNotAuthorized
	}

	// Check if already cancelled
	if b.CancelledAt.Valid {
		return Booking{}, ErrCannotCancel
	}

	// Check if booking has already started (can only cancel confirmed bookings)
	if !b.StartTime.After(time.Now()) {
		return Booking{}, ErrCannotCancel
	}

	// Set cancelled_at
	err = tx.GetContext(ctx, &b, cancelBookingQuery, id)
	if err != nil {
		return Booking{}, err
	}

	return b, tx.Commit()
}

const getBookingForUpdateQuery = `SELECT * FROM bookings WHERE id = $1 FOR UPDATE`

const cancelBookingQuery = `UPDATE bookings SET cancelled_at = now() WHERE id = $1 RETURNING *`

// GetBookingsForBike fetches non-cancelled booking time slots for a bike within a date range.
func (r *Repository) GetBookingsForBike(ctx context.Context, bikeID uuid.UUID, startDate, endDate *time.Time) ([]BookingTimeSlot, error) {
	var slots []BookingTimeSlot

	if startDate != nil && endDate != nil {
		err := r.db.SelectContext(ctx, &slots, getBookingsForBikeWithRangeQuery, bikeID, *startDate, *endDate)
		return slots, err
	}

	if startDate != nil {
		err := r.db.SelectContext(ctx, &slots, getBookingsForBikeFromStartQuery, bikeID, *startDate)
		return slots, err
	}

	if endDate != nil {
		err := r.db.SelectContext(ctx, &slots, getBookingsForBikeToEndQuery, bikeID, *endDate)
		return slots, err
	}

	err := r.db.SelectContext(ctx, &slots, getBookingsForBikeQuery, bikeID)
	return slots, err
}

const getBookingsForBikeQuery = `
SELECT start_time, end_time, user_id FROM bookings
WHERE bike_id = $1 AND cancelled_at IS NULL
ORDER BY start_time ASC
`

const getBookingsForBikeWithRangeQuery = `
SELECT start_time, end_time, user_id FROM bookings
WHERE bike_id = $1
  AND cancelled_at IS NULL
  AND start_time < $3
  AND end_time > $2
ORDER BY start_time ASC
`

const getBookingsForBikeFromStartQuery = `
SELECT start_time, end_time, user_id FROM bookings
WHERE bike_id = $1
  AND cancelled_at IS NULL
  AND end_time > $2
ORDER BY start_time ASC
`

const getBookingsForBikeToEndQuery = `
SELECT start_time, end_time, user_id FROM bookings
WHERE bike_id = $1
  AND cancelled_at IS NULL
  AND start_time < $2
ORDER BY start_time ASC
`

// GetNextBookingByOtherUser finds the next non-cancelled booking for a bike
// by a different user after the specified time. Returns nil if no such booking exists.
func (r *Repository) GetNextBookingByOtherUser(ctx context.Context, bikeLabel string, userID string, after time.Time) (*Booking, error) {
	var b Booking
	err := r.db.GetContext(ctx, &b, getNextBookingByOtherUserQuery, bikeLabel, userID, after)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &b, nil
}

const getNextBookingByOtherUserQuery = `
SELECT bk.* FROM bookings bk
JOIN bikes ON bikes.label = $1
WHERE user_id != $2
  AND cancelled_at IS NULL
  AND start_time > $3
ORDER BY start_time ASC
LIMIT 1
`
