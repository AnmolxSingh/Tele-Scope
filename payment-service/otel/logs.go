package otel

import (
	"context"
	"log/slog"
	"os"

	"go.opentelemetry.io/contrib/bridges/otelslog"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp"
	logglobal "go.opentelemetry.io/otel/log/global"
	"go.opentelemetry.io/otel/sdk/log"
	"go.opentelemetry.io/otel/sdk/resource"
)

// InitLogger configures slog to send logs to the OpenTelemetry Collector.
func InitLogger(res *resource.Resource) (func(context.Context) error, error) {
	ctx := context.Background()
	endpoint := os.Getenv("MY_LOGS_ENDPOINT")

	// 1. Create a new OTLP log exporter
	logExporter, err := otlploghttp.New(ctx, otlploghttp.WithInsecure(), otlploghttp.WithEndpoint(endpoint))
	if err != nil {
		return nil, err
	}

	// 2. Create a logger provider with a batch processor and the exporter.
	loggerProvider := log.NewLoggerProvider(
		log.WithProcessor(log.NewBatchProcessor(logExporter)),
		log.WithResource(res),
	)

	// 3. Set this logger provider as the global logger provider.
	//    The otelslog bridge will use this global provider to send logs.
	logglobal.SetLoggerProvider(loggerProvider)

	// 4. THIS IS THE CORRECTED PART: Create the otelslog.Handler.
	//    It doesn't wrap another handler. It *is* the handler.
	//    It needs a name (we can use the schema URL) and the LoggerProvider.
	handler := otelslog.NewHandler(res.SchemaURL(), otelslog.WithLoggerProvider(loggerProvider))

	// 5. Create a new slog.Logger with our OpenTelemetry handler.
	logger := slog.New(handler)

	// 6. Set the new logger as the default for the application.
	slog.SetDefault(logger)

	// Optional: Log a message to confirm initialization
	slog.Info("Logger initialized and configured to send to OTLP endpoint")

	// Return the shutdown function for the logger provider.
	return loggerProvider.Shutdown, nil
}
