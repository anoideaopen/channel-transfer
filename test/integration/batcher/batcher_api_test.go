package batcher

import (
	"context"
	"fmt"
	"net"

	"github.com/anoideaopen/channel-transfer/proto"
	"github.com/hyperledger/fabric/integration"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

// Batcher API emulation structure
type batcherAPI struct {
	proto.UnimplementedHLFBatcherAdapterServer
}

func (s *batcherAPI) SubmitTransaction(
	ctx context.Context,
	_ *proto.HlfBatcherRequest,
) (*proto.HLFBatcherResponse, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return &proto.HLFBatcherResponse{
			Status:  proto.HLFBatcherResponse_STATUS_REJECTED,
			Message: "failed to retrieve metadata from context",
		}, nil
	}

	requestID := md.Get("requestID")
	if len(requestID) == 0 {
		return &proto.HLFBatcherResponse{
			Status:  proto.HLFBatcherResponse_STATUS_REJECTED,
			Message: "empty requestID in metadata from context",
		}, nil
	}

	return &proto.HLFBatcherResponse{
		Status:  proto.HLFBatcherResponse_STATUS_ACCEPTED,
		Message: fmt.Sprintf("request with ID %s submitted successfully", requestID[0]),
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
