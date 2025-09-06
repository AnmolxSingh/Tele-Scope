// otel/metrics.go
package otel

import (
	"context"
	"log/slog"
	"os"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
)

// InitMetrics sets up the Prometheus metrics exporter.
// It returns the HTTP handler for the /metrics endpoint and a shutdown function.
func InitMetrics(res *resource.Resource) (func(context.Context) error, error) {
	ctx := context.Background()
	endpoint := os.Getenv("MY_METRICS_ENDPOINT")

	// CHANGED: Use the OTLP/HTTP exporter which pushes to a collector.
	exporter, err := otlpmetrichttp.New(ctx,
		otlpmetrichttp.WithEndpoint(endpoint), // Assumes collector is on this address
		otlpmetrichttp.WithInsecure(),         // Use HTTP for local dev
	)
	if err != nil {
		return nil, err
	}

	// The MeterProvider is the entry point for the metrics SDK.
	// CHANGED: The OTLP exporter is wrapped in a PeriodicReader.
	// This reader periodically collects and sends metrics.
	mp := metric.NewMeterProvider(
		metric.WithReader(metric.NewPeriodicReader(exporter)),
		metric.WithResource(res),
	)

	// Set the global meter provider, so you can get a Meter from it globally.
	otel.SetMeterProvider(mp)
	slog.Info("OTLP metrics exporter initialized")

	// CHANGED: Return only the shutdown function.
	return mp.Shutdown, nil
}
