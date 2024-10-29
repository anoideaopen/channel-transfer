package telemetry

import (
	"context"

	"github.com/anoideaopen/channel-transfer/pkg/model"
	"google.golang.org/grpc/metadata"
)

// TransferMetadataFromContext extracts transfer metadata from context
func TransferMetadataFromContext(ctx context.Context) model.TransferMetadata {
	metadataModel := model.TransferMetadata{}

	carrier := carrierFromContext(ctx)
	for _, k := range carrier.Keys() {
		metadataModel[k] = carrier.Get(k)
	}

	return metadataModel
}

// AppendTransferMetadataToContext appends transfer metadata to context
func AppendTransferMetadataToContext(ctx context.Context, md model.TransferMetadata) context.Context {
	carrier := &metadataCarrier{MD: metadata.New(nil)}

	for k, v := range md {
		carrier.Set(k, v)
	}

	return metadata.NewOutgoingContext(ctx, carrier.MD)
}

// TransientMapFromContext prepares tracing context for using in transient map
func TransientMapFromContext(ctx context.Context) map[string][]byte {
	transientMap := make(map[string][]byte)

	md, ok := metadata.FromOutgoingContext(ctx)
	if !ok {
		return transientMap
	}

	carrier := carrierFromMetadata(md)

	for _, k := range carrier.Keys() {
		transientMap[k] = []byte(carrier.Get(k))
	}

	return transientMap
}
