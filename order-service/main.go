package main

import (
	"context"
	"log/slog"
	"net/http"
	"order-service/handler"
	"order-service/otel"
	"os"

	"github.com/gorilla/mux"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.7.0"
)

func main() {
	//creating a resource
	res := resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.ServiceNameKey.String("order-service"),
	)
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
	r.HandleFunc("/orders", handler.CreateOrder).Methods("POST")

	r.Use(handler.MetricsMiddleware)

	slog.Info("Order Service running on :8080")
	if err := http.ListenAndServe(":8080", r); err != nil {
		slog.Error("failed to start server", "error", err)
		os.Exit(1)
	}
}
