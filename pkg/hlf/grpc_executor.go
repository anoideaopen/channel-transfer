package hlf

import (
	"context"

	"github.com/anoideaopen/channel-transfer/proto"
	"github.com/google/uuid"
	"github.com/hyperledger/fabric-sdk-go/pkg/client/channel"
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/metadata"
)

// gRPCExecutor stores a gRPC client to work with external batcher service
type gRPCExecutor struct {
	Channel string
	Client  *grpc.ClientConn
}

// invoke sends a transaction request to an external batcher service instead of HLF
func (ex *gRPCExecutor) invoke(ctx context.Context, req channel.Request, _ []channel.RequestOption) (channel.Response, error) {
	adaptor := proto.NewHLFBatcherAdapterClient(ex.Client)

	ctx = metadata.NewOutgoingContext(ctx, metadata.Pairs("requestID", uuid.New().String()))

	request := &proto.HlfBatcherRequest{
		Channel:   ex.Channel,
		Chaincode: req.ChaincodeID,
		Method:    req.Fcn,
		Args:      req.Args,
	}

	if _, err := adaptor.SubmitTransaction(ctx, request); err != nil {
		return channel.Response{}, err
	}

	return channel.Response{}, nil
}

func (ex *gRPCExecutor) close() error {
	if ex.Client != nil && ex.Client.GetState() != connectivity.Shutdown {
		return ex.Client.Close()
	}

	return nil
}
