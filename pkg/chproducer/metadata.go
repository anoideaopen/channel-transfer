package chproducer

import (
	"context"

	"github.com/anoideaopen/channel-transfer/pkg/model"
	"google.golang.org/grpc/metadata"
)

func (h *Handler) appendTransferMetadataToContext(ctx context.Context, transferID model.ID) (context.Context, error) {
	md, err := h.metadataStorage.MetadataLoad(ctx, transferID)
	if err != nil {
		return nil, err
	}

	if len(md.TraceID) > 0 {
		ctx = metadata.AppendToOutgoingContext(ctx, "trace_id", md.TraceID)
	}

	if len(md.SpanID) > 0 {
		ctx = metadata.AppendToOutgoingContext(ctx, "span_id", md.SpanID)
	}

	return ctx, nil
}
