package api

import (
	"errors"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/semanticallynull/bookingengine-backend/bike"
)

func (a *API) bikesHandler(c *gin.Context) {
	bikes, err := a.br.GetBikes()
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, bikes)
}

func (a *API) bikeHandler(c *gin.Context) {
	id := c.Param("id")
	b, err := a.br.GetBike(id)
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
	Label          string    `json:"name"`
	IMEI           string    `json:"bleId"`
	Lat            float64   `json:"latitude"`
	Lng            float64   `json:"longitude"`
	BatteryVoltage int       `json:"battery_voltage"`
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
