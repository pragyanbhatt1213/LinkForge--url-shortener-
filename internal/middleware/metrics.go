package middleware

import (
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/gofiber/fiber/v2"
)

var HTTPRequests = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Name: "linkforge_http_requests_total",
		Help: "Total number of HTTP requests",
	},
	[]string{"method", "path", "status"},
)

var HTTPDuration = prometheus.NewHistogramVec(
	prometheus.HistogramOpts{
		Name: "linkforge_http_request_duration_seconds",
		Help: "HTTP request duration in seconds",
	},
	[]string{"method", "path"},
)

func InitMetrics() {
	prometheus.MustRegister(HTTPRequests)
	prometheus.MustRegister(HTTPDuration)
}

func MetricsMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		start := time.Now()
		err := c.Next()
		duration := time.Since(start).Seconds()
		HTTPRequests.WithLabelValues(
			c.Method(),
			c.Path(),
			strconv.Itoa(c.Response().StatusCode()),
		).Inc()
		HTTPDuration.WithLabelValues(
			c.Method(),
			c.Path(),
		).Observe(duration)
		return err
	}
}