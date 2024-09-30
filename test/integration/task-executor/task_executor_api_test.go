package task_executor

import (
	"context"
	"fmt"
	"net"

	"github.com/anoideaopen/channel-transfer/proto"
	"github.com/anoideaopen/foundation/test/integration/cmn"
	"github.com/hyperledger/fabric/integration"
	"github.com/hyperledger/fabric/integration/nwo"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

// Task Executor API emulation structure
type taskExecutorAPI struct {
	proto.UnimplementedTaskExecutorAdapterServer
}

func (s *taskExecutorAPI) SubmitTransaction(
	ctx context.Context,
	_ *proto.TaskExecutorRequest,
) (*proto.TaskExecutorResponse, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return &proto.TaskExecutorResponse{
			Status:  proto.TaskExecutorResponse_STATUS_REJECTED,
			Message: "failed to retrieve metadata from context",
		}, nil
	}

	requestID := md.Get("requestID")
	if len(requestID) == 0 {
		return &proto.TaskExecutorResponse{
			Status:  proto.TaskExecutorResponse_STATUS_REJECTED,
			Message: "empty requestID in metadata from context",
		}, nil
	}

	return &proto.TaskExecutorResponse{
		Status:  proto.TaskExecutorResponse_STATUS_ACCEPTED,
		Message: fmt.Sprintf("request with ID %s submitted successfully", requestID[0]),
	}, nil
}

// StartTaskExecutor starts grpc server which listens batcher API on given port
func StartTaskExecutor() *grpc.Server {
	gRPCServer := grpc.NewServer()

	proto.RegisterTaskExecutorAdapterServer(gRPCServer, &taskExecutorAPI{})

	go func() {
		lis, err := net.Listen("tcp", fmt.Sprintf(":%d", taskExecutorPort()))
		if err != nil {
			panic(err)
		}

		if err = gRPCServer.Serve(lis); err != nil {
			panic(err)
		}
	}()

	return gRPCServer
}

// StopTaskExecutor stops given grpc server
func StopTaskExecutor(server *grpc.Server) {
	server.GracefulStop()
}

func taskExecutorHost() string {
	return "localhost"
}

func taskExecutorPort() uint16 {
	return uint16(integration.LifecyclePort)
}

func taskExecutorPorts() nwo.Ports {
	return nwo.Ports{cmn.GrpcPort: taskExecutorPort()}
}
