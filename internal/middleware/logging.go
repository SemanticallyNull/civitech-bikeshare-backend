package middleware

import (
	"log/slog"
	"time"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel/trace"
)

// LoggerKey for storing logger in Gin context
const LoggerKey = "logger"

func Logging(baseLogger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		// Extract trace context from request context
		span := trace.SpanFromContext(c.Request.Context())
		traceID := span.SpanContext().TraceID().String()
		spanID := span.SpanContext().SpanID().String()

		// Create request-scoped logger with trace context
		logger := baseLogger.With(
			slog.String("trace_id", traceID),
			slog.String("span_id", spanID),
			slog.String("method", c.Request.Method),
			slog.String("path", c.Request.URL.Path),
		)

		// Store in Gin context for handlers to use
		c.Set(LoggerKey, logger)

		c.Next()

		// Log request completion
		logger.Info("request completed",
			slog.Int("status", c.Writer.Status()),
			slog.Duration("duration", time.Since(start)),
			slog.Int("size", c.Writer.Size()),
		)
	}
}

// Helper to extract logger from context
func GetLogger(c *gin.Context) *slog.Logger {
	if logger, exists := c.Get(LoggerKey); exists {
		return logger.(*slog.Logger)
	}
	return slog.Default()
}
