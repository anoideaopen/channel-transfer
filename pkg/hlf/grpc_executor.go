package hlf

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"time"

	"github.com/anoideaopen/channel-transfer/proto"
	"github.com/hyperledger/fabric-sdk-go/pkg/client/channel"
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
)

// gRPCExecutor stores a gRPC client to work with external batcher service
type gRPCExecutor struct {
	Channel string
	Client  *grpc.ClientConn
}

// invoke sends a transaction request to an external batcher service instead of HLF
func (ex *gRPCExecutor) invoke(ctx context.Context, req channel.Request, _ []channel.RequestOption) (channel.Response, error) {
	adaptor := proto.NewHLFBatcherAdapterClient(ex.Client)

	requestID, err := computeRequestID(req)
	if err != nil {
		return channel.Response{}, err
	}

	request := &proto.HlfBatcherRequest{
		Channel:          ex.Channel,
		Chaincode:        req.ChaincodeID,
		Method:           req.Fcn,
		BatcherRequestId: requestID,
		Args:             req.Args,
		TraceId:          nil,
		SpanId:           nil,
	}

	_, err = adaptor.SubmitTransaction(ctx, request)
	if err != nil {
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

func computeRequestID(req channel.Request) (string, error) {
	b, err := json.Marshal(req)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, 8)
	binary.LittleEndian.PutUint64(nonce, uint64(time.Now().UnixMilli()))
	b = append(nonce, b...)

	digest := sha256.Sum256(b)
	return hex.EncodeToString(digest[:]), nil
}
