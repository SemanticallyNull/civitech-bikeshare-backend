package acceptance

import (
	"database/sql"
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/semanticallynull/bookingengine-backend/bike"
	"github.com/semanticallynull/bookingengine-backend/booking"
	"github.com/semanticallynull/bookingengine-backend/station"
)

type bikeAvailabilityResponse struct {
	BikeID      uuid.UUID                 `json:"bikeId"`
	BikeName    string                    `json:"bikeName"`
	BikeImage   *string                   `json:"bikeImage,omitempty"`
	StationID   *uuid.UUID                `json:"stationId,omitempty"`
	StationName string                    `json:"stationName,omitempty"`
	Bookings    []bookingTimeSlotResponse `json:"bookings"`
}

type bookingTimeSlotResponse struct {
	StartTime time.Time `json:"startTime"`
	EndTime   time.Time `json:"endTime"`
}

type bookingResponse struct {
	ID          uuid.UUID             `json:"id"`
	BikeID      uuid.UUID             `json:"bikeId"`
	BikeName    string                `json:"bikeName"`
	UserID      string                `json:"userId"`
	StationID   *uuid.UUID            `json:"stationId,omitempty"`
	StationName string                `json:"stationName,omitempty"`
	StartTime   time.Time             `json:"startTime"`
	EndTime     time.Time             `json:"endTime"`
	Status      booking.BookingStatus `json:"status"`
	CreatedAt   time.Time             `json:"createdAt"`
	TotalCost   *int32                `json:"totalCost,omitempty"`
}

type createBookingRequest struct {
	BikeID    string `json:"bikeId" binding:"required"`
	StartTime string `json:"startTime" binding:"required"`
	EndTime   string `json:"endTime" binding:"required"`
}

func (ts *TestServer) makeAvailabilityHandler(br *bike.Repository, bkr *booking.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		stationID := c.Query("stationId")
		startDateStr := c.Query("startDate")
		endDateStr := c.Query("endDate")

		var stationIDPtr *string
		if stationID != "" {
			stationIDPtr = &stationID
		}

		var startDate, endDate *time.Time
		if startDateStr != "" {
			t, err := time.Parse(time.RFC3339, startDateStr)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"code": "INVALID_DATE", "message": "Invalid startDate format"})
				return
			}
			startDate = &t
		}
		if endDateStr != "" {
			t, err := time.Parse(time.RFC3339, endDateStr)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"code": "INVALID_DATE", "message": "Invalid endDate format"})
				return
			}
			endDate = &t
		}

		bikes, err := br.GetBikesWithStations(c, stationIDPtr)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
			return
		}

		availability := make([]bikeAvailabilityResponse, 0, len(bikes))
		for _, b := range bikes {
			slots, err := bkr.GetBookingsForBike(c, b.ID, startDate, endDate)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
				return
			}

			bookings := make([]bookingTimeSlotResponse, 0, len(slots))
			for _, slot := range slots {
				bookings = append(bookings, bookingTimeSlotResponse{
					StartTime: slot.StartTime,
					EndTime:   slot.EndTime,
				})
			}

			// Use DisplayName if available, otherwise fall back to Label
			bikeName := b.Label
			if b.DisplayName != nil && *b.DisplayName != "" {
				bikeName = *b.DisplayName
			}

			availability = append(availability, bikeAvailabilityResponse{
				BikeID:      b.ID,
				BikeName:    bikeName,
				BikeImage:   b.ImageURL,
				StationID:   b.StationID,
				StationName: b.StationName,
				Bookings:    bookings,
			})
		}

		c.JSON(http.StatusOK, availability)
	}
}

func (ts *TestServer) makeGetBookingsHandler(bkr *booking.Repository, br *bike.Repository, sr *station.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, ok := getUserID(c)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"code": "UNAUTHORIZED", "message": "Authentication required"})
			return
		}

		statusStr := c.Query("status")
		var statusPtr *booking.BookingStatus
		if statusStr != "" {
			status := booking.BookingStatus(statusStr)
			statusPtr = &status
		}

		bookings, err := bkr.GetByUserID(c, userID, statusPtr)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
			return
		}

		responses := make([]bookingResponse, 0, len(bookings))
		for _, b := range bookings {
			resp := toBookingResponse(c, b, br, sr)
			responses = append(responses, resp)
		}

		c.JSON(http.StatusOK, responses)
	}
}

