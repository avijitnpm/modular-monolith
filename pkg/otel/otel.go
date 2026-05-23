package otel

import (
	"context"
	"log/slog"
	"time"

	"github.com/avijitnpm/modular-monolith/internal/config"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

type ShutdownFunc func(context.Context) error

func Init(
	ctx context.Context,
	cfg *config.Config,
	logger *slog.Logger,
) ShutdownFunc {

	if !cfg.OTEL.Enabled {
		return noopShutdown
	}

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	opts := []otlptracehttp.Option{}

	if cfg.OTEL.Endpoint != "" {
		opts = append(opts, otlptracehttp.WithEndpointURL(cfg.OTEL.Endpoint))
	}

	if cfg.OTEL.Insecure {
		opts = append(opts, otlptracehttp.WithInsecure())
	}

	exporter, err := otlptracehttp.New(ctx, opts...)
	if err != nil {
		logger.Error(
			"otel exporter initialization failed",
			"error", err,
		)

		return noopShutdown
	}

	res := resource.NewWithAttributes(
		"",
		attribute.String("service.name", cfg.OTEL.ServiceName),
	)

	provider := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
	)

	otel.SetTracerProvider(provider)
	otel.SetTextMapPropagator(
		propagation.NewCompositeTextMapPropagator(
			propagation.TraceContext{},
			propagation.Baggage{},
		),
	)

	logger.Info(
		"otel tracing initialized",
		"service", cfg.OTEL.ServiceName,
		"endpoint", cfg.OTEL.Endpoint,
	)

	return provider.Shutdown
}

func noopShutdown(context.Context) error {
	return nil
}
