package handler

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

// 1. Define your metrics globally.
var (
	meter               = otel.Meter("inventory-service/handler")
	httpRequestsTotal   metric.Int64Counter
	httpRequestDuration metric.Float64Histogram
)

// 2. Use the init() function to create the metric instruments.
func init() {
	var err error
	httpRequestsTotal, err = meter.Int64Counter(
		"http_requests_total",
		metric.WithDescription("Total number of HTTP requests."),
	)
	if err != nil {
		panic(err)
	}

	httpRequestDuration, err = meter.Float64Histogram(
		"http_request_duration_seconds",
		metric.WithDescription("HTTP request latency distribution."),
	)
	if err != nil {
		panic(err)
	}
}

// 3. Create a custom response writer to capture the status code.
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func newResponseWriter(w http.ResponseWriter) *responseWriter {
	return &responseWriter{w, http.StatusOK}
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// 4. Create the middleware function. This will be public so main.go can use it.
func MetricsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rw := newResponseWriter(w)

		next.ServeHTTP(rw, r)

		duration := time.Since(start).Seconds()

		attrs := attribute.NewSet(
			attribute.String("method", r.Method),
			attribute.String("path", r.URL.Path),
			attribute.String("status_code", strconv.Itoa(rw.statusCode)),
		)

		httpRequestDuration.Record(r.Context(), duration, metric.WithAttributeSet(attrs))
		httpRequestsTotal.Add(r.Context(), 1, metric.WithAttributeSet(attrs))
	})
}

func CheckInventory(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Start new span for inventory check
	tr := otel.Tracer("inventory-service")
	_, span := tr.Start(ctx, "CheckInventory")
	defer span.End()

	// Example: simulate inventory check
	fmt.Println("Checking inventory for product...")
	w.Write([]byte("Inventory available"))
}
