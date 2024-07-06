package transfer

import (
	"context"

	"github.com/anoideaopen/channel-transfer/pkg/config"
	"github.com/anoideaopen/channel-transfer/pkg/data/redis"
	"github.com/anoideaopen/channel-transfer/pkg/model"
	grpcprom "github.com/grpc-ecosystem/go-grpc-middleware/providers/prometheus"
	"golang.org/x/sync/errgroup"
)

// Execute - Run grpc and http server
func Execute(
	ctx context.Context,
	group *errgroup.Group,
	cfg *config.ListenAPI,
	channels []string,
	output chan model.TransferRequest,
	storage *redis.Storage,
	grpcMetrics *grpcprom.ServerMetrics,
) error {
	tlsConfig := cfg.TLSConfig()

	apiServer := NewAPIServer(ctx, output, NewRequest(storage), channels)

	group.Go(func() error {
		return runGRPC(ctx, apiServer, tlsConfig, cfg.AddressGRPC, cfg.AccessToken, grpcMetrics)
	})

	group.Go(func() error {
		return runHTTP(ctx, tlsConfig, cfg.AddressHTTP, cfg.AddressGRPC, grpcMetrics != nil)
	})

	return nil
}
