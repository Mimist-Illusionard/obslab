package trace

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/getsentry/sentry-go"
	sentryotel "github.com/getsentry/sentry-go/otel"
	sentryotlp "github.com/getsentry/sentry-go/otel/otlp"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.43.0"
)

const (
	BackendJaeger = "jaeger"
	BackendSentry = "sentry"
)

type Provider struct {
	tp            *sdktrace.TracerProvider
	sentryEnabled bool
}

func New(ctx context.Context) (*Provider, error) {
	backend := strings.ToLower(getEnv("TRACE_BACKEND", BackendJaeger))
	serviceName := getEnv("OTEL_SERVICE_NAME", "redditclone")

	var (
		exporter sdktrace.SpanExporter
		err      error
	)

	provider := &Provider{}

	switch backend {
	case BackendJaeger:
		exporter, err = newJaegerExporter(ctx)

	case BackendSentry:
		exporter, err = newSentryExporter(ctx)
		provider.sentryEnabled = true

	default:
		return nil, fmt.Errorf("unknown trace backend: %q", backend)
	}

	if err != nil {
		return nil, err
	}

	res, err := resource.New(ctx, resource.WithAttributes(
		semconv.ServiceNameKey.String(serviceName),
	),
	)
	if err != nil {
		return nil, fmt.Errorf("create resource: %w", err)
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
	)

	otel.SetTracerProvider(tp)

	otel.SetTextMapPropagator(
		propagation.NewCompositeTextMapPropagator(
			propagation.TraceContext{},
			propagation.Baggage{},
		),
	)

	provider.tp = tp

	return provider, nil
}

func newJaegerExporter(ctx context.Context) (sdktrace.SpanExporter, error) {
	endpoint := getEnv("OTEL_EXPORTER_OTLP_ENDPOINT", "jaeger:4317")

	exporter, err := otlptracegrpc.New(
		ctx,
		otlptracegrpc.WithEndpoint(endpoint),
		otlptracegrpc.WithInsecure(),
	)
	if err != nil {
		return nil, fmt.Errorf(
			"create jaeger exporter: %w",
			err,
		)
	}

	return exporter, nil
}

func newSentryExporter(ctx context.Context) (sdktrace.SpanExporter, error) {
	dsn := os.Getenv("SENTRY_DSN")

	if dsn == "" {
		return nil, errors.New("SENTRY_DSN is required for sentry tracing")
	}

	err := sentry.Init(
		sentry.ClientOptions{
			Dsn: dsn,

			EnableTracing:    true,
			TracesSampleRate: 1.0,

			Environment: getEnv("SENTRY_ENVIRONMENT", "development"),

			Integrations: func(integrations []sentry.Integration) []sentry.Integration {
				return append(integrations,
					sentryotel.NewOtelIntegration(),
				)
			},
		},
	)
	if err != nil {
		return nil, fmt.Errorf("init sentry: %w", err)
	}

	exporter, err := sentryotlp.NewTraceExporter(ctx, dsn)

	if err != nil {
		return nil, fmt.Errorf("create sentry otlp exporter: %w", err)
	}

	return exporter, nil
}

func (p *Provider) Shutdown(
	ctx context.Context,
) error {
	err := p.tp.Shutdown(ctx)

	if p.sentryEnabled {
		sentry.Flush(2 * time.Second)
	}

	return err
}

func getEnv(key, fallback string) string {
	value := os.Getenv(key)

	if value == "" {
		return fallback
	}

	return value
}
