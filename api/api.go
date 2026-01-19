package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/stripe/stripe-go/v84"

	"github.com/semanticallynull/bookingengine-backend/bike"
	"github.com/semanticallynull/bookingengine-backend/customer"
	"github.com/semanticallynull/bookingengine-backend/internal/middleware"
	"github.com/semanticallynull/bookingengine-backend/internal/o11y"
	"github.com/semanticallynull/bookingengine-backend/ride"
	"github.com/semanticallynull/bookingengine-backend/station"
)

type API struct {
	r  *gin.Engine
	br *bike.Repository
	sr *station.Repository
	cr *customer.Repository
	rr *ride.Repository

	jwtValidator *middleware.JWTValidator
	stripePK     string
	stripeSK     string
}

func New(br *bike.Repository, sr *station.Repository, cr *customer.Repository, rr *ride.Repository,
	o *o11y.Observability, auth0Domain, audience, metricsUsername, metricsPassword, stripePK, stripeSK string) *API {

	a := &API{
		r:        gin.New(),
		br:       br,
		sr:       sr,
		cr:       cr,
		rr:       rr,
		stripePK: stripePK,
		stripeSK: stripeSK,
	}

	stripe.Key = stripeSK

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
		protected.GET("/bikes/:id/unlock", a.bikeUnlockHandler)
		protected.GET("/stations", a.stationsHandler)
		protected.GET("/stations/:id", a.stationHandler)
		protected.GET("/stripe/pubkey", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"publishableKey": stripePK})
		})
		protected.POST("/customer/session", a.createCustomerSession)
		protected.POST("/customer/setupintent", a.createSetupIntent)
		protected.POST("/customer/paymentmethod", a.setPaymentMethod)
		protected.GET("/customer/preride", a.preRide)
		protected.POST("/ride/start", a.startRideHandler)
		protected.POST("/ride/end", a.endRideHandler)
		protected.GET("/ride/current", a.currentRideHandler)
	}

	return a
}

func (a *API) Router() *gin.Engine {
	return a.r
}
