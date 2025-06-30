package hlf

import (
	"context"

	"github.com/anoideaopen/channel-transfer/pkg/tracing"
	"github.com/anoideaopen/channel-transfer/proto"
	"github.com/anoideaopen/glog"
	"github.com/google/uuid"
	"github.com/hyperledger/fabric-sdk-go/pkg/client/channel"
	"go.opentelemetry.io/otel/attribute"
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
	var err error
	ctx, span := tracer.Start(ctx, "hlfexecutor: invoke gRPC")
	defer func() {
		tracing.FinishSpan(span, err)
	}()
	log := glog.FromContext(ctx)
	log.Set(glog.Field{K: "transfer.span.context", V: span.SpanContext()})
	var argsListForTracing string
	for _, arg := range req.Args {
		argsListForTracing += string(arg) + ", "
	}
	span.SetAttributes(
		attribute.String("invoke.args", argsListForTracing),
		attribute.String("invoke.method", req.Fcn),
	)
	log.Set(glog.Field{K: "invoke.args", V: argsListForTracing})
	log.Set(glog.Field{K: "invoke.method", V: req.Fcn})
	log.Debug("invoke gRPC")

	adaptor := proto.NewTaskExecutorAdapterClient(ex.Client)

	ctx = metadata.AppendToOutgoingContext(ctx, "requestID", uuid.New().String())

	request := &proto.TaskExecutorRequest{
		Channel:   ex.Channel,
		Chaincode: req.ChaincodeID,
		Method:    req.Fcn,
		Args:      req.Args,
	}

	if _, err = adaptor.SubmitTransaction(ctx, request); err != nil {
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
