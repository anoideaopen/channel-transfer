package telemetry

import (
	"context"
	"strings"

	"go.opentelemetry.io/otel/propagation"
	"google.golang.org/grpc/metadata"
)

type metadataCarrier struct {
	MD metadata.MD
}

func carrierFromMetadata(md metadata.MD) propagation.TextMapCarrier {
	return &metadataCarrier{MD: md}
}

func carrierFromContext(ctx context.Context) propagation.TextMapCarrier {
	carrier := metadataCarrier{MD: metadata.New(nil)}
	propagation.TraceContext{}.Inject(ctx, &carrier)
	return &carrier
}

func (c *metadataCarrier) Set(key, value string) {
	c.MD.Set(key, value)
}

func (c *metadataCarrier) Get(key string) string {
	return strings.Join(c.MD.Get(key), "")
}

func (c *metadataCarrier) Keys() []string {
	keys := make([]string, 0, len(c.MD))
	for k := range c.MD {
		keys = append(keys, k)
	}
	return keys
}
