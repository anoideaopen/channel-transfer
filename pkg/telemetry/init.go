package telemetry

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"errors"
	"fmt"

	"github.com/anoideaopen/channel-transfer/pkg/config"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.37.0"
)

// InitTracing creates and registers globally a new TracerProvider.
func InitTracing(ctx context.Context, tracingCollector *config.Collector, serviceName string) error {
	if tracingCollector == nil {
		return errors.New("tracing collector configuration is nil")
	}

	// Set up propagator.
	prop := newPropagator()
	otel.SetTextMapPropagator(prop)

	exporter, err := newOtlpTracerExporter(ctx, tracingCollector)
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
func newOtlpTracerExporter(ctx context.Context, tracingCollector *config.Collector) (sdktrace.SpanExporter, error) {
	var (
		safetyOption = otlptracehttp.WithInsecure()
		headers      = map[string]string{}
	)

	options := []otlptracehttp.Option{
		otlptracehttp.WithEndpoint(tracingCollector.Endpoint),
	}

	if tracingCollector.TLSCA != "" {
		// If the header is not empty but there are no certificates, consider it an error
		tlsCfg, err := getTLSConfig(tracingCollector.TLSCA)
		if err != nil {
			return nil, fmt.Errorf("failed to get TLS config: %w", err)
		}
		safetyOption = otlptracehttp.WithTLSClientConfig(tlsCfg)
	}

	if tracingCollector.AuthorizationHeaderKey != "" &&
		tracingCollector.AuthorizationHeaderValue != "" {
		if tracingCollector.TLSCA == "" {
			return nil, errors.New("TLSCA must be set if authorization headers are provided")
		}
		headers = map[string]string{
			tracingCollector.AuthorizationHeaderKey: tracingCollector.AuthorizationHeaderValue,
		}
	}

	options = append(options,
		safetyOption,
		otlptracehttp.WithHeaders(headers),
	)

	traceExporter, err := otlptracehttp.New(ctx, options...)
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

func getTLSConfig(caCertsBase64 string) (*tls.Config, error) {
	caCertsBytes, err := base64.StdEncoding.DecodeString(caCertsBase64)
	if err != nil {
		return nil, fmt.Errorf("failed to decode TLS configuration: %w", err)
	}

	caCertPool := x509.NewCertPool()
	ok := caCertPool.AppendCertsFromPEM(caCertsBytes)
	if !ok {
		return nil, errors.New("failed to add CA certificates to CA cert pool")
	}

	tlsConfig := &tls.Config{
		RootCAs: caCertPool,
	}

	return tlsConfig, nil
}
