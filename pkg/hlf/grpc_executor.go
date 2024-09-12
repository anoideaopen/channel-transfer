package hlf

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"

	"github.com/anoideaopen/channel-transfer/proto"
	"github.com/hyperledger/fabric-sdk-go/pkg/client/channel"
	"github.com/hyperledger/fabric/common/crypto"
	"google.golang.org/grpc"
)

type gRPCExecutor struct {
	Channel string
	Client  *grpc.ClientConn
}

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

func computeRequestID(req channel.Request) (string, error) {
	nonce, err := crypto.GetRandomNonce()
	if err != nil {
		return "", err
	}

	b, err := json.Marshal(req)
	if err != nil {
		return "", err
	}

	b = append(nonce, b...)

	digest := sha256.Sum256(b)
	return hex.EncodeToString(digest[:]), nil
}
