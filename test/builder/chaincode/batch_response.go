package chaincode

import (
	fpb "github.com/anoideaopen/foundation/proto"
	"github.com/go-errors/errors"
	"google.golang.org/protobuf/proto"
)

type BatchResponseBuilder struct {
	batchResponse *fpb.BatchResponse
}

func NewBatchResponseBuilder() *BatchResponseBuilder {
	return &BatchResponseBuilder{
		batchResponse: &fpb.BatchResponse{},
	}
}

func (b *BatchResponseBuilder) AddTxResponse(txResponse *fpb.TxResponse) *BatchResponseBuilder {
	b.batchResponse.TxResponses = append(b.batchResponse.TxResponses, txResponse)
	return b
}

func (b *BatchResponseBuilder) Build() *fpb.BatchResponse {
	return b.batchResponse
}

func (b *BatchResponseBuilder) Marshal() []byte {
	batchResponseBytes, err := proto.Marshal(b.batchResponse)
	if err != nil {
		panic(errors.Errorf("Failed to marshal ChaincodeAction: %v\n", err))
	}

	return batchResponseBytes
}
