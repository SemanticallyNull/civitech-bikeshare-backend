package api

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/semanticallynull/bookingengine-backend/bike"
	"github.com/semanticallynull/bookingengine-backend/station"
)

type API struct {
	r  *gin.Engine
	br *bike.Repository
	sr *station.Repository
}

func New(br *bike.Repository, sr *station.Repository) *API {
	a := &API{
		r:  gin.Default(),
		br: br,
		sr: sr,
	}

	a.r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})
	a.r.GET("/bikes/nearby", a.bikesHandler)
	a.r.GET("/bikes/:id", a.bikeHandler)
	a.r.GET("/stations", func(c *gin.Context) {
		stations, err := sr.GetStations()
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}

		var stationResponses []stationResponse
		for _, s := range stations {
			stationResponses = append(stationResponses, toStationResponse(s))
		}
		c.JSON(200, stationResponses)
	})

	return a
}

func (a *API) Router() *gin.Engine {
	return a.r
}

type stationResponse struct {
	ID           uuid.UUID
	Name         string
	Address      string
	OpeningHours string  `db:"opening_hours"`
	Lat          float64 `json:"latitude"`
	Lng          float64 `json:"longitude"`
	Type         station.Type
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
