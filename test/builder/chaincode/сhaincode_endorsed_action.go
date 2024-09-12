package chaincode

import (
	"github.com/golang/protobuf/proto" //nolint:staticcheck
	"github.com/hyperledger/fabric-protos-go/peer"
	"github.com/pkg/errors"
)

type ChaincodeEndorsedActionBuilder struct {
	action *peer.ChaincodeEndorsedAction
}

func NewChaincodeEndorsedActionBuilder() *ChaincodeEndorsedActionBuilder {
	return &ChaincodeEndorsedActionBuilder{
		action: &peer.ChaincodeEndorsedAction{},
	}
}

func (b *ChaincodeEndorsedActionBuilder) SetProposalResponsePayload(proposalHash []byte, extension []byte) *ChaincodeEndorsedActionBuilder {
	payload := &peer.ProposalResponsePayload{
		ProposalHash: proposalHash,
		Extension:    extension,
	}

	payloadBytes, err := proto.Marshal(payload)
	if err != nil {
		panic(errors.Errorf("Failed to marshal ProposalResponsePayload: %v\n", err))
	}
	b.action.ProposalResponsePayload = payloadBytes

	return b
}

func (b *ChaincodeEndorsedActionBuilder) AddEndorsement(signature, endorser []byte) *ChaincodeEndorsedActionBuilder {
	endorsement := &peer.Endorsement{
		Signature: signature,
		Endorser:  endorser,
	}
	b.action.Endorsements = append(b.action.Endorsements, endorsement)

	return b
}

func (b *ChaincodeEndorsedActionBuilder) Build() *peer.ChaincodeEndorsedAction {
	return b.action
}
