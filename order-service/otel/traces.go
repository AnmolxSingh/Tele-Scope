package otel

import (
	"context"
	"log/slog"
	"os"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

func InitTracer(res *resource.Resource) (*sdktrace.TracerProvider, error) {
	endpoint := os.Getenv("MY_TRACES_ENDPOINT")
	exporter, err := otlptracegrpc.New(context.Background(),
		otlptracegrpc.WithEndpoint(endpoint),
		otlptracegrpc.WithInsecure(),
	)
	if err != nil {
		slog.Error("failed to create OTLP trace exporter", "error", err)
		return nil, err
	}

	// Creating the resource for sdktrace
	// This is where we can set service name and other attributes
	// This is optional but a recommended practice
	// This will help us identifying the service in Jaeger Backend

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
	)

	otel.SetTracerProvider(tp)

	slog.Info("OTLP Trace exporter initialized")
	return tp, nil
}
