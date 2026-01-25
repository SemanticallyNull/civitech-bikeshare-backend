package acceptance

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"
)

func TestStartRide_RejectsWhenUpcomingBooking(t *testing.T) {
	ts := NewTestServer(t)
	defer ts.Close()

	stationID := ts.CreateTestStation(t, "Test Station")
	bikeID := ts.CreateTestBike(t, "BIKE-001", &stationID)

	// Create a booking by user-1 starting in 30 minutes
	bookingStart := time.Now().Add(30 * time.Minute)
	bookingEnd := bookingStart.Add(2 * time.Hour)
	ts.CreateTestBooking(t, bikeID, "user-1", bookingStart.Format(time.RFC3339), bookingEnd.Format(time.RFC3339), false)

	// User-2 tries to start a ride - should be rejected
	body := map[string]string{"bikeId": bikeID}
	w := ts.POST("/ride/start", body, map[string]string{"X-User-ID": "user-2"})

	if w.Code != http.StatusConflict {
		t.Errorf("expected status %d, got %d: %s", http.StatusConflict, w.Code, w.Body.String())
	}

	var resp map[string]string
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["code"] != "UPCOMING_BOOKING_CONFLICT" {
		t.Errorf("expected code UPCOMING_BOOKING_CONFLICT, got %s", resp["code"])
	}
}

func TestStartRide_AllowsWhenOwnUpcomingBooking(t *testing.T) {
	ts := NewTestServer(t)
	defer ts.Close()

	stationID := ts.CreateTestStation(t, "Test Station")
	bikeID := ts.CreateTestBike(t, "BIKE-001", &stationID)

	// Create a booking by user-1 starting in 30 minutes
	bookingStart := time.Now().Add(30 * time.Minute)
	bookingEnd := bookingStart.Add(2 * time.Hour)
	ts.CreateTestBooking(t, bikeID, "user-1", bookingStart.Format(time.RFC3339), bookingEnd.Format(time.RFC3339), false)

	// User-1 (same user) tries to start a ride - should be allowed
	body := map[string]string{"bikeId": bikeID}
	w := ts.POST("/ride/start", body, map[string]string{"X-User-ID": "user-1"})

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
	}
}

func TestStartRide_AllowsWhenNoUpcomingBooking(t *testing.T) {
	ts := NewTestServer(t)
	defer ts.Close()

	stationID := ts.CreateTestStation(t, "Test Station")
	bikeID := ts.CreateTestBike(t, "BIKE-001", &stationID)

	// No bookings exist - ride should be allowed
	body := map[string]string{"bikeId": bikeID}
	w := ts.POST("/ride/start", body, map[string]string{"X-User-ID": "user-1"})

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
	}
}

func TestStartRide_AllowsWhenBookingMoreThanOneHourAway(t *testing.T) {
	ts := NewTestServer(t)
	defer ts.Close()

	stationID := ts.CreateTestStation(t, "Test Station")
	bikeID := ts.CreateTestBike(t, "BIKE-001", &stationID)

	// Create a booking by user-1 starting in 2 hours (more than 1 hour away)
	bookingStart := time.Now().Add(2 * time.Hour)
	bookingEnd := bookingStart.Add(2 * time.Hour)
	ts.CreateTestBooking(t, bikeID, "user-1", bookingStart.Format(time.RFC3339), bookingEnd.Format(time.RFC3339), false)

	// User-2 tries to start a ride - should be allowed since booking is more than 1 hour away
	body := map[string]string{"bikeId": bikeID}
	w := ts.POST("/ride/start", body, map[string]string{"X-User-ID": "user-2"})

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
	}
}

func TestStartRide_IgnoresCancelledBooking(t *testing.T) {
	ts := NewTestServer(t)
	defer ts.Close()

	stationID := ts.CreateTestStation(t, "Test Station")
	bikeID := ts.CreateTestBike(t, "BIKE-001", &stationID)

	// Create a cancelled booking by user-1 starting in 30 minutes
	bookingStart := time.Now().Add(30 * time.Minute)
	bookingEnd := bookingStart.Add(2 * time.Hour)
	ts.CreateTestBooking(t, bikeID, "user-1", bookingStart.Format(time.RFC3339), bookingEnd.Format(time.RFC3339), true) // cancelled

	// User-2 tries to start a ride - should be allowed since booking is cancelled
	body := map[string]string{"bikeId": bikeID}
	w := ts.POST("/ride/start", body, map[string]string{"X-User-ID": "user-2"})

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
	}
}

func TestStartRide_Returns401WithoutAuth(t *testing.T) {
	ts := NewTestServer(t)
	defer ts.Close()

	stationID := ts.CreateTestStation(t, "Test Station")
	bikeID := ts.CreateTestBike(t, "BIKE-001", &stationID)

	body := map[string]string{"bikeId": bikeID}
	w := ts.POST("/ride/start", body, nil)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d, got %d", http.StatusUnauthorized, w.Code)
	}
}
