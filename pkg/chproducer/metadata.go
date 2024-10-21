package chproducer

import (
	"context"

	"github.com/anoideaopen/channel-transfer/pkg/model"
	"google.golang.org/grpc/metadata"
)

func (h *Handler) appendTransferMetadataToContext(ctx context.Context, transferID model.ID) context.Context {
	md, err := h.metadataStorage.MetadataLoad(ctx, transferID)
	if err != nil {
		h.log.Warningf("couldn't load metadata for transfer %s from storage", transferID)
		return ctx
	}

	if len(md.TraceID) > 0 {
		ctx = metadata.AppendToOutgoingContext(ctx, "trace_id", md.TraceID)
	}

	if len(md.SpanID) > 0 {
		ctx = metadata.AppendToOutgoingContext(ctx, "span_id", md.SpanID)
	}

	return ctx
}
