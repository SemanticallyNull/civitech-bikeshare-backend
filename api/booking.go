package api

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/semanticallynull/bookingengine-backend/bike"
	"github.com/semanticallynull/bookingengine-backend/booking"
	"github.com/semanticallynull/bookingengine-backend/internal/middleware"
)

type bookingResponse struct {
	ID          uuid.UUID             `json:"id"`
	BikeID      uuid.UUID             `json:"bikeId"`
	BikeName    string                `json:"bikeName"`
	BikeLabel   string                `json:"bikeLabel"`
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

func (a *API) getBookingsHandler(c *gin.Context) {
	logger := middleware.GetLogger(c)

	userID, ok := middleware.GetAuth0ID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"code": "UNAUTHORIZED", "message": "Authentication required"})
		return
	}
	user, err := a.cr.GetCustomerByAuth0ID(userID)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"code": "UNAUTHORIZED", "message": "Authentication required"})
		return
	}

	// Parse optional status filter
	statusStr := c.Query("status")
	var statusPtr *booking.BookingStatus
	if statusStr != "" {
		status := booking.BookingStatus(statusStr)
		statusPtr = &status
	}

	bookings, err := a.bkr.GetByUserID(c, user.ID, statusPtr)
	if err != nil {
		logger.ErrorContext(c, "failed to get user bookings", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}

	responses := make([]bookingResponse, 0, len(bookings))
	for _, b := range bookings {
		resp, err := a.toBookingResponse(c, b)
		if err != nil {
			logger.ErrorContext(c, "failed to build booking response", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
			return
		}
		responses = append(responses, resp)
	}

	c.JSON(http.StatusOK, responses)
}

func (a *API) createBookingHandler(c *gin.Context) {
	logger := middleware.GetLogger(c)

	userID, ok := middleware.GetAuth0ID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"code": "UNAUTHORIZED", "message": "Authentication required"})
		return
	}
	user, err := a.cr.GetCustomerByAuth0ID(userID)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"code": "UNAUTHORIZED", "message": "Authentication required"})
		return
	}

	var req createBookingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": "INVALID_REQUEST", "message": err.Error()})
		return
	}

	// Parse times
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

	// Validate duration (1-24 hours)
	duration := endTime.Sub(startTime)
	if duration < time.Hour {
		c.JSON(http.StatusBadRequest, gin.H{"code": "INVALID_DURATION", "message": "Booking duration must be at least 1 hour"})
		return
	}
	if duration > 24*time.Hour {
		c.JSON(http.StatusBadRequest, gin.H{"code": "INVALID_DURATION", "message": "Booking duration cannot exceed 24 hours"})
		return
	}
	fmt.Println(req)

	// Verify bike exists
	bikeID := req.BikeID

	bk, err := a.br.GetBike(c, req.BikeID)
	if err != nil {
		if errors.Is(err, bike.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"code": "BIKE_NOT_FOUND", "message": "Bike not found"})
			return
		}
		logger.ErrorContext(c, "failed to get bike", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}

	// Check for buffer conflict: another user's booking within 1 hour of our end time
	nextBooking, err := a.bkr.GetNextBookingByOtherUser(c, bikeID, userID, endTime)
	if err != nil {
		logger.ErrorContext(c, "failed to check for buffer conflict", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}
	if nextBooking != nil && nextBooking.StartTime.Before(endTime.Add(time.Hour)) {
		c.JSON(http.StatusConflict, gin.H{
			"code":    "BUFFER_CONFLICT",
			"message": "Another booking starts within 1 hour of your booking's end time",
		})
		return
	}

	// Create booking
	b := &booking.Booking{
		ID:        uuid.New(),
		BikeID:    bk.ID,
		UserID:    user.ID,
		StartTime: startTime,
		EndTime:   endTime,
	}

	err = a.bkr.Create(c, b)
	if err != nil {
		if errors.Is(err, booking.ErrOverlap) {
			c.JSON(http.StatusConflict, gin.H{"code": "BOOKING_OVERLAP", "message": "Booking overlaps with existing booking"})
			return
		}
		logger.ErrorContext(c, "failed to create booking", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}

	resp, err := a.toBookingResponse(c, *b)
	if err != nil {
		logger.ErrorContext(c, "failed to build booking response", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}

	c.JSON(http.StatusCreated, resp)
}

func (a *API) getCurrentBookingHandler(c *gin.Context) {
	logger := middleware.GetLogger(c)

	userID, ok := middleware.GetAuth0ID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"code": "UNAUTHORIZED", "message": "Authentication required"})
		return
	}

	b, err := a.bkr.GetCurrentByUserID(c, userID)
	if err != nil {
		logger.ErrorContext(c, "failed to get current booking", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}

	if b == nil {
		c.JSON(http.StatusOK, nil)
		return
	}

	resp, err := a.toBookingResponse(c, *b)
	if err != nil {
		logger.ErrorContext(c, "failed to build booking response", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}

	c.JSON(http.StatusOK, resp)
}

func (a *API) cancelBookingHandler(c *gin.Context) {
	logger := middleware.GetLogger(c)

	userID, ok := middleware.GetAuth0ID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"code": "UNAUTHORIZED", "message": "Authentication required"})
		return
	}

	customer, err := a.cr.GetCustomerByAuth0ID(userID)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"code": "UNAUTHORIZED", "message": "Authentication required"})
		return
	}

	bookingID, err := uuid.Parse(c.Param("bookingId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": "INVALID_REQUEST", "message": "Invalid bookingId"})
		return
	}

	b, err := a.bkr.Cancel(c, bookingID, customer.ID)
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
		logger.ErrorContext(c, "failed to cancel booking", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}

	resp, err := a.toBookingResponse(c, b)
	if err != nil {
		logger.ErrorContext(c, "failed to build booking response", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}

	c.JSON(http.StatusOK, resp)
}

// toBookingResponse converts a booking to an API response, fetching bike/station info.
func (a *API) toBookingResponse(c *gin.Context, b booking.Booking) (bookingResponse, error) {
	// Get bike info for the response
	bikeInfo, err := a.br.GetBike(c, b.BikeLabel)
	if err != nil && !errors.Is(err, bike.ErrNotFound) {
		return bookingResponse{}, err
	}

	var stationID *uuid.UUID
	var stationName string

	if err == nil {
		stationID = bikeInfo.StationID
		if stationID != nil {
			station, err := a.sr.GetStation(stationID.String())
			if err == nil {
				stationName = station.Name
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
		BikeName:    b.BikeName.String,
		BikeLabel:   b.BikeLabel,
		UserID:      b.UserID.String(),
		StationID:   stationID,
		StationName: stationName,
		StartTime:   b.StartTime,
		EndTime:     b.EndTime,
		Status:      b.Status(),
		CreatedAt:   b.CreatedAt,
		TotalCost:   totalCost,
	}, nil
}

// Ensure sql.NullInt32 is imported
var _ = sql.NullInt32{}
