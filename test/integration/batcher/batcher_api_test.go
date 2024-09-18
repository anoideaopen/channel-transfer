package batcher

import (
	"context"
	"fmt"
	"net"

	"github.com/anoideaopen/channel-transfer/proto"
	"github.com/hyperledger/fabric/integration"
	"google.golang.org/grpc"
)

// Batcher API emulation structure
type batcherAPI struct {
	proto.UnimplementedHLFBatcherAdapterServer
}

func (s *batcherAPI) SubmitTransaction(
	_ context.Context,
	_ *proto.HlfBatcherRequest,
) (*proto.HLFBatcherResponse, error) {
	return &proto.HLFBatcherResponse{
		Status: proto.Status_STATUS_ACCEPTED,
	}, nil
}

// StartBatcher starts grpc server which listens batcher API on given port
func StartBatcher() *grpc.Server {
	gRPCServer := grpc.NewServer()

	proto.RegisterHLFBatcherAdapterServer(gRPCServer, &batcherAPI{})

	go func() {
		lis, err := net.Listen("tcp", fmt.Sprintf(":%s", batcherPort()))
		if err != nil {
			panic(err)
		}

		if err = gRPCServer.Serve(lis); err != nil {
			panic(err)
		}
	}()

	return gRPCServer
}

// StopBatcher stops given grpc server
func StopBatcher(server *grpc.Server) {
	server.GracefulStop()
}

func batcherPort() string {
	return fmt.Sprintf("%d", integration.LifecyclePort)
}
