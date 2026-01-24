package api

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stripe/stripe-go/v84"
	"github.com/stripe/stripe-go/v84/invoice"
	"go.opentelemetry.io/otel"

	"github.com/semanticallynull/bookingengine-backend/customer"
	"github.com/semanticallynull/bookingengine-backend/internal/middleware"
	riderepo "github.com/semanticallynull/bookingengine-backend/ride"
)

type rideRequest struct {
	BikeID string `json:"bikeId"`
}

func (a *API) startRideHandler(c *gin.Context) {
	logger := middleware.GetLogger(c)

	var req rideRequest
	if err := c.Bind(&req); err != nil {
		logger.Error("Failed to bind request", "error", err)
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	userID, _ := middleware.GetUserID(c)
	customer, err := a.cr.GetCustomerByAuth0ID(userID)
	if err != nil {
		logger.Error("Failed to get customer", "error", err)
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	bike, err := a.br.GetBike(c, req.BikeID)
	if err != nil {
		logger.Error("Failed to get bike", "error", err)
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	ride, err := a.rr.StartRide(c, bike.ID, customer.ID)
	if err != nil {
		custID, ok := riderepo.CustomerFromRideInProgressError(err)
		if ok && custID == customer.ID {
			logger.Info("Customer already has an active ride", "error", err)
			c.JSON(200, gin.H{"ok": "Customer already has an active ride"})
			return
		}

		logger.Error("Failed to start ride", "error", err)
		c.JSON(500, gin.H{"error": err.Error()})
	}

	c.JSON(200, ride)
}

func (a *API) endRideHandler(c *gin.Context) {
	logger := middleware.GetLogger(c)

	var req rideRequest
	if err := c.Bind(&req); err != nil {
		logger.Error("Failed to bind request", "error", err)
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	userID, _ := middleware.GetUserID(c)
	customer, err := a.cr.GetCustomerByAuth0ID(userID)
	if err != nil {
		logger.Error("Failed to get customer", "error", err)
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	mins, err := a.rr.EndRide(c, customer.ID)
	if err != nil {
		logger.Error("Failed to start ride", "error", err)
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	go func() {
		inParams := &stripe.InvoiceParams{
			Customer: stripe.String(customer.StripeID.String),
		}
		in, err := invoice.New(inParams)
		if err != nil {
			logger.Error("Failed to create invoice", "error", err)
			return
		}

		ilParams := &stripe.InvoiceAddLinesParams{
			Params:          stripe.Params{},
			Expand:          nil,
			InvoiceMetadata: nil,
			Lines: []*stripe.InvoiceAddLinesLineParams{
				{
					Amount:      stripe.Int64(100),
					Description: stripe.String("Ride Unlock"),
					TaxAmounts: []*stripe.InvoiceAddLinesLineTaxAmountParams{
						{
							Amount:        stripe.Int64(12),
							TaxableAmount: stripe.Int64(88),
							TaxRateData: &stripe.InvoiceAddLinesLineTaxAmountTaxRateDataParams{
								Percentage:  stripe.Float64(13.5),
								Description: stripe.String("VAT - Reduced Rate"),
								DisplayName: stripe.String("VAT - Reduced Rate (13.5%)"),
								Inclusive:   stripe.Bool(true),
							},
						},
					},
				},
				{
					Amount:      stripe.Int64(int64(15 * mins)),
					Description: stripe.String(fmt.Sprintf("Ride - %d minutes", mins)),
					TaxAmounts: []*stripe.InvoiceAddLinesLineTaxAmountParams{
						{
							Amount:        stripe.Int64(int64(2 * mins)),
							TaxableAmount: stripe.Int64(int64(13 * mins)),
							TaxRateData: &stripe.InvoiceAddLinesLineTaxAmountTaxRateDataParams{
								Percentage:  stripe.Float64(13.5),
								Description: stripe.String("VAT - Reduced Rate"),
								DisplayName: stripe.String("VAT - Reduced Rate (13.5%)"),
								Inclusive:   stripe.Bool(true),
							},
						},
					},
				},
			},
		}
		_, err = invoice.AddLines(in.ID, ilParams)
		if err != nil {
			logger.Error("Failed to add lines to invoice", "error", err)
			return
		}

		params := &stripe.InvoiceFinalizeInvoiceParams{}
		_, err = invoice.FinalizeInvoice(in.ID, params)
		if err != nil {
			logger.Error("Failed to finalize invoice", "error", err)
			return
		}
		_, err = invoice.Pay(in.ID, nil)
		if err != nil {
			logger.Error("Failed to pay invoice", "error", err)
		}
	}()

	c.JSON(200, "OK")
}

type RideState struct {
	InProgress bool      `json:"inProgress"`
	BikeID     string    `json:"bikeId"`
	StartedAt  time.Time `json:"startedAt"`
}

func (a *API) currentRideHandler(c *gin.Context) {
	_, span := otel.GetTracerProvider().Tracer("api").Start(c.Request.Context(), "currentRideHandler")
	defer span.End()

	logger := middleware.GetLogger(c)

	userID, _ := middleware.GetUserID(c)
	cust, err := a.cr.GetCustomerByAuth0ID(userID)
	if err != nil {
		if errors.Is(err, customer.ErrNotFound) {
			cust, err = a.cr.CreateCustomer(userID)
			if err != nil {
				logger.Error("Failed to save customer", "error", err)
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
		} else {
			logger.Error("Failed to get customer", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}

	ride, err := a.cr.CurrentRide(cust.ID)
	if err != nil {
		if errors.Is(err, customer.ErrNoRideInProgress) {
			c.JSON(200, RideState{
				InProgress: false,
			})
			return
		}
		logger.Error("Failed to get current ride", "error", err)
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, RideState{
		InProgress: true,
		BikeID:     ride.BikeID,
		StartedAt:  ride.StartedAt,
	})
}
