package api

import (
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/semanticallynull/bookingengine-backend/bike"
	"github.com/semanticallynull/bookingengine-backend/internal/middleware"
	"github.com/semanticallynull/bookingengine-backend/internal/o11y"
	"github.com/semanticallynull/bookingengine-backend/station"
)

type API struct {
	r  *gin.Engine
	br *bike.Repository
	sr *station.Repository

	jwtValidator *middleware.JWTValidator
}

func New(br *bike.Repository, sr *station.Repository, o *o11y.Observability, auth0Domain, audience, metricsUsername, metricsPassword string) *API {
	a := &API{
		r:  gin.New(),
		br: br,
		sr: sr,
	}

	// Global middleware (apply to all routes)
	a.r.Use(gin.Recovery())
	a.r.Use(middleware.Tracing())
	a.r.Use(middleware.Logging(o.Logger))
	a.r.Use(middleware.Metrics(o.Registry))

	// Metrics endpoint with basic auth (if credentials provided)
	if metricsUsername != "" && metricsPassword != "" {
		authorized := a.r.Group("/", gin.BasicAuth(gin.Accounts{
			metricsUsername: metricsPassword,
		}))
		authorized.GET("/metrics", gin.WrapH(promhttp.HandlerFor(o.Registry, promhttp.HandlerOpts{})))
	}

	// Protected API routes (require JWT)
	a.jwtValidator = middleware.NewJWTValidator(auth0Domain, audience)
	protected := a.r.Group("/")
	protected.Use(a.jwtValidator.EnsureValidToken())
	{
		protected.GET("/bikes/nearby", a.bikesHandler)
		protected.GET("/bikes/:id", a.bikeHandler)
		protected.GET("/stations", a.stationsHandler)
		protected.GET("/stations/:id", a.stationHandler)
	}

	return a
}

func (a *API) Router() *gin.Engine {
	return a.r
}
