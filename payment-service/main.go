package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"payment-service/handler"
	"payment-service/otel"

	"github.com/gorilla/mux"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	trace "go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.7.0"
)

func main() {
	ctx := context.Background()
	//creating a resource
	res := resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.ServiceNameKey.String("payment-service"),
	)

	trace.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))
	// Setup OpenTelemetry
	// Setup OpenTelemetry
	tp, err := otel.InitTracer(res)
	if err != nil {
		slog.Error("failed to initialize tracer", "error", err)
	}
	defer tp.Shutdown(context.Background())

	shutdownMetrics, err := otel.InitMetrics(res)
	if err != nil {
		slog.Error("failed to initialize metrics", "error", err)
	}
	defer shutdownMetrics(context.Background())

	shutdownLogger, err := otel.InitLogger(res) // Pass resource to logger
	if err != nil {
		slog.Error("failed to initialize logger", "error", err)
		os.Exit(1)
	}
	defer shutdownLogger(context.Background())

	r := mux.NewRouter()
	paymentHandler := http.HandlerFunc(handler.MakePayment)
	wrappedHandler := otelhttp.NewHandler(paymentHandler, "MakePayment")
	r.Handle("/payment", wrappedHandler).Methods(http.MethodPost)

	r.Use(handler.MetricsMiddleware)

	slog.InfoContext(ctx, "Payment Service running on :8082")
	if err := http.ListenAndServe(":8082", r); err != nil {
		slog.Error("failed to start server", "error", err)
		os.Exit(1)
	}
}
