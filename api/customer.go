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

	userID, _ := middleware.GetAuth0ID(c)

	// Fetch user profile from Auth0 (best effort)
	var email, name string
	if accessToken, ok := c.Get("access_token"); ok {
		if token, ok := accessToken.(string); ok && token != "" {
			userInfo, err := a.auth0Client.GetUserInfo(c, token)
			if err != nil {
				logger.WarnContext(c, "failed to fetch user info from Auth0", "error", err)
			} else {
				email = userInfo.Email
				name = userInfo.Name
			}
		}
	}

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
			logger.Error("Failed to save stripe to customer", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}

	// Update profile if we have new data and it differs from stored values
	if email != "" || name != "" {
		currentEmail := ""
		currentName := ""
		if cust.Email.Valid {
			currentEmail = cust.Email.String
		}
		if cust.Name.Valid {
			currentName = cust.Name.String
		}

		if email != currentEmail || name != currentName {
			if err := a.cr.UpdateProfile(c, userID, email, name); err != nil {
				logger.WarnContext(c, "failed to update customer profile", "error", err)
			}
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

	userID, _ := middleware.GetAuth0ID(c)
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
	userID, _ := middleware.GetAuth0ID(c)
	cust, err := a.cr.GetCustomerByAuth0ID(userID)
	if err != nil {
		if errors.Is(err, customer.ErrNotFound) {
			cust, err = a.cr.CreateCustomer(userID)
		} else {
			logger.Error("Cannot get customer", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}

	if !cust.StripeID.Valid {
		logger.Error("Failed to retrieve payment method", "error", err)
		c.JSON(http.StatusPreconditionFailed, gin.H{"state": "require payment method"})
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

type paymentMethodRequest struct {
	PaymentMethod string `json:"paymentMethodId"`
}

func (a *API) setPaymentMethod(c *gin.Context) {
	logger := middleware.GetLogger(c)

	userID, _ := middleware.GetAuth0ID(c)
	cust, err := a.cr.GetCustomerByAuth0ID(userID)
	if err != nil {
		logger.Error("Cannot get customer", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	var req paymentMethodRequest
	err = c.Bind(&req)
	if err != nil {
		logger.Error("Failed to bind request", "error", err)
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	if req.PaymentMethod == "" {
		logger.Error("Payment method ID is required", "error", err)
		c.JSON(400, gin.H{"error": "payment method ID is required"})
		return
	}

	cuParams := stripe.CustomerParams{
		InvoiceSettings: &stripe.CustomerInvoiceSettingsParams{
			DefaultPaymentMethod: stripe.String(req.PaymentMethod),
		},
	}
	_, err = stripecustomer.Update(cust.StripeID.String, &cuParams)
	if err != nil {
		logger.Error("Failed to set payment method", "error", err)
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, gin.H{})
}

type profileRequest struct {
	Email string `json:"email"`
	Name  string `json:"name"`
}

type profileResponse struct {
	Email string `json:"email"`
	Name  string `json:"name"`
}

func (a *API) getProfile(c *gin.Context) {
	logger := middleware.GetLogger(c)

	userID, _ := middleware.GetAuth0ID(c)
	cust, err := a.cr.GetCustomerByAuth0ID(userID)
	if err != nil {
		if errors.Is(err, customer.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "customer not found"})
			return
		}
		logger.ErrorContext(c, "failed to get customer", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get profile"})
		return
	}

	c.JSON(http.StatusOK, profileResponse{
		Email: cust.Email.String,
		Name:  cust.Name.String,
	})
}

func (a *API) updateProfile(c *gin.Context) {
	logger := middleware.GetLogger(c)

	userID, _ := middleware.GetAuth0ID(c)

	var req profileRequest
	if err := c.Bind(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := a.cr.UpdateProfile(c, userID, req.Email, req.Name); err != nil {
		logger.ErrorContext(c, "failed to update customer profile", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update profile"})
		return
	}

	c.JSON(http.StatusOK, profileResponse{
		Email: req.Email,
		Name:  req.Name,
	})
}
