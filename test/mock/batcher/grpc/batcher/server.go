package batcher

import (
	"context"

	"github.com/anoideaopen/channel-transfer/proto"
	"google.golang.org/grpc"
)

type serverAPI struct {
	proto.UnimplementedHLFBatcherAdapterServer
}

func Register(gRPC *grpc.Server) {
	proto.RegisterHLFBatcherAdapterServer(gRPC, &serverAPI{})
}

func (s *serverAPI) SubmitTransaction(
	_ context.Context,
	_ *proto.HlfBatcherRequest,
) (*proto.HLFBatcherResponse, error) {
	return &proto.HLFBatcherResponse{
		Status: proto.Status_STATUS_ACCEPTED,
	}, nil
}
