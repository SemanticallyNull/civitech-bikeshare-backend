package acceptance

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
)

// Test GET /bookings

func TestGetBookings_ReturnsUserBookingsSortedByStartTime(t *testing.T) {
	ts := NewTestServer(t)
	defer ts.Close()

	// Create test data
	stationID := ts.CreateTestStation(t, "Test Station")
	bikeID := ts.CreateTestBike(t, "BIKE-001", &stationID)

	userID := "test-user-1"

	// Create bookings with different start times (out of order)
	later := time.Now().Add(48 * time.Hour)
	earlier := time.Now().Add(24 * time.Hour)

	ts.CreateTestBooking(t, bikeID, userID, later.Format(time.RFC3339), later.Add(2*time.Hour).Format(time.RFC3339), false)
	ts.CreateTestBooking(t, bikeID, userID, earlier.Format(time.RFC3339), earlier.Add(2*time.Hour).Format(time.RFC3339), false)

	// Make request
	w := ts.GET("/bookings", map[string]string{"X-User-ID": userID})

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
	}

	var resp []bookingResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if len(resp) != 2 {
		t.Fatalf("expected 2 bookings, got %d", len(resp))
	}

	// Should be sorted by start time ASC
	if resp[0].StartTime.After(resp[1].StartTime) {
		t.Errorf("bookings should be sorted by start time ASC")
	}
}

func TestGetBookings_WithStatusFilter(t *testing.T) {
	ts := NewTestServer(t)
	defer ts.Close()

	// Create test data
	stationID := ts.CreateTestStation(t, "Test Station")
	bikeID := ts.CreateTestBike(t, "BIKE-001", &stationID)

	userID := "test-user-1"

	// Create a confirmed booking (future)
	future := time.Now().Add(24 * time.Hour)
	ts.CreateTestBooking(t, bikeID, userID, future.Format(time.RFC3339), future.Add(2*time.Hour).Format(time.RFC3339), false)

	// Create a cancelled booking
	ts.CreateTestBooking(t, bikeID, userID, future.Add(48*time.Hour).Format(time.RFC3339), future.Add(50*time.Hour).Format(time.RFC3339), true)

	// Request only confirmed bookings
	w := ts.GET("/bookings?status=confirmed", map[string]string{"X-User-ID": userID})

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
	}

	var resp []bookingResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if len(resp) != 1 {
		t.Fatalf("expected 1 confirmed booking, got %d", len(resp))
	}

	if resp[0].Status != "confirmed" {
		t.Errorf("expected status confirmed, got %s", resp[0].Status)
	}
}

func TestGetBookings_Returns401WithoutAuth(t *testing.T) {
	ts := NewTestServer(t)
	defer ts.Close()

	// Make request without auth header
	w := ts.GET("/bookings", nil)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d, got %d", http.StatusUnauthorized, w.Code)
	}
}

// Test POST /bookings

func TestCreateBooking_ReturnsCreatedBooking(t *testing.T) {
	ts := NewTestServer(t)
	defer ts.Close()

	// Create test data
	stationID := ts.CreateTestStation(t, "Test Station")
	bikeID := ts.CreateTestBike(t, "BIKE-001", &stationID)

	userID := "test-user-1"
	startTime := time.Now().Add(24 * time.Hour)
	endTime := startTime.Add(2 * time.Hour)

	body := map[string]string{
		"bikeId":    bikeID,
		"startTime": startTime.Format(time.RFC3339),
		"endTime":   endTime.Format(time.RFC3339),
	}

	w := ts.POST("/bookings", body, map[string]string{"X-User-ID": userID})

	if w.Code != http.StatusCreated {
		t.Errorf("expected status %d, got %d: %s", http.StatusCreated, w.Code, w.Body.String())
	}

	var resp bookingResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if resp.Status != "confirmed" {
		t.Errorf("expected status confirmed, got %s", resp.Status)
	}

	if resp.BikeID.String() != bikeID {
		t.Errorf("expected bikeId %s, got %s", bikeID, resp.BikeID)
	}
}

func TestCreateBooking_InvalidDuration_LessThan1Hour(t *testing.T) {
	ts := NewTestServer(t)
	defer ts.Close()

	stationID := ts.CreateTestStation(t, "Test Station")
	bikeID := ts.CreateTestBike(t, "BIKE-001", &stationID)

	userID := "test-user-1"
	startTime := time.Now().Add(24 * time.Hour)
	endTime := startTime.Add(30 * time.Minute) // Only 30 minutes

	body := map[string]string{
		"bikeId":    bikeID,
		"startTime": startTime.Format(time.RFC3339),
		"endTime":   endTime.Format(time.RFC3339),
	}

	w := ts.POST("/bookings", body, map[string]string{"X-User-ID": userID})

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d: %s", http.StatusBadRequest, w.Code, w.Body.String())
	}

	var resp map[string]string
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["code"] != "INVALID_DURATION" {
		t.Errorf("expected code INVALID_DURATION, got %s", resp["code"])
	}
}

