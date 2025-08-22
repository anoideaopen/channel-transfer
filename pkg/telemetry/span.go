package telemetry

import (
	"context"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

func FinishSpan(span trace.Span, err error) {
	if span == nil {
		return
	}
	if err == nil {
		span.SetStatus(codes.Ok, "")
	} else {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	}
	span.End()
}

type TraceableObject interface {
	GetTraceAttributes() []attribute.KeyValue
}

// StartSpan creates a new span with the provided context, tracer and [TransferDataGetter].
// [TransferDataGetter] is used to extract transfer-related attributes for the span.
// It is a client responsibility to call [FinishSpan] when the span is no longer needed.
func StartSpan(
	ctx context.Context,
	tracer trace.Tracer,
	spanName string,
	req TraceableObject,
	attributes ...attribute.KeyValue,
) (context.Context, trace.Span) {
	attributes = append(attributes,
		req.GetTraceAttributes()...,
	)
	return tracer.Start(ctx, //nolint:spancheck
		spanName,
		trace.WithAttributes(
			attributes...,
		),
	)
}
