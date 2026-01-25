package acceptance

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gin-gonic/gin"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"

	"github.com/semanticallynull/bookingengine-backend/bike"
	"github.com/semanticallynull/bookingengine-backend/booking"
	"github.com/semanticallynull/bookingengine-backend/customer"
	"github.com/semanticallynull/bookingengine-backend/ride"
	"github.com/semanticallynull/bookingengine-backend/station"
)

type TestServer struct {
	DB         *sqlx.DB
	Router     *gin.Engine
	BikeRepo   *bike.Repository
	BookingRepo *booking.Repository
	StationRepo *station.Repository
}

func NewTestServer(t *testing.T) *TestServer {
	t.Helper()

	gin.SetMode(gin.TestMode)

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable"
	}

	db, err := sqlx.Connect("pgx", dbURL)
	if err != nil {
		t.Fatalf("failed to connect to database: %v", err)
	}

	// Clean up test data before each test
	cleanupTestData(t, db)

	br := bike.NewRepository(db)
	sr := station.NewRepository(db)
	cr := customer.NewRepository(db)
	rr := ride.NewRepository(db)
	bkr := booking.NewRepository(db)

	// Create router with test middleware (no real JWT validation)
	r := gin.New()
	r.Use(gin.Recovery())

	// Create API-like handlers but with fake auth
	ts := &TestServer{
		DB:         db,
		Router:     r,
		BikeRepo:   br,
		BookingRepo: bkr,
		StationRepo: sr,
	}

	ts.setupRoutes(br, sr, cr, rr, bkr)

	return ts
}

func (ts *TestServer) setupRoutes(br *bike.Repository, sr *station.Repository, cr *customer.Repository, rr *ride.Repository, bkr *booking.Repository) {
	// Protected routes with fake auth
	protected := ts.Router.Group("/")
	protected.Use(fakeAuthMiddleware())
	{
		protected.GET("/availability", ts.makeAvailabilityHandler(br, bkr))
		protected.GET("/bikes/:id/upcoming-booking-check", ts.makeUpcomingBookingCheckHandler(bkr, br))
		protected.GET("/bookings", ts.makeGetBookingsHandler(bkr, br, sr))
		protected.POST("/bookings", ts.makeCreateBookingHandler(bkr, br, sr))
		protected.GET("/bookings/current", ts.makeGetCurrentBookingHandler(bkr, br, sr))
		protected.POST("/bookings/:bookingId/cancel", ts.makeCancelBookingHandler(bkr, br, sr))
		protected.POST("/ride/start", ts.makeStartRideHandler(bkr, br))
	}
}

func (ts *TestServer) Close() {
	ts.DB.Close()
}

func cleanupTestData(t *testing.T, db *sqlx.DB) {
	t.Helper()

	// Delete in order of dependencies
	_, err := db.Exec("DELETE FROM bookings")
	if err != nil {
		t.Logf("warning: failed to clean bookings: %v", err)
	}
	_, err = db.Exec("DELETE FROM rides")
	if err != nil {
		t.Logf("warning: failed to clean rides: %v", err)
	}
	_, err = db.Exec("DELETE FROM customers")
	if err != nil {
		t.Logf("warning: failed to clean customers: %v", err)
	}
	_, err = db.Exec("DELETE FROM bikes")
	if err != nil {
		t.Logf("warning: failed to clean bikes: %v", err)
	}
	_, err = db.Exec("DELETE FROM stations")
	if err != nil {
		t.Logf("warning: failed to clean stations: %v", err)
	}
}

// fakeAuthMiddleware extracts user ID from X-User-ID header for testing
func fakeAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetHeader("X-User-ID")
		if userID == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"code": "UNAUTHORIZED", "message": "Authentication required"})
			c.Abort()
			return
		}
		c.Set("user_id", userID)
		c.Next()
	}
}

// getUserID gets user ID from context (set by fake auth middleware)
func getUserID(c *gin.Context) (string, bool) {
	userID, exists := c.Get("user_id")
	if !exists {
		return "", false
	}
	return userID.(string), true
}

// Helper methods for making requests
func (ts *TestServer) GET(path string, headers map[string]string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, path, nil)
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	w := httptest.NewRecorder()
	ts.Router.ServeHTTP(w, req)
	return w
}

func (ts *TestServer) POST(path string, body interface{}, headers map[string]string) *httptest.ResponseRecorder {
	var buf bytes.Buffer
	if body != nil {
		json.NewEncoder(&buf).Encode(body)
	}
	req := httptest.NewRequest(http.MethodPost, path, &buf)
	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	w := httptest.NewRecorder()
	ts.Router.ServeHTTP(w, req)
	return w
}

// Helper to create test station
func (ts *TestServer) CreateTestStation(t *testing.T, name string) string {
	t.Helper()
	var id string
	err := ts.DB.Get(&id, `
		INSERT INTO stations (id, name, address, opening_hours, location, type)
		VALUES (gen_random_uuid(), $1, 'Test Address', '9-5', point(0, 0), 'public')
		RETURNING id
	`, name)
	if err != nil {
		t.Fatalf("failed to create test station: %v", err)
	}
	return id
}

// Helper to create test bike
func (ts *TestServer) CreateTestBike(t *testing.T, label string, stationID *string) string {
	t.Helper()
	var id string
	err := ts.DB.Get(&id, `
		INSERT INTO bikes (id, label, imei, location, station_id)
		VALUES (gen_random_uuid(), $1, $2, point(0, 0), $3)
		RETURNING id
	`, label, fmt.Sprintf("IMEI-%s", label), stationID)
	if err != nil {
		t.Fatalf("failed to create test bike: %v", err)
	}
	return id
}

// Helper to create test booking directly in DB
func (ts *TestServer) CreateTestBooking(t *testing.T, bikeID, userID, startTime, endTime string, cancelled bool) string {
	t.Helper()
	var id string

	query := `
		INSERT INTO bookings (id, bike_id, user_id, start_time, end_time, cancelled_at, created_at)
		VALUES (gen_random_uuid(), $1, $2, $3::timestamp with time zone, $4::timestamp with time zone, `

	if cancelled {
		query += `now(), now()) RETURNING id`
	} else {
		query += `NULL, now()) RETURNING id`
	}

	err := ts.DB.Get(&id, query, bikeID, userID, startTime, endTime)
	if err != nil {
		t.Fatalf("failed to create test booking: %v", err)
	}
	return id
}

// SetBookingTimes updates a booking's start/end times directly in DB for testing time-based status
func (ts *TestServer) SetBookingTimes(t *testing.T, bookingID, startTime, endTime string) {
	t.Helper()
	_, err := ts.DB.Exec(`
		UPDATE bookings SET start_time = $2::timestamp with time zone, end_time = $3::timestamp with time zone
		WHERE id = $1
	`, bookingID, startTime, endTime)
	if err != nil {
		t.Fatalf("failed to update booking times: %v", err)
	}
}

// CancelBookingInDB cancels a booking directly in the database
func (ts *TestServer) CancelBookingInDB(t *testing.T, bookingID string) {
	t.Helper()
	_, err := ts.DB.Exec(`UPDATE bookings SET cancelled_at = now() WHERE id = $1`, bookingID)
	if err != nil {
		t.Fatalf("failed to cancel booking: %v", err)
	}
}

var _ = context.Background // Import context for potential future use