func (ts *TestServer) makeCreateBookingHandler(bkr *booking.Repository, br *bike.Repository, sr *station.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, ok := getUserID(c)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"code": "UNAUTHORIZED", "message": "Authentication required"})
			return
		}

		var req createBookingRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"code": "INVALID_REQUEST", "message": err.Error()})
			return
		}

		startTime, err := time.Parse(time.RFC3339, req.StartTime)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"code": "INVALID_REQUEST", "message": "Invalid startTime format"})
			return
		}
		endTime, err := time.Parse(time.RFC3339, req.EndTime)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"code": "INVALID_REQUEST", "message": "Invalid endTime format"})
			return
		}

		duration := endTime.Sub(startTime)
		if duration < time.Hour {
			c.JSON(http.StatusBadRequest, gin.H{"code": "INVALID_DURATION", "message": "Booking duration must be at least 1 hour"})
			return
		}
		if duration > 24*time.Hour {
			c.JSON(http.StatusBadRequest, gin.H{"code": "INVALID_DURATION", "message": "Booking duration cannot exceed 24 hours"})
			return
		}

		bikeID, err := uuid.Parse(req.BikeID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"code": "INVALID_REQUEST", "message": "Invalid bikeId"})
			return
		}

		_, err = br.GetBikeByID(c, req.BikeID)
		if err != nil {
			if errors.Is(err, bike.ErrNotFound) {
				c.JSON(http.StatusNotFound, gin.H{"code": "BIKE_NOT_FOUND", "message": "Bike not found"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
			return
		}

		b := &booking.Booking{
			ID:        uuid.New(),
			BikeID:    bikeID,
			UserID:    userID,
			StartTime: startTime,
			EndTime:   endTime,
		}

		err = bkr.Create(c, b)
		if err != nil {
			if errors.Is(err, booking.ErrOverlap) {
				c.JSON(http.StatusConflict, gin.H{"code": "BOOKING_OVERLAP", "message": "Booking overlaps with existing booking"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
			return
		}

		resp := toBookingResponse(c, *b, br, sr)
		c.JSON(http.StatusCreated, resp)
	}
}

func (ts *TestServer) makeGetCurrentBookingHandler(bkr *booking.Repository, br *bike.Repository, sr *station.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, ok := getUserID(c)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"code": "UNAUTHORIZED", "message": "Authentication required"})
			return
		}

		b, err := bkr.GetCurrentByUserID(c, userID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
			return
		}

		if b == nil {
			c.JSON(http.StatusOK, nil)
			return
		}

		resp := toBookingResponse(c, *b, br, sr)
		c.JSON(http.StatusOK, resp)
	}
}

func (ts *TestServer) makeCancelBookingHandler(bkr *booking.Repository, br *bike.Repository, sr *station.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, ok := getUserID(c)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"code": "UNAUTHORIZED", "message": "Authentication required"})
			return
		}

		bookingID, err := uuid.Parse(c.Param("bookingId"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"code": "INVALID_REQUEST", "message": "Invalid bookingId"})
			return
		}

		b, err := bkr.Cancel(c, bookingID, userID)
		if err != nil {
			if errors.Is(err, booking.ErrNotFound) {
				c.JSON(http.StatusNotFound, gin.H{"code": "BOOKING_NOT_FOUND", "message": "Booking not found"})
				return
			}
			if errors.Is(err, booking.ErrNotAuthorized) {
				c.JSON(http.StatusForbidden, gin.H{"code": "NOT_AUTHORIZED", "message": "Not authorized to cancel this booking"})
				return
			}
			if errors.Is(err, booking.ErrCannotCancel) {
				c.JSON(http.StatusBadRequest, gin.H{"code": "CANNOT_CANCEL", "message": "Cannot cancel booking that has already started"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
			return
		}

		resp := toBookingResponse(c, b, br, sr)
		c.JSON(http.StatusOK, resp)
	}
}

func toBookingResponse(c *gin.Context, b booking.Booking, br *bike.Repository, sr *station.Repository) bookingResponse {
	bikeInfo, err := br.GetBikeByID(c, b.BikeID.String())

	var bikeName string
	var stationID *uuid.UUID
	var stationName string

	if err == nil {
		// Use DisplayName if available, otherwise fall back to Label
		bikeName = bikeInfo.Label
		if bikeInfo.DisplayName != nil && *bikeInfo.DisplayName != "" {
			bikeName = *bikeInfo.DisplayName
		}
		stationID = bikeInfo.StationID
		if stationID != nil {
			st, err := sr.GetStation(stationID.String())
			if err == nil {
				stationName = st.Name
			}
		}
	}

	var totalCost *int32
	if b.TotalCost.Valid {
		totalCost = &b.TotalCost.Int32
	}

	return bookingResponse{
		ID:          b.ID,
		BikeID:      b.BikeID,
		BikeName:    bikeName,
		UserID:      b.UserID,
		StationID:   stationID,
		StationName: stationName,
		StartTime:   b.StartTime,
		EndTime:     b.EndTime,
		Status:      b.Status(),
		CreatedAt:   b.CreatedAt,
		TotalCost:   totalCost,
	}
}

var _ = sql.NullInt32{}
