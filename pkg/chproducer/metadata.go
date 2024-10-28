package chproducer

import (
	"context"

	"github.com/anoideaopen/channel-transfer/pkg/model"
	"github.com/anoideaopen/channel-transfer/pkg/telemetry"
)

func (h *Handler) appendTransferMetadataToContext(ctx context.Context, metadata model.TransferMetadata) context.Context {
	carrier := telemetry.NewCarrier()
	for k, v := range metadata {
		carrier.Set(k, v)
	}

	return telemetry.AppendCarrierToContext(ctx, carrier)
}
