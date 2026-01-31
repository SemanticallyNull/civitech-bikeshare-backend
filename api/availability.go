package api

import (
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/semanticallynull/bookingengine-backend/internal/middleware"
)

type bikeAvailabilityResponse struct {
	BikeID      uuid.UUID                 `json:"bikeId"`
	BikeName    string                    `json:"bikeName"`
	DisplayName *string                   `json:"displayName,omitempty"`
	BikeImage   *string                   `json:"imageUrl,omitempty"`
	StationID   *uuid.UUID                `json:"stationId,omitempty"`
	StationName string                    `json:"stationName,omitempty"`
	Bookings    []bookingTimeSlotResponse `json:"bookings"`
}

type bookingTimeSlotResponse struct {
	StartTime    time.Time `json:"startTime"`
	EndTime      time.Time `json:"endTime"`
	IsOwnBooking bool      `json:"isOwnBooking"`
}

func (a *API) availabilityHandler(c *gin.Context) {
	logger := middleware.GetLogger(c)

	userID, ok := middleware.GetAuth0ID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"code": "UNAUTHORIZED", "message": "Authentication required"})
		return
	}

	// Parse optional query params
	stationID := c.Query("stationId")
	startDateStr := c.Query("startDate")
	endDateStr := c.Query("endDate")

	var stationIDPtr *string
	if stationID != "" {
		stationIDPtr = &stationID
	}

	startDate, endDate, err := parseDate(startDateStr, endDateStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": "INVALID_DATE", "message": err})
		return
	}

	// Fetch bikes with station info
	bikes, err := a.br.GetBikesWithStations(c, stationIDPtr)
	if err != nil {
		logger.ErrorContext(c, "failed to get bikes with stations", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}

	// Build availability response for each bike
	availability := make([]bikeAvailabilityResponse, 0, len(bikes))
	for _, bike := range bikes {
		// Get bookings for this bike
		slots, err := a.bkr.GetBookingsForBike(c, bike.ID, startDate, endDate)
		if err != nil {
			logger.ErrorContext(c, "failed to get bookings for bike", "bikeId", bike.ID, "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
			return
		}

		bookings := make([]bookingTimeSlotResponse, 0, len(slots))
		for _, slot := range slots {
			bookings = append(bookings, bookingTimeSlotResponse{
				StartTime:    slot.StartTime,
				EndTime:      slot.EndTime,
				IsOwnBooking: slot.UserID == userID,
			})
		}

		availability = append(availability, bikeAvailabilityResponse{
			BikeID:      bike.ID,
			BikeName:    bike.Label,
			DisplayName: bike.DisplayName,
			BikeImage:   bike.ImageURL,
			StationID:   bike.StationID,
			StationName: bike.StationName,
			Bookings:    bookings,
		})
	}

	c.JSON(http.StatusOK, availability)
}

func parseDate(startDateStr string, endDateStr string) (*time.Time, *time.Time, error) {
	var startDate, endDate *time.Time
	if startDateStr != "" {
		t, err := time.Parse(time.RFC3339, startDateStr)
		if err != nil {
			return nil, nil, errors.New("invalid startDate format")
		}
		startDate = &t
	}
	if endDateStr != "" {
		t, err := time.Parse(time.RFC3339, endDateStr)
		if err != nil {
			return nil, nil, errors.New("invalid endDate format")
		}
		endDate = &t
	}
	return startDate, endDate, nil
}
