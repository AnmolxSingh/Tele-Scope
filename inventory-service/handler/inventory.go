package handler

import (
	"log/slog"
	"math/rand"
	"net/http"
	"strconv"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

// metrics defined globally.
var (
	meter               = otel.Meter("inventory-service/handler")
	httpRequestsTotal   metric.Int64Counter
	httpRequestDuration metric.Float64Histogram
)

// the init() function to create the metric instruments.
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

// a custom response writer to capture the status code.
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

// the middleware function
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
	slog.InfoContext(ctx, "Checking inventory for product...")

	status := "success"
	if rand.Intn(10) < 2 { // 20% chance of failure
		status = "failed"
	}

	if status == "failed" {
		http.Error(w, "Inventory check failed", http.StatusInternalServerError)
		slog.InfoContext(ctx, "Inventory check",
			"status", status,
		)
		w.Write([]byte("Inventory unavailable"))
		return
	}

	w.Write([]byte("Inventory available"))
}
