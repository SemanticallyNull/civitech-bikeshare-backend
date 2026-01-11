package middleware

import (
	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
)

func Tracing() gin.HandlerFunc {
	tracer := otel.Tracer("gin-server")
	propagator := otel.GetTextMapPropagator()

	return func(c *gin.Context) {
		// Extract parent span from headers (for distributed tracing)
		ctx := propagator.Extract(c.Request.Context(), propagation.HeaderCarrier(c.Request.Header))

		// Start new span
		ctx, span := tracer.Start(ctx, c.Request.Method+" "+c.FullPath())
		defer span.End()

		// Store context in Gin context
		c.Request = c.Request.WithContext(ctx)

		// Add trace metadata
		span.SetAttributes(
			attribute.String("http.method", c.Request.Method),
			attribute.String("http.url", c.Request.URL.String()),
		)

		c.Next()

		// Record response
		span.SetAttributes(attribute.Int("http.status_code", c.Writer.Status()))
	}
}