func TestCreateBooking_InvalidDuration_MoreThan24Hours(t *testing.T) {
	ts := NewTestServer(t)
	defer ts.Close()

	stationID := ts.CreateTestStation(t, "Test Station")
	bikeID := ts.CreateTestBike(t, "BIKE-001", &stationID)

	userID := "test-user-1"
	startTime := time.Now().Add(24 * time.Hour)
	endTime := startTime.Add(25 * time.Hour) // 25 hours

	body := map[string]string{
		"bikeId":    bikeID,
		"startTime": startTime.Format(time.RFC3339),
		"endTime":   endTime.Format(time.RFC3339),
	}

	w := ts.POST("/bookings", body, map[string]string{"X-User-ID": userID})

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d: %s", http.StatusBadRequest, w.Code, w.Body.String())
	}

	var resp map[string]string
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["code"] != "INVALID_DURATION" {
		t.Errorf("expected code INVALID_DURATION, got %s", resp["code"])
	}
}

func TestCreateBooking_BookingOverlap(t *testing.T) {
	ts := NewTestServer(t)
	defer ts.Close()

	stationID := ts.CreateTestStation(t, "Test Station")
	bikeID := ts.CreateTestBike(t, "BIKE-001", &stationID)

	startTime := time.Now().Add(24 * time.Hour)
	endTime := startTime.Add(2 * time.Hour)

	// Create existing booking
	ts.CreateTestBooking(t, bikeID, "user-1", startTime.Format(time.RFC3339), endTime.Format(time.RFC3339), false)

	// Try to create overlapping booking
	body := map[string]string{
		"bikeId":    bikeID,
		"startTime": startTime.Add(1 * time.Hour).Format(time.RFC3339), // Overlaps
		"endTime":   startTime.Add(3 * time.Hour).Format(time.RFC3339),
	}

	w := ts.POST("/bookings", body, map[string]string{"X-User-ID": "user-2"})

	if w.Code != http.StatusConflict {
		t.Errorf("expected status %d, got %d: %s", http.StatusConflict, w.Code, w.Body.String())
	}

	var resp map[string]string
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["code"] != "BOOKING_OVERLAP" {
		t.Errorf("expected code BOOKING_OVERLAP, got %s", resp["code"])
	}
}

func TestCreateBooking_BikeNotFound(t *testing.T) {
	ts := NewTestServer(t)
	defer ts.Close()

	userID := "test-user-1"
	startTime := time.Now().Add(24 * time.Hour)
	endTime := startTime.Add(2 * time.Hour)

	body := map[string]string{
		"bikeId":    uuid.New().String(), // Non-existent bike
		"startTime": startTime.Format(time.RFC3339),
		"endTime":   endTime.Format(time.RFC3339),
	}

	w := ts.POST("/bookings", body, map[string]string{"X-User-ID": userID})

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status %d, got %d: %s", http.StatusNotFound, w.Code, w.Body.String())
	}

	var resp map[string]string
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["code"] != "BIKE_NOT_FOUND" {
		t.Errorf("expected code BIKE_NOT_FOUND, got %s", resp["code"])
	}
}

func TestCreateBooking_Returns401WithoutAuth(t *testing.T) {
	ts := NewTestServer(t)
	defer ts.Close()

	body := map[string]string{
		"bikeId":    uuid.New().String(),
		"startTime": time.Now().Add(24 * time.Hour).Format(time.RFC3339),
		"endTime":   time.Now().Add(26 * time.Hour).Format(time.RFC3339),
	}

	w := ts.POST("/bookings", body, nil)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d, got %d", http.StatusUnauthorized, w.Code)
	}
}

// Test GET /bookings/current

func TestGetCurrentBooking_ReturnsActiveBooking(t *testing.T) {
	ts := NewTestServer(t)
	defer ts.Close()

	stationID := ts.CreateTestStation(t, "Test Station")
	bikeID := ts.CreateTestBike(t, "BIKE-001", &stationID)

	userID := "test-user-1"

	// Create a booking that is currently active
	startTime := time.Now().Add(-1 * time.Hour) // Started 1 hour ago
	endTime := time.Now().Add(1 * time.Hour)    // Ends in 1 hour
	ts.CreateTestBooking(t, bikeID, userID, startTime.Format(time.RFC3339), endTime.Format(time.RFC3339), false)

	w := ts.GET("/bookings/current", map[string]string{"X-User-ID": userID})

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
	}

	var resp bookingResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if resp.Status != "active" {
		t.Errorf("expected status active, got %s", resp.Status)
	}
}

