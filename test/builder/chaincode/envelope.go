package chaincode

import (
	"github.com/go-errors/errors"
	"github.com/golang/protobuf/proto" //nolint:staticcheck
	"github.com/hyperledger/fabric-protos-go/common"
	"github.com/hyperledger/fabric-protos-go/peer"
)

type EnvelopeBuilder struct {
	envelope *common.Envelope
	payload  *common.Payload
	header   *common.Header
}

func NewEnvelopeBuilder() *EnvelopeBuilder {
	return &EnvelopeBuilder{
		envelope: &common.Envelope{},
		payload:  &common.Payload{},
		header:   &common.Header{},
	}
}

func (b *EnvelopeBuilder) SetChannelHeader(channelHeader *common.ChannelHeader) *EnvelopeBuilder {
	channelHeaderBytes, err := proto.Marshal(channelHeader)
	if err != nil {
		panic(errors.Errorf("failed to marshal channel header: %v", err))
	}
	b.header.ChannelHeader = channelHeaderBytes

	return b
}

func (b *EnvelopeBuilder) SetSignatureHeader(signatureHeader *common.SignatureHeader) *EnvelopeBuilder {
	signatureHeaderBytes, err := proto.Marshal(signatureHeader)
	if err != nil {
		panic(errors.Errorf("failed to marshal signature header: %v", err))
	}
	b.header.SignatureHeader = signatureHeaderBytes

	return b
}

func (b *EnvelopeBuilder) SetData(transaction *peer.Transaction) *EnvelopeBuilder {
	transactionBytes, err := proto.Marshal(transaction)
	if err != nil {
		panic(errors.Errorf("failed to marshal transaction: %v", err))
	}
	b.payload.Data = transactionBytes

	return b
}

func (b *EnvelopeBuilder) SetSignature(signature []byte) *EnvelopeBuilder {
	b.envelope.Signature = signature
	return b
}

func (b *EnvelopeBuilder) Build() *common.Envelope {
	b.payload.Header = b.header

	payloadBytes, err := proto.Marshal(b.payload)
	if err != nil {
		panic(errors.Errorf("failed to marshal payload: %v", err))
	}
	b.envelope.Payload = payloadBytes

	return b.envelope
}
