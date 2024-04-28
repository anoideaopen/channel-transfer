package transfer

import (
	"context"
	"crypto/tls"
	"errors"
	"net"

	"github.com/anoideaopen/channel-transfer/pkg/logger"
	"github.com/anoideaopen/channel-transfer/pkg/logger/grpclog"
	"github.com/anoideaopen/channel-transfer/pkg/middleware"
	"github.com/anoideaopen/channel-transfer/proto"
	"github.com/anoideaopen/glog"
	grpcmiddleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpcprom "github.com/grpc-ecosystem/go-grpc-middleware/providers/prometheus"
	grpcrecovery "github.com/grpc-ecosystem/go-grpc-middleware/recovery"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

const tcpNetwork = "tcp"

func runGRPC(ctx context.Context, transferServer *APIServer, tlsConfig *tls.Config, address string, accessToken string, grpcMetrics *grpcprom.ServerMetrics) error {
	log := glog.FromContext(ctx).With(logger.Labels{Component: logger.ComponentAPI}.Fields()...)

	listen, err := net.Listen(new(net.TCPAddr).Network(), address)
	if err != nil {
		return err
	}
	defer func() {
		if err := listen.Close(); err != nil {
			log.Errorf("Failed to close %s %s: %v", tcpNetwork, address, err)
		}
	}()

	unaryInterceptors := getUnaryInterceptors(log, accessToken)

	var serverOptions []grpc.ServerOption

	if tlsConfig != nil {
		serverOptions = append(serverOptions, grpc.Creds(credentials.NewTLS(tlsConfig)))
	}

	if grpcMetrics != nil {
		serverOptions = append(
			serverOptions,
			grpc.StreamInterceptor(grpcMetrics.StreamServerInterceptor()),
		)
		unaryInterceptors = append(unaryInterceptors, grpcMetrics.UnaryServerInterceptor())
	}

	serverOptions = append(
		serverOptions,
		grpc.UnaryInterceptor(
			grpcmiddleware.ChainUnaryServer(
				unaryInterceptors...,
			),
		),
	)

	grpcServer := grpc.NewServer(serverOptions...)

	proto.RegisterAPIServer(grpcServer, transferServer)
	if grpcMetrics != nil {
		grpcMetrics.InitializeMetrics(grpcServer)
	}

	go func() {
		<-ctx.Done()
		grpcServer.Stop()
	}()

	log.Infof("grpc listen on %s", address)

	err = grpcServer.Serve(listen)

	if !errors.Is(err, context.Canceled) {
		return err
	}

	return nil
}

func getUnaryInterceptors(log glog.Logger, accessToken string) []grpc.UnaryServerInterceptor {
	if accessToken == "" {
		return []grpc.UnaryServerInterceptor{
			grpclog.UnaryServerInterceptor(log),
			grpcrecovery.UnaryServerInterceptor(),
		}
	}
	return []grpc.UnaryServerInterceptor{
		middleware.NewGRPCAuthenticationInterceptor(accessToken),
		grpclog.UnaryServerInterceptor(log),
		grpcrecovery.UnaryServerInterceptor(),
	}
}