func TestGetCurrentBooking_ReturnsNullWhenNoActiveBooking(t *testing.T) {
	ts := NewTestServer(t)
	defer ts.Close()

	stationID := ts.CreateTestStation(t, "Test Station")
	bikeID := ts.CreateTestBike(t, "BIKE-001", &stationID)

	userID := "test-user-1"

	// Create a future booking (not active)
	startTime := time.Now().Add(24 * time.Hour)
	endTime := startTime.Add(2 * time.Hour)
	ts.CreateTestBooking(t, bikeID, userID, startTime.Format(time.RFC3339), endTime.Format(time.RFC3339), false)

	w := ts.GET("/bookings/current", map[string]string{"X-User-ID": userID})

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
	}

	if w.Body.String() != "null" {
		t.Errorf("expected null response, got %s", w.Body.String())
	}
}

func TestGetCurrentBooking_Returns401WithoutAuth(t *testing.T) {
	ts := NewTestServer(t)
	defer ts.Close()

	w := ts.GET("/bookings/current", nil)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d, got %d", http.StatusUnauthorized, w.Code)
	}
}

// Test POST /bookings/:bookingId/cancel

func TestCancelBooking_Success(t *testing.T) {
	ts := NewTestServer(t)
	defer ts.Close()

	stationID := ts.CreateTestStation(t, "Test Station")
	bikeID := ts.CreateTestBike(t, "BIKE-001", &stationID)

	userID := "test-user-1"

	// Create a future booking (can be cancelled)
	startTime := time.Now().Add(24 * time.Hour)
	endTime := startTime.Add(2 * time.Hour)
	bookingID := ts.CreateTestBooking(t, bikeID, userID, startTime.Format(time.RFC3339), endTime.Format(time.RFC3339), false)

	w := ts.POST("/bookings/"+bookingID+"/cancel", nil, map[string]string{"X-User-ID": userID})

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
	}

	var resp bookingResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if resp.Status != "cancelled" {
		t.Errorf("expected status cancelled, got %s", resp.Status)
	}
}

func TestCancelBooking_CannotCancelActiveBooking(t *testing.T) {
	ts := NewTestServer(t)
	defer ts.Close()

	stationID := ts.CreateTestStation(t, "Test Station")
	bikeID := ts.CreateTestBike(t, "BIKE-001", &stationID)

	userID := "test-user-1"

	// Create an active booking (already started)
	startTime := time.Now().Add(-1 * time.Hour)
	endTime := time.Now().Add(1 * time.Hour)
	bookingID := ts.CreateTestBooking(t, bikeID, userID, startTime.Format(time.RFC3339), endTime.Format(time.RFC3339), false)

	w := ts.POST("/bookings/"+bookingID+"/cancel", nil, map[string]string{"X-User-ID": userID})

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d: %s", http.StatusBadRequest, w.Code, w.Body.String())
	}

	var resp map[string]string
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["code"] != "CANNOT_CANCEL" {
		t.Errorf("expected code CANNOT_CANCEL, got %s", resp["code"])
	}
}

func TestCancelBooking_NotAuthorized(t *testing.T) {
	ts := NewTestServer(t)
	defer ts.Close()

	stationID := ts.CreateTestStation(t, "Test Station")
	bikeID := ts.CreateTestBike(t, "BIKE-001", &stationID)

	// Create a booking for user-1
	startTime := time.Now().Add(24 * time.Hour)
	endTime := startTime.Add(2 * time.Hour)
	bookingID := ts.CreateTestBooking(t, bikeID, "user-1", startTime.Format(time.RFC3339), endTime.Format(time.RFC3339), false)

	// Try to cancel as user-2
	w := ts.POST("/bookings/"+bookingID+"/cancel", nil, map[string]string{"X-User-ID": "user-2"})

	if w.Code != http.StatusForbidden {
		t.Errorf("expected status %d, got %d: %s", http.StatusForbidden, w.Code, w.Body.String())
	}

	var resp map[string]string
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["code"] != "NOT_AUTHORIZED" {
		t.Errorf("expected code NOT_AUTHORIZED, got %s", resp["code"])
	}
}

