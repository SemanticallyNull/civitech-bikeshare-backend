package middleware

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	httpRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests (Rate)",
		},
		[]string{"method", "path", "status"},
	)

	httpRequestErrorsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_request_errors_total",
			Help: "Total number of HTTP request errors",
		},
		[]string{"method", "path", "status", "error_type"},
	)

	httpRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "HTTP request duration in seconds (Duration)",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "path", "status"},
	)
)

func Metrics(reg *prometheus.Registry) gin.HandlerFunc {
	// Register metrics with the provided registry
	reg.MustRegister(httpRequestsTotal, httpRequestErrorsTotal, httpRequestDuration)

	return func(c *gin.Context) {
		// Capture start time
		start := time.Now()

		// Process request
		c.Next()

		// Extract request details
		status := c.Writer.Status()
		statusStr := strconv.Itoa(status)
		path := c.FullPath()
		if path == "" {
			path = c.Request.URL.Path
		}
		method := c.Request.Method

		// Calculate duration
		duration := time.Since(start).Seconds()

		// Increment total requests counter (Rate)
		httpRequestsTotal.WithLabelValues(method, path, statusStr).Inc()

		// Increment error counters if applicable
		if status >= 400 && status < 500 {
			// Client errors (4xx)
			httpRequestErrorsTotal.WithLabelValues(method, path, statusStr, "client").Inc()
		} else if status >= 500 {
			// Server errors (5xx)
			httpRequestErrorsTotal.WithLabelValues(method, path, statusStr, "server").Inc()
		}

		// Observe request duration (Duration)
		httpRequestDuration.WithLabelValues(method, path, statusStr).Observe(duration)
	}
}
