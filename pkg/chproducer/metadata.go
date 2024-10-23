package chproducer

import (
	"context"

	"github.com/anoideaopen/channel-transfer/pkg/model"
	"github.com/anoideaopen/channel-transfer/pkg/telemetry"
)

func (h *Handler) appendTransferMetadataToContext(ctx context.Context, transferID model.ID) context.Context {
	md, err := h.metadataStorage.MetadataLoad(ctx, transferID)
	if err != nil {
		h.log.Warningf("couldn't load metadata for transfer %s from storage", transferID)
		return ctx
	}

	carrier := telemetry.NewCarrier()
	for k, v := range md {
		carrier.Set(k, v)
	}

	return telemetry.AppendCarrierToContext(ctx, carrier)
}
