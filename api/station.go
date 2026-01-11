package api

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/semanticallynull/bookingengine-backend/internal/middleware"
	"github.com/semanticallynull/bookingengine-backend/station"
)

func (a *API) stationsHandler(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)
	fmt.Println("UserID", userID)

	stations, err := a.sr.GetStations()
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	var stationResponses []stationResponse
	for _, s := range stations {
		stationResponses = append(stationResponses, toStationResponse(s))
	}
	c.JSON(200, stationResponses)
}

func (a *API) stationHandler(c *gin.Context) {
	id := c.Param("id")

	stations, err := a.sr.GetStation(id)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, toStationResponse(stations))
}

type stationResponse struct {
	ID           uuid.UUID    `json:"id"`
	Name         string       `json:"name"`
	Address      string       `json:"address"`
	OpeningHours string       `json:"opening_hours"`
	Lat          float64      `json:"latitude"`
	Lng          float64      `json:"longitude"`
	Type         station.Type `json:"type"`
}

func toStationResponse(station station.Station) stationResponse {
	return stationResponse{
		ID:           station.ID,
		Name:         station.Name,
		Address:      station.Address,
		OpeningHours: station.OpeningHours,
		Type:         station.Type,
		Lat:          station.Location.P.X,
		Lng:          station.Location.P.Y,
	}
}
