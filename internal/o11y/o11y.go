package o11y

import (
	"context"
	"log/slog"
	"os"

	"github.com/prometheus/client_golang/prometheus"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/sdk/trace"
)

type Observability struct {
	Logger   *slog.Logger
	Tracer   *trace.TracerProvider
	Registry *prometheus.Registry
}

func Setup(ctx context.Context) (*Observability, func(), error) {
	// Initialize slog
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	// Initialize OpenTelemetry (with sampling)
	exporter, _ := otlptracehttp.New(ctx,
		otlptracehttp.WithInsecure(),
		otlptracehttp.WithEndpoint("localhost:4318"),
	)
	tp := trace.NewTracerProvider(
		trace.WithBatcher(exporter),
		trace.WithSampler(trace.ParentBased(
			trace.TraceIDRatioBased(1), // 1% sampling
		)),
	)
	otel.SetTracerProvider(tp)

	// Initialize Prometheus registry
	registry := prometheus.NewRegistry()

	cleanup := func() {
		tp.Shutdown(ctx)
	}

	return &Observability{
		Logger:   logger,
		Tracer:   tp,
		Registry: registry,
	}, cleanup, nil
}
