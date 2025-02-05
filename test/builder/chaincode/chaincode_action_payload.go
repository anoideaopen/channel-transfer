package chaincode

import (
	"github.com/go-errors/errors"
	"github.com/hyperledger/fabric-protos-go-apiv2/peer"
	"google.golang.org/protobuf/proto"
)

type ChaincodeActionPayloadBuilder struct {
	chaincodeActionPayload *peer.ChaincodeActionPayload
}

func NewChaincodeActionPayloadBuilder() *ChaincodeActionPayloadBuilder {
	return &ChaincodeActionPayloadBuilder{
		chaincodeActionPayload: &peer.ChaincodeActionPayload{},
	}
}

func (b *ChaincodeActionPayloadBuilder) SetChaincodeProposalPayload(inputPayload *peer.ChaincodeProposalPayload) *ChaincodeActionPayloadBuilder {
	payloadBytes, err := proto.Marshal(inputPayload)
	if err != nil {
		panic(errors.Errorf("Failed to marshal ChaincodeProposalPayload: %v\n", err))
	}

	b.chaincodeActionPayload.ChaincodeProposalPayload = payloadBytes
	return b
}

func (b *ChaincodeActionPayloadBuilder) SetEndorsedAction(endorsedAction *peer.ChaincodeEndorsedAction) *ChaincodeActionPayloadBuilder {
	b.chaincodeActionPayload.Action = endorsedAction
	return b
}

func (b *ChaincodeActionPayloadBuilder) Build() *peer.ChaincodeActionPayload {
	return b.chaincodeActionPayload
}