func TestCancelBooking_BookingNotFound(t *testing.T) {
	ts := NewTestServer(t)
	defer ts.Close()

	w := ts.POST("/bookings/"+uuid.New().String()+"/cancel", nil, map[string]string{"X-User-ID": "user-1"})

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status %d, got %d: %s", http.StatusNotFound, w.Code, w.Body.String())
	}

	var resp map[string]string
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["code"] != "BOOKING_NOT_FOUND" {
		t.Errorf("expected code BOOKING_NOT_FOUND, got %s", resp["code"])
	}
}

func TestCancelBooking_Returns401WithoutAuth(t *testing.T) {
	ts := NewTestServer(t)
	defer ts.Close()

	w := ts.POST("/bookings/"+uuid.New().String()+"/cancel", nil, nil)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d, got %d", http.StatusUnauthorized, w.Code)
	}
}

// Test derived status

func TestDerivedStatus_ConfirmedForFutureBooking(t *testing.T) {
	ts := NewTestServer(t)
	defer ts.Close()

	stationID := ts.CreateTestStation(t, "Test Station")
	bikeID := ts.CreateTestBike(t, "BIKE-001", &stationID)

	userID := "test-user-1"

	// Create a future booking
	startTime := time.Now().Add(24 * time.Hour)
	endTime := startTime.Add(2 * time.Hour)
	ts.CreateTestBooking(t, bikeID, userID, startTime.Format(time.RFC3339), endTime.Format(time.RFC3339), false)

	w := ts.GET("/bookings", map[string]string{"X-User-ID": userID})

	var resp []bookingResponse
	json.Unmarshal(w.Body.Bytes(), &resp)

	if len(resp) != 1 {
		t.Fatalf("expected 1 booking, got %d", len(resp))
	}

	if resp[0].Status != "confirmed" {
		t.Errorf("expected status confirmed for future booking, got %s", resp[0].Status)
	}
}

func TestDerivedStatus_ActiveForCurrentBooking(t *testing.T) {
	ts := NewTestServer(t)
	defer ts.Close()

	stationID := ts.CreateTestStation(t, "Test Station")
	bikeID := ts.CreateTestBike(t, "BIKE-001", &stationID)

	userID := "test-user-1"

	// Create a booking that is currently active
	startTime := time.Now().Add(-1 * time.Hour)
	endTime := time.Now().Add(1 * time.Hour)
	ts.CreateTestBooking(t, bikeID, userID, startTime.Format(time.RFC3339), endTime.Format(time.RFC3339), false)

	w := ts.GET("/bookings", map[string]string{"X-User-ID": userID})

	var resp []bookingResponse
	json.Unmarshal(w.Body.Bytes(), &resp)

	if len(resp) != 1 {
		t.Fatalf("expected 1 booking, got %d", len(resp))
	}

	if resp[0].Status != "active" {
		t.Errorf("expected status active for current booking, got %s", resp[0].Status)
	}
}

func TestDerivedStatus_CompletedForPastBooking(t *testing.T) {
	ts := NewTestServer(t)
	defer ts.Close()

	stationID := ts.CreateTestStation(t, "Test Station")
	bikeID := ts.CreateTestBike(t, "BIKE-001", &stationID)

	userID := "test-user-1"

	// Create a past booking
	startTime := time.Now().Add(-3 * time.Hour)
	endTime := time.Now().Add(-1 * time.Hour)
	ts.CreateTestBooking(t, bikeID, userID, startTime.Format(time.RFC3339), endTime.Format(time.RFC3339), false)

	w := ts.GET("/bookings", map[string]string{"X-User-ID": userID})

	var resp []bookingResponse
	json.Unmarshal(w.Body.Bytes(), &resp)

	if len(resp) != 1 {
		t.Fatalf("expected 1 booking, got %d", len(resp))
	}

	if resp[0].Status != "completed" {
		t.Errorf("expected status completed for past booking, got %s", resp[0].Status)
	}
}

func TestDerivedStatus_CancelledOverridesTime(t *testing.T) {
	ts := NewTestServer(t)
	defer ts.Close()

	stationID := ts.CreateTestStation(t, "Test Station")
	bikeID := ts.CreateTestBike(t, "BIKE-001", &stationID)

	userID := "test-user-1"

	// Create a future booking but mark it cancelled
	startTime := time.Now().Add(24 * time.Hour)
	endTime := startTime.Add(2 * time.Hour)
	ts.CreateTestBooking(t, bikeID, userID, startTime.Format(time.RFC3339), endTime.Format(time.RFC3339), true)

	w := ts.GET("/bookings", map[string]string{"X-User-ID": userID})

	var resp []bookingResponse
	json.Unmarshal(w.Body.Bytes(), &resp)

	if len(resp) != 1 {
		t.Fatalf("expected 1 booking, got %d", len(resp))
	}

	if resp[0].Status != "cancelled" {
		t.Errorf("expected status cancelled regardless of time, got %s", resp[0].Status)
	}
}

