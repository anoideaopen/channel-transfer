package app

import (
	"fmt"
	"log/slog"
	"net"

	"github.com/anoideaopen/channel-transfer/test/external-services/batcher/grpc/batcher"
	"google.golang.org/grpc"
)

type App struct {
	logger     *slog.Logger
	gRPCServer *grpc.Server
	port       int
}

func New(logger *slog.Logger, port int) *App {
	gRPCServer := grpc.NewServer()

	batcher.Register(gRPCServer)

	return &App{
		logger:     logger,
		gRPCServer: gRPCServer,
		port:       port,
	}
}

func (a *App) Start() {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", a.port))
	if err != nil {
		a.logger.Error("failed to listen: %v", err)
		panic(err)
	}

	a.logger.Info("listening on port %d", a.port)

	if err = a.gRPCServer.Serve(lis); err != nil {
		a.logger.Error("failed to serve: %v", err)
		panic(err)
	}
}

func (a *App) Stop() {
	a.logger.Info("stopping")
	a.gRPCServer.GracefulStop()
}
