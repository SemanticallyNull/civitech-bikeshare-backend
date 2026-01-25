package api

import (
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/semanticallynull/bookingengine-backend/bike"
	"github.com/semanticallynull/bookingengine-backend/internal/middleware"
)

func (a *API) bikesHandler(c *gin.Context) {
	bikes, err := a.br.GetBikes(c)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, bikes)
}

func (a *API) bikeHandler(c *gin.Context) {
	label := c.Param("label")
	b, err := a.br.GetBike(c, label)
	if err != nil {
		if errors.Is(err, bike.ErrNotFound) {
			c.JSON(404, gin.H{"error": err.Error()})
			return
		}
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, toBikeResponse(b))
}

func voltageToPercentage(voltage int) int {
	return (voltage - 340) * 100 / 72
}

type bikeResponse struct {
	ID             uuid.UUID `json:"id"`
	Label          string    `json:"label"`
	DisplayName    string    `json:"displayName"`
	IMEI           string    `json:"bleId"`
	Lat            float64   `json:"latitude"`
	Lng            float64   `json:"longitude"`
	BatteryVoltage int       `json:"batteryVoltage"`
	Available      bool      `json:"available"`
	StationName    string    `json:"stationName"`
}

func toBikeResponse(bike bike.Bike) bikeResponse {
	br := bikeResponse{
		ID:             bike.ID,
		Label:          bike.Label,
		IMEI:           bike.IMEI,
		Lat:            bike.Location.P.X,
		Lng:            bike.Location.P.Y,
		BatteryVoltage: voltageToPercentage(340),
		Available:      bike.Available,
	}
	if bike.StationName != nil {
		br.StationName = *bike.StationName
	}
	if bike.DisplayName != nil {
		br.DisplayName = *bike.DisplayName
	}
	return br
}

type upcomingBookingCheckResponse struct {
	HasUpcomingBooking      bool       `json:"hasUpcomingBooking"`
	NextBookingStart        *time.Time `json:"nextBookingStart,omitempty"`
	MinutesUntilNextBooking *int       `json:"minutesUntilNextBooking,omitempty"`
}

func (a *API) upcomingBookingCheckHandler(c *gin.Context) {
	logger := middleware.GetLogger(c)

	userID, ok := middleware.GetAuth0ID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"code": "UNAUTHORIZED", "message": "Authentication required"})
		return
	}

	label := c.Param("label")

	// Verify bike exists
	_, err := a.br.GetBike(c, label)
	if err != nil {
		if errors.Is(err, bike.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"code": "BIKE_NOT_FOUND", "message": "Bike not found"})
			return
		}
		logger.ErrorContext(c, "failed to get bike", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}

	// Check for upcoming booking by another user
	now := time.Now()
	nextBooking, err := a.bkr.GetNextBookingByOtherUser(c, label, userID, now)
	if err != nil {
		logger.ErrorContext(c, "failed to check upcoming bookings", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}

	resp := upcomingBookingCheckResponse{
		HasUpcomingBooking: false,
	}

	if nextBooking != nil && nextBooking.StartTime.Before(now.Add(time.Hour)) {
		resp.HasUpcomingBooking = true
		resp.NextBookingStart = &nextBooking.StartTime
		minutes := int(nextBooking.StartTime.Sub(now).Minutes())
		resp.MinutesUntilNextBooking = &minutes
	}

	c.JSON(http.StatusOK, resp)
}
