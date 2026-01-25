package acceptance

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestUpcomingBookingCheck_HasUpcoming(t *testing.T) {
	ts := NewTestServer(t)
	defer ts.Close()

	stationID := ts.CreateTestStation(t, "Test Station")
	bikeID := ts.CreateTestBike(t, "BIKE-001", &stationID)

	// Create a booking by user-1 starting in 30 minutes
	bookingStart := time.Now().Add(30 * time.Minute)
	bookingEnd := bookingStart.Add(2 * time.Hour)
	ts.CreateTestBooking(t, bikeID, "user-1", bookingStart.Format(time.RFC3339), bookingEnd.Format(time.RFC3339), false)

	// User-2 checks upcoming bookings
	w := ts.GET("/bikes/"+bikeID+"/upcoming-booking-check", map[string]string{"X-User-ID": "user-2"})

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
	}

	var resp upcomingBookingCheckResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if !resp.HasUpcomingBooking {
		t.Errorf("expected hasUpcomingBooking to be true")
	}

	if resp.NextBookingStart == nil {
		t.Errorf("expected nextBookingStart to be set")
	}

	if resp.MinutesUntilNextBooking == nil {
		t.Errorf("expected minutesUntilNextBooking to be set")
	} else if *resp.MinutesUntilNextBooking < 25 || *resp.MinutesUntilNextBooking > 35 {
		t.Errorf("expected minutesUntilNextBooking to be around 30, got %d", *resp.MinutesUntilNextBooking)
	}
}

func TestUpcomingBookingCheck_NoUpcoming(t *testing.T) {
	ts := NewTestServer(t)
	defer ts.Close()

	stationID := ts.CreateTestStation(t, "Test Station")
	bikeID := ts.CreateTestBike(t, "BIKE-001", &stationID)

	// No bookings exist
	w := ts.GET("/bikes/"+bikeID+"/upcoming-booking-check", map[string]string{"X-User-ID": "user-1"})

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
	}

	var resp upcomingBookingCheckResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if resp.HasUpcomingBooking {
		t.Errorf("expected hasUpcomingBooking to be false")
	}

	if resp.NextBookingStart != nil {
		t.Errorf("expected nextBookingStart to be nil")
	}

	if resp.MinutesUntilNextBooking != nil {
		t.Errorf("expected minutesUntilNextBooking to be nil")
	}
}

func TestUpcomingBookingCheck_IgnoresOwnBooking(t *testing.T) {
	ts := NewTestServer(t)
	defer ts.Close()

	stationID := ts.CreateTestStation(t, "Test Station")
	bikeID := ts.CreateTestBike(t, "BIKE-001", &stationID)

	// Create a booking by user-1 starting in 30 minutes
	bookingStart := time.Now().Add(30 * time.Minute)
	bookingEnd := bookingStart.Add(2 * time.Hour)
	ts.CreateTestBooking(t, bikeID, "user-1", bookingStart.Format(time.RFC3339), bookingEnd.Format(time.RFC3339), false)

	// User-1 (same user) checks upcoming bookings - should not see their own booking
	w := ts.GET("/bikes/"+bikeID+"/upcoming-booking-check", map[string]string{"X-User-ID": "user-1"})

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
	}

	var resp upcomingBookingCheckResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if resp.HasUpcomingBooking {
		t.Errorf("expected hasUpcomingBooking to be false for own booking")
	}
}

func TestUpcomingBookingCheck_IgnoresCancelledBooking(t *testing.T) {
	ts := NewTestServer(t)
	defer ts.Close()

	stationID := ts.CreateTestStation(t, "Test Station")
	bikeID := ts.CreateTestBike(t, "BIKE-001", &stationID)

	// Create a cancelled booking by user-1 starting in 30 minutes
	bookingStart := time.Now().Add(30 * time.Minute)
	bookingEnd := bookingStart.Add(2 * time.Hour)
	ts.CreateTestBooking(t, bikeID, "user-1", bookingStart.Format(time.RFC3339), bookingEnd.Format(time.RFC3339), true) // cancelled

	// User-2 checks upcoming bookings - cancelled booking should be ignored
	w := ts.GET("/bikes/"+bikeID+"/upcoming-booking-check", map[string]string{"X-User-ID": "user-2"})

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
	}

	var resp upcomingBookingCheckResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if resp.HasUpcomingBooking {
		t.Errorf("expected hasUpcomingBooking to be false for cancelled booking")
	}
}

func TestUpcomingBookingCheck_IgnoresBookingMoreThanOneHourAway(t *testing.T) {
	ts := NewTestServer(t)
	defer ts.Close()

	stationID := ts.CreateTestStation(t, "Test Station")
	bikeID := ts.CreateTestBike(t, "BIKE-001", &stationID)

	// Create a booking by user-1 starting in 2 hours (more than 1 hour away)
	bookingStart := time.Now().Add(2 * time.Hour)
	bookingEnd := bookingStart.Add(2 * time.Hour)
	ts.CreateTestBooking(t, bikeID, "user-1", bookingStart.Format(time.RFC3339), bookingEnd.Format(time.RFC3339), false)

	// User-2 checks upcoming bookings - should not see booking more than 1 hour away
	w := ts.GET("/bikes/"+bikeID+"/upcoming-booking-check", map[string]string{"X-User-ID": "user-2"})

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
	}

	var resp upcomingBookingCheckResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if resp.HasUpcomingBooking {
		t.Errorf("expected hasUpcomingBooking to be false for booking more than 1 hour away")
	}
}

func TestUpcomingBookingCheck_BikeNotFound(t *testing.T) {
	ts := NewTestServer(t)
	defer ts.Close()

	w := ts.GET("/bikes/"+uuid.New().String()+"/upcoming-booking-check", map[string]string{"X-User-ID": "user-1"})

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status %d, got %d: %s", http.StatusNotFound, w.Code, w.Body.String())
	}
}

func TestUpcomingBookingCheck_Returns401WithoutAuth(t *testing.T) {
	ts := NewTestServer(t)
	defer ts.Close()

	stationID := ts.CreateTestStation(t, "Test Station")
	bikeID := ts.CreateTestBike(t, "BIKE-001", &stationID)

	w := ts.GET("/bikes/"+bikeID+"/upcoming-booking-check", nil)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d, got %d", http.StatusUnauthorized, w.Code)
	}
}
