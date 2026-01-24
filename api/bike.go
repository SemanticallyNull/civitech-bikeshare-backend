package api

import (
	"errors"
	"log/slog"
	"net/http"

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
	id := c.Param("id")
	b, err := a.br.GetBike(c, id)
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

func (a *API) bikeUnlockHandler(c *gin.Context) {
	logger := middleware.GetLogger(c)

	id := c.Param("id")

	err := a.br.ReserveBike(c, id)
	if err != nil {
		if errors.Is(err, bike.ErrNotFound) {
			logger.ErrorContext(c, "bike not found", slog.String("id", id))
			c.JSON(http.StatusNotFound, gin.H{"error": "an error occurred with your request"})
			return
		}

		if errors.Is(err, bike.ErrNotAvailable) {
			logger.ErrorContext(c, "bike not available", slog.String("id", id))
			c.JSON(http.StatusConflict, gin.H{"error": "bike not available"})
			return
		}

		logger.ErrorContext(c, "error reserving bike", slog.String("id", id), slog.String("error", err.Error()))
		c.JSON(500, gin.H{"error": "an error occurred with your request"})
		return
	}

	c.JSON(200, gin.H{"status": "unlocked"})
}

func voltageToPercentage(voltage int) int {
	return (voltage - 340) * 100 / 72
}

type bikeResponse struct {
	ID             uuid.UUID `json:"id"`
	Label          string    `json:"name"`
	IMEI           string    `json:"bleId"`
	Lat            float64   `json:"latitude"`
	Lng            float64   `json:"longitude"`
	BatteryVoltage int       `json:"batteryVoltage"`
	Available      bool      `json:"available"`
}

func toBikeResponse(bike bike.Bike) bikeResponse {
	return bikeResponse{
		ID:             bike.ID,
		Label:          bike.Label,
		IMEI:           bike.IMEI,
		Lat:            bike.Location.P.X,
		Lng:            bike.Location.P.Y,
		BatteryVoltage: voltageToPercentage(340),
		Available:      true,
	}
}
