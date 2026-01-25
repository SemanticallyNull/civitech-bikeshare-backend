package acceptance

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"
)

func TestGetAvailability_ReturnsAllBikesWithBookings(t *testing.T) {
	ts := NewTestServer(t)
	defer ts.Close()

	// Create test data
	stationID := ts.CreateTestStation(t, "Test Station")
	bikeID := ts.CreateTestBike(t, "BIKE-001", &stationID)

	// Create a booking for the bike
	startTime := time.Now().Add(24 * time.Hour).Format(time.RFC3339)
	endTime := time.Now().Add(26 * time.Hour).Format(time.RFC3339)
	ts.CreateTestBooking(t, bikeID, "user-1", startTime, endTime, false)

	// Make request with auth header
	w := ts.GET("/availability", map[string]string{"X-User-ID": "user-1"})

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
	}

	var resp []bikeAvailabilityResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if len(resp) != 1 {
		t.Fatalf("expected 1 bike, got %d", len(resp))
	}

	if resp[0].BikeName != "BIKE-001" {
		t.Errorf("expected bike name BIKE-001, got %s", resp[0].BikeName)
	}

	if len(resp[0].Bookings) != 1 {
		t.Errorf("expected 1 booking, got %d", len(resp[0].Bookings))
	}
}

func TestGetAvailability_FilterByStationId(t *testing.T) {
	ts := NewTestServer(t)
	defer ts.Close()

	// Create two stations with bikes
	station1ID := ts.CreateTestStation(t, "Station 1")
	station2ID := ts.CreateTestStation(t, "Station 2")
	ts.CreateTestBike(t, "BIKE-001", &station1ID)
	ts.CreateTestBike(t, "BIKE-002", &station2ID)

	// Request bikes only from station 1
	w := ts.GET("/availability?stationId="+station1ID, map[string]string{"X-User-ID": "user-1"})

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
	}

	var resp []bikeAvailabilityResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if len(resp) != 1 {
		t.Fatalf("expected 1 bike, got %d", len(resp))
	}

	if resp[0].BikeName != "BIKE-001" {
		t.Errorf("expected bike name BIKE-001, got %s", resp[0].BikeName)
	}
}

func TestGetAvailability_FilterByDateRange(t *testing.T) {
	ts := NewTestServer(t)
	defer ts.Close()

	// Create test data
	stationID := ts.CreateTestStation(t, "Test Station")
	bikeID := ts.CreateTestBike(t, "BIKE-001", &stationID)

	// Create a booking for tomorrow
	tomorrowStart := time.Now().Add(24 * time.Hour)
	tomorrowEnd := time.Now().Add(26 * time.Hour)
	ts.CreateTestBooking(t, bikeID, "user-1", tomorrowStart.Format(time.RFC3339), tomorrowEnd.Format(time.RFC3339), false)

	// Create a booking for next week (outside query range)
	nextWeekStart := time.Now().Add(7 * 24 * time.Hour)
	nextWeekEnd := time.Now().Add(7*24*time.Hour + 2*time.Hour)
	ts.CreateTestBooking(t, bikeID, "user-1", nextWeekStart.Format(time.RFC3339), nextWeekEnd.Format(time.RFC3339), false)

	// Query only for tomorrow's date range
	startDate := time.Now().Format(time.RFC3339)
	endDate := time.Now().Add(48 * time.Hour).Format(time.RFC3339)

	w := ts.GET("/availability?startDate="+startDate+"&endDate="+endDate, map[string]string{"X-User-ID": "user-1"})

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
	}

	var resp []bikeAvailabilityResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if len(resp) != 1 {
		t.Fatalf("expected 1 bike, got %d", len(resp))
	}

	// Should only show the booking within the date range (tomorrow's booking)
	if len(resp[0].Bookings) != 1 {
		t.Errorf("expected 1 booking in range, got %d", len(resp[0].Bookings))
	}
}

func TestGetAvailability_ExcludesCancelledBookings(t *testing.T) {
	ts := NewTestServer(t)
	defer ts.Close()

	// Create test data
	stationID := ts.CreateTestStation(t, "Test Station")
	bikeID := ts.CreateTestBike(t, "BIKE-001", &stationID)

	// Create a cancelled booking
	startTime := time.Now().Add(24 * time.Hour).Format(time.RFC3339)
	endTime := time.Now().Add(26 * time.Hour).Format(time.RFC3339)
	ts.CreateTestBooking(t, bikeID, "user-1", startTime, endTime, true) // cancelled = true

	// Make request with auth header
	w := ts.GET("/availability", map[string]string{"X-User-ID": "user-1"})

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
	}

	var resp []bikeAvailabilityResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if len(resp) != 1 {
		t.Fatalf("expected 1 bike, got %d", len(resp))
	}

	// Cancelled bookings should not appear
	if len(resp[0].Bookings) != 0 {
		t.Errorf("expected 0 bookings (cancelled should be excluded), got %d", len(resp[0].Bookings))
	}
}

func TestGetAvailability_Returns401WithoutAuth(t *testing.T) {
	ts := NewTestServer(t)
	defer ts.Close()

	// Make request without auth header
	w := ts.GET("/availability", nil)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d, got %d", http.StatusUnauthorized, w.Code)
	}
}

func TestGetAvailability_ReturnsIsOwnBookingTrue(t *testing.T) {
	ts := NewTestServer(t)
	defer ts.Close()

	// Create test data
	stationID := ts.CreateTestStation(t, "Test Station")
	bikeID := ts.CreateTestBike(t, "BIKE-001", &stationID)

	// Create a booking as user-1
	startTime := time.Now().Add(24 * time.Hour).Format(time.RFC3339)
	endTime := time.Now().Add(26 * time.Hour).Format(time.RFC3339)
	ts.CreateTestBooking(t, bikeID, "user-1", startTime, endTime, false)

	// Request as user-1 (same user who made the booking)
	w := ts.GET("/availability", map[string]string{"X-User-ID": "user-1"})

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
	}

	var resp []bikeAvailabilityResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if len(resp) != 1 {
		t.Fatalf("expected 1 bike, got %d", len(resp))
	}

	if len(resp[0].Bookings) != 1 {
		t.Fatalf("expected 1 booking, got %d", len(resp[0].Bookings))
	}

	if !resp[0].Bookings[0].IsOwnBooking {
		t.Errorf("expected isOwnBooking to be true for user's own booking")
	}
}

func TestGetAvailability_ReturnsIsOwnBookingFalse(t *testing.T) {
	ts := NewTestServer(t)
	defer ts.Close()

	// Create test data
	stationID := ts.CreateTestStation(t, "Test Station")
	bikeID := ts.CreateTestBike(t, "BIKE-001", &stationID)

	// Create a booking as user-1
	startTime := time.Now().Add(24 * time.Hour).Format(time.RFC3339)
	endTime := time.Now().Add(26 * time.Hour).Format(time.RFC3339)
	ts.CreateTestBooking(t, bikeID, "user-1", startTime, endTime, false)

	// Request as user-2 (different user)
	w := ts.GET("/availability", map[string]string{"X-User-ID": "user-2"})

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
	}

	var resp []bikeAvailabilityResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if len(resp) != 1 {
		t.Fatalf("expected 1 bike, got %d", len(resp))
	}

	if len(resp[0].Bookings) != 1 {
		t.Fatalf("expected 1 booking, got %d", len(resp[0].Bookings))
	}

	if resp[0].Bookings[0].IsOwnBooking {
		t.Errorf("expected isOwnBooking to be false for another user's booking")
	}
}
