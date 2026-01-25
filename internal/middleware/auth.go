package middleware

import (
	"log"

	jwtmiddleware "github.com/auth0/go-jwt-middleware/v2"
	"github.com/auth0/go-jwt-middleware/v2/validator"
	"github.com/gin-gonic/gin"
)

// GetAuth0ID extracts the user ID (sub claim) from the JWT token in the Gin context
func GetAuth0ID(c *gin.Context) (string, bool) {
	// The JWT middleware stores the validated token in the request context
	// under the key "user" by default
	claims, exists := c.Request.Context().Value(jwtmiddleware.ContextKey{}).(*validator.ValidatedClaims)
	if !exists {
		log.Printf("No user claims found in context")
		return "", false
	}

	return claims.RegisteredClaims.Subject, true
}
