package telemetry

import (
	"context"

	"google.golang.org/grpc/metadata"
)

// TransientMapFromContext prepares tracing context for using in transient map
func TransientMapFromContext(ctx context.Context) map[string][]byte {
	transientMap := make(map[string][]byte)

	md, ok := metadata.FromOutgoingContext(ctx)
	if !ok {
		return transientMap
	}

	carrier := CarrierFromMetadata(md)

	for _, k := range carrier.Keys() {
		transientMap[k] = []byte(carrier.Get(k))
	}

	return transientMap
}
