package telemetry

import (
	"context"

	"go.opentelemetry.io/otel/propagation"
	"google.golang.org/grpc/metadata"
)

func AppendCarrierToContext(ctx context.Context, carrier propagation.TextMapCarrier) context.Context {
	if mdCarrier, ok := carrier.(*metadataCarrier); ok {
		ctx = metadata.NewOutgoingContext(ctx, mdCarrier.MD)
	}
	return ctx
}
