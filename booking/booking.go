package booking

import (
	"database/sql"
	"time"

	"github.com/google/uuid"
)

type BookingStatus string

const (
	StatusConfirmed BookingStatus = "confirmed"
	StatusActive    BookingStatus = "active"
	StatusCompleted BookingStatus = "completed"
	StatusCancelled BookingStatus = "cancelled"
)

type Booking struct {
	ID          uuid.UUID      `db:"id"`
	BikeID      uuid.UUID      `db:"bike_id"`
	BikeLabel   string         `db:"bike_label"`
	BikeName    sql.NullString `db:"bike_name"`
	UserID      string         `db:"user_id"`
	StartTime   time.Time      `db:"start_time"`
	EndTime     time.Time      `db:"end_time"`
	CancelledAt sql.NullTime   `db:"cancelled_at"`
	TotalCost   sql.NullInt32  `db:"total_cost"`
	CreatedAt   time.Time      `db:"created_at"`
}

// Status derives the booking status from the booking's immutable data.
func (b Booking) Status() BookingStatus {
	return b.StatusAt(time.Now())
}

// StatusAt derives the booking status at a given time.
func (b Booking) StatusAt(now time.Time) BookingStatus {
	if b.CancelledAt.Valid {
		return StatusCancelled
	}
	if b.EndTime.Before(now) {
		return StatusCompleted
	}
	if !b.StartTime.After(now) && !b.EndTime.Before(now) {
		return StatusActive
	}
	return StatusConfirmed
}

// BookingTimeSlot represents a booked time slot for availability queries.
type BookingTimeSlot struct {
	StartTime time.Time `db:"start_time"`
	EndTime   time.Time `db:"end_time"`
}
