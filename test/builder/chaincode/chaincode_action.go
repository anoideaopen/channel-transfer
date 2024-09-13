package chaincode

import (
	"github.com/go-errors/errors"
	"github.com/golang/protobuf/proto" //nolint:staticcheck
	"github.com/hyperledger/fabric-protos-go/ledger/rwset"
	"github.com/hyperledger/fabric-protos-go/peer"
)

type ChaincodeActionBuilder struct {
	chaincodeAction *peer.ChaincodeAction
}

func NewChaincodeActionBuilder() *ChaincodeActionBuilder {
	return &ChaincodeActionBuilder{
		chaincodeAction: &peer.ChaincodeAction{},
	}
}

func (b *ChaincodeActionBuilder) SetResults(results *rwset.TxReadWriteSet) *ChaincodeActionBuilder {
	resultBytes, err := proto.Marshal(results)
	if err != nil {
		panic(errors.Errorf("Failed to marshal TxReadWriteSet: %v\n", err))
	}
	b.chaincodeAction.Results = resultBytes

	return b
}

func (b *ChaincodeActionBuilder) SetEvents(events []byte) *ChaincodeActionBuilder {
	b.chaincodeAction.Events = events
	return b
}

func (b *ChaincodeActionBuilder) SetResponse(response *peer.Response) *ChaincodeActionBuilder {
	b.chaincodeAction.Response = response
	return b
}

func (b *ChaincodeActionBuilder) SetChaincodeID(chaincodeID *peer.ChaincodeID) *ChaincodeActionBuilder {
	b.chaincodeAction.ChaincodeId = chaincodeID
	return b
}

func (b *ChaincodeActionBuilder) Build() *peer.ChaincodeAction {
	return b.chaincodeAction
}

func (b *ChaincodeActionBuilder) Marshal() []byte {
	actionBytes, err := proto.Marshal(b.chaincodeAction)
	if err != nil {
		panic(errors.Errorf("Failed to marshal ChaincodeAction: %v\n", err))
	}

	return actionBytes
}
