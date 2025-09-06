package handler

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"math/rand"
	"net/http"
	"strconv"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
)

// 1. Define your metrics globally.
var (
	meter               = otel.Meter("order-service/handler")
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

// In-memory storage
var orders = make(map[string]map[string]interface{})

// CreateOrder handles new orders
func CreateOrder(w http.ResponseWriter, r *http.Request) {
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))
	// Optional: tracing span if using OpenTelemetry
	tr := otel.Tracer("order-service")
	ctx, span := tr.Start(r.Context(), "CreateOrder")
	defer span.End()

	// Define request structure
	var req struct {
		ProductID string `json:"productId"`
		Quantity  int    `json:"quantity"`
		User      string `json:"user"`
		Amount    int    `json:"amount"`
	}

	// Parse JSON request body
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid request"})
		return
	}

	request1, _ := http.NewRequestWithContext(ctx, "GET", "http://inventory-service:8081/inventory", nil)
	// Send request
	client := http.Client{
		Transport: otelhttp.NewTransport(http.DefaultTransport),
		Timeout:   5 * time.Second,
	}
	resp1, err := client.Do(request1)
	if err != nil {
		// logging the error with context
		slog.ErrorContext(ctx, "failed to contact inventory service", "error", err)
		http.Error(w, "failed to contact inventory", 500)
		return
	}
	defer resp1.Body.Close()

	// 2. Call Payment Service
	paymentReq := map[string]interface{}{
		"orderId": "temp-order", // replace with actual ID later
		"user":    req.User,
		"amount":  req.Amount,
	}
	paymentReqBody, _ := json.Marshal(paymentReq)

	request2, _ := http.NewRequestWithContext(ctx, "POST", "http://payment-service:8082/payment", bytes.NewBuffer(paymentReqBody))
	request2.Header.Set("Content-Type", "application/json")

	resp2, err := client.Do(request2)
	if err != nil {
		http.Error(w, "failed to contact payment", http.StatusInternalServerError)
		return
	}
	defer resp2.Body.Close()

	// Check HTTP status
	if resp2.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp2.Body)
		http.Error(w, "payment service error: "+string(bodyBytes), resp2.StatusCode)
		return
	}

	// Parse Payment Response
	var paymentResp map[string]interface{}
	if err := json.NewDecoder(resp2.Body).Decode(&paymentResp); err != nil {
		http.Error(w, "invalid payment response", http.StatusInternalServerError)
		return
	}

	// Create order with input fields
	id := strconv.Itoa(rand.Intn(100000))
	order := map[string]interface{}{
		"id":        id,
		"productId": req.ProductID,
		"quantity":  req.Quantity,
		"user":      req.User,
		"status":    "created",
	}

	slog.InfoContext(ctx, "Order created successfully", "order_id", id)

	// Store in memory
	orders[id] = order

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(order)
}

// GetOrder returns order by ID
// func GetOrder(w http.ResponseWriter, r *http.Request) {
// 	tr := otel.Tracer("order-service")
// 	_, span := tr.Start(r.Context(), "GetOrder")
// 	vars := mux.Vars(r)
// 	id := vars["id"]
// 	order, exists := orders[id]
// 	if !exists {
// 		w.WriteHeader(http.StatusNotFound)
// 		json.NewEncoder(w).Encode(map[string]string{"error": "Order not found"})
// 		return
// 	}
// 	w.Header().Set("Content-Type", "application/json")
// 	json.NewEncoder(w).Encode(order)
// }