// Test buffer conflict validation

func TestCreateBooking_BufferConflict(t *testing.T) {
	ts := NewTestServer(t)
	defer ts.Close()

	stationID := ts.CreateTestStation(t, "Test Station")
	bikeID := ts.CreateTestBike(t, "BIKE-001", &stationID)

	// Create an existing booking by user-1 starting in 3 hours
	existingStart := time.Now().Add(3 * time.Hour)
	existingEnd := existingStart.Add(2 * time.Hour)
	ts.CreateTestBooking(t, bikeID, "user-1", existingStart.Format(time.RFC3339), existingEnd.Format(time.RFC3339), false)

	// User-2 tries to create a booking ending 30 minutes before user-1's booking starts
	// This is within the 1-hour buffer and should fail
	newStart := time.Now().Add(1 * time.Hour)
	newEnd := existingStart.Add(-30 * time.Minute) // Ends 30 minutes before existing booking starts

	body := map[string]string{
		"bikeId":    bikeID,
		"startTime": newStart.Format(time.RFC3339),
		"endTime":   newEnd.Format(time.RFC3339),
	}

	w := ts.POST("/bookings", body, map[string]string{"X-User-ID": "user-2"})

	if w.Code != http.StatusConflict {
		t.Errorf("expected status %d, got %d: %s", http.StatusConflict, w.Code, w.Body.String())
	}

	var resp map[string]string
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["code"] != "BUFFER_CONFLICT" {
		t.Errorf("expected code BUFFER_CONFLICT, got %s", resp["code"])
	}
}

func TestCreateBooking_BufferConflict_AllowsOwnBooking(t *testing.T) {
	ts := NewTestServer(t)
	defer ts.Close()

	stationID := ts.CreateTestStation(t, "Test Station")
	bikeID := ts.CreateTestBike(t, "BIKE-001", &stationID)

	// Create an existing booking by user-1 starting in 3 hours
	existingStart := time.Now().Add(3 * time.Hour)
	existingEnd := existingStart.Add(2 * time.Hour)
	ts.CreateTestBooking(t, bikeID, "user-1", existingStart.Format(time.RFC3339), existingEnd.Format(time.RFC3339), false)

	// User-1 (same user) tries to create a booking ending 30 minutes before their own booking starts
	// This should be allowed since it's their own booking
	newStart := time.Now().Add(1 * time.Hour)
	newEnd := existingStart.Add(-30 * time.Minute) // Ends 30 minutes before existing booking starts

	body := map[string]string{
		"bikeId":    bikeID,
		"startTime": newStart.Format(time.RFC3339),
		"endTime":   newEnd.Format(time.RFC3339),
	}

	w := ts.POST("/bookings", body, map[string]string{"X-User-ID": "user-1"})

	if w.Code != http.StatusCreated {
		t.Errorf("expected status %d, got %d: %s", http.StatusCreated, w.Code, w.Body.String())
	}
}

func TestCreateBooking_BufferConflict_ExactlyOneHour(t *testing.T) {
	ts := NewTestServer(t)
	defer ts.Close()

	stationID := ts.CreateTestStation(t, "Test Station")
	bikeID := ts.CreateTestBike(t, "BIKE-001", &stationID)

	// Create an existing booking by user-1 starting in 3 hours
	existingStart := time.Now().Add(3 * time.Hour)
	existingEnd := existingStart.Add(2 * time.Hour)
	ts.CreateTestBooking(t, bikeID, "user-1", existingStart.Format(time.RFC3339), existingEnd.Format(time.RFC3339), false)

	// User-2 creates a booking ending exactly 1 hour before user-1's booking starts
	// This should be allowed (boundary case)
	newStart := time.Now().Add(1 * time.Hour)
	newEnd := existingStart.Add(-1 * time.Hour) // Ends exactly 1 hour before existing booking starts

	body := map[string]string{
		"bikeId":    bikeID,
		"startTime": newStart.Format(time.RFC3339),
		"endTime":   newEnd.Format(time.RFC3339),
	}

	w := ts.POST("/bookings", body, map[string]string{"X-User-ID": "user-2"})

	if w.Code != http.StatusCreated {
		t.Errorf("expected status %d, got %d: %s", http.StatusCreated, w.Code, w.Body.String())
	}
}
