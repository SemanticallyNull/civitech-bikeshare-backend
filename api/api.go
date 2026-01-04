package api

import (
	"github.com/gin-gonic/gin"

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
	a.r.GET("/stations", a.stationsHandler)
	a.r.GET("/stations/:id", a.stationHandler)

	return a
}

func (a *API) Router() *gin.Engine {
	return a.r
}
