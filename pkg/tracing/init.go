package tracing

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

// InitTracing creates and registers globally a new TracerProvider.
func InitTracing(ctx context.Context, tracingCollectorEndpoint string, tracingCollectorBasicAuthToken string, serviceName string) error {
	// Set up propagator.
	prop := newPropagator()
	otel.SetTextMapPropagator(prop)

	exporter, err := newOtlpTracerExporter(ctx, tracingCollectorEndpoint, tracingCollectorBasicAuthToken)
	if err != nil {
		return fmt.Errorf("initialize trace exporter: %w", err)
	}

	// Set up trace provider.
	tracerProvider, err := newTracerProvider(exporter, serviceName)
	if err != nil {
		return fmt.Errorf("initialize trace provider: %w", err)
	}
	otel.SetTracerProvider(tracerProvider)
	return nil
}

// newOtlpTraceExporter create and start new OTLP trace exporter
func newOtlpTracerExporter(ctx context.Context, tracingCollectorEndpoint string, tracingCollectorBasicAuthToken string) (sdktrace.SpanExporter, error) {
	options := []otlptracegrpc.Option{
		otlptracegrpc.WithInsecure(),
		otlptracegrpc.WithEndpoint(tracingCollectorEndpoint),
	}

	if tracingCollectorBasicAuthToken != "" {
		headers := map[string]string{
			"Authorization": "Basic " + tracingCollectorBasicAuthToken,
		}
		options = append(options, otlptracegrpc.WithHeaders(headers))
	}

	traceExporter, err := otlptracegrpc.New(ctx, options...)
	if err != nil {
		return nil, fmt.Errorf("failed to create new OTLP trace exporter: %w", err)
	}
	return traceExporter, nil
}

func newTracerProvider(exp sdktrace.SpanExporter, serviceName string) (*sdktrace.TracerProvider, error) {
	r, err := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceNameKey.String(serviceName),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("merge resource: %w", err)
	}

	return sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exp),
		sdktrace.WithResource(r),
	), nil
}

func newPropagator() propagation.TextMapPropagator {
	return propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	)
}
