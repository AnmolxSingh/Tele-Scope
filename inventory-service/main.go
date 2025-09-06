package main

import (
	"context"
	"inventory-service/handler"
	"inventory-service/otel"
	"log/slog"
	"net/http"
	"os"

	"github.com/gorilla/mux"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	trace "go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.7.0"
)

func main() {
	//creating a resource
	res := resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.ServiceNameKey.String("inventory-service"),
	)

	trace.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))
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

	inventoryHandler := http.HandlerFunc(handler.CheckInventory)
	wrappedHandler := otelhttp.NewHandler(inventoryHandler, "CheckInventory")

	r := mux.NewRouter()
	r.Handle("/inventory", wrappedHandler).Methods("GET")

	r.Use(handler.MetricsMiddleware)

	slog.Info("Inventory Service running on :8081")
	if err := http.ListenAndServe(":8081", r); err != nil {
		slog.Error("failed to start server", "error", err)
		os.Exit(1)
	}
}
