package api

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/stripe/stripe-go/v84"
	stripecustomer "github.com/stripe/stripe-go/v84/customer"
	"github.com/stripe/stripe-go/v84/customersession"
	"github.com/stripe/stripe-go/v84/setupintent"

	"github.com/semanticallynull/bookingengine-backend/customer"
	"github.com/semanticallynull/bookingengine-backend/internal/middleware"
)

func (a *API) createCustomerSession(c *gin.Context) {
	logger := middleware.GetLogger(c)

	userID, _ := middleware.GetUserID(c)
	cust, err := a.cr.GetCustomerByAuth0ID(userID)
	if err != nil {
		if errors.Is(err, customer.ErrNotFound) {
			err = a.cr.CreateCustomer(userID)
			if err != nil {
				logger.Error("Failed to save customer", "error", err)
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
		} else {
			logger.Error("Failed to save stripe to customer", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}

	if !cust.StripeID.Valid {
		stripeCustomer, err := stripecustomer.New(&stripe.CustomerParams{
			Metadata: map[string]string{
				"auth0_id": userID,
				"id":       cust.ID.String(),
			},
		})
		if err != nil {
			logger.Error("Failed to create stripe customer", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		cust.StripeID.String = stripeCustomer.ID

		err = a.cr.AddStripeIDToCustomer(userID, stripeCustomer.ID)
		if err != nil {
			logger.Error("Failed to save stripe customer ID to customer", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}

	csParams := &stripe.CustomerSessionParams{
		Customer: stripe.String(cust.StripeID.String),
	}
	csParams.AddExtra("components[customer_sheet][enabled]", "true")
	csParams.AddExtra("components[customer_sheet][features][payment_method_remove]", "enabled")
	cs, _ := customersession.New(csParams)

	c.JSON(http.StatusOK, struct {
		CustomerID   string `json:"customerId"`
		ClientSecret string `json:"clientSecret"`
	}{
		CustomerID:   cust.StripeID.String,
		ClientSecret: cs.ClientSecret,
	})
}

func (a *API) createSetupIntent(c *gin.Context) {
	logger := middleware.GetLogger(c)

	userID, _ := middleware.GetUserID(c)
	cust, err := a.cr.GetCustomerByAuth0ID(userID)
	if err != nil {
		logger.Error("Cannot get customer", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if !cust.StripeID.Valid {
		logger.Error("Customer has no stripe ID", "customerId", cust.ID)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Customer has no stripe ID"})
		return
	}

	siparams := &stripe.SetupIntentParams{
		Customer: stripe.String(cust.StripeID.String),
		// In the latest version of the API, specifying the `automatic_payment_methods` parameter
		// is optional because Stripe enables its functionality by default.
		AutomaticPaymentMethods: &stripe.SetupIntentAutomaticPaymentMethodsParams{
			Enabled: stripe.Bool(true),
		},
	}
	si, err := setupintent.New(siparams)
	if err != nil {
		logger.Error("Failed to create setup intent", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, struct {
		SetupIntent string `json:"setupIntent"`
	}{
		SetupIntent: si.ClientSecret,
	})
}

func (a *API) preRide(c *gin.Context) {
	logger := middleware.GetLogger(c)
	userID, _ := middleware.GetUserID(c)
	cust, err := a.cr.GetCustomerByAuth0ID(userID)
	if err != nil {
		logger.Error("Cannot get customer", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if !cust.StripeID.Valid {
		logger.Error("Failed to retrieve payment method", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "no stripe ID"})
		return
	}

	params := &stripe.CustomerListPaymentMethodsParams{
		Customer: stripe.String(cust.StripeID.String),
	}
	result := stripecustomer.ListPaymentMethods(params)
	if result.Err() != nil {
		logger.Error("Failed to retrieve payment method", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if result.Next() {
		c.JSON(http.StatusOK, gin.H{"paymentMethod": result.PaymentMethod().ID})
		return
	}

	c.JSON(http.StatusPreconditionFailed, gin.H{"state": "require payment method"})
}
