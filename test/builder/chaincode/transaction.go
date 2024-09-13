package chaincode

import (
	"github.com/go-errors/errors"
	"github.com/golang/protobuf/proto" //nolint:staticcheck
	"github.com/hyperledger/fabric-protos-go/peer"
)

type TransactionBuilder struct {
	transaction *peer.Transaction
}

func NewTransactionBuilder() *TransactionBuilder {
	return &TransactionBuilder{
		transaction: &peer.Transaction{
			Actions: []*peer.TransactionAction{},
		},
	}
}

func (b *TransactionBuilder) AddAction(header []byte, payload *peer.ChaincodeActionPayload) *TransactionBuilder {
	payloadBytes, err := proto.Marshal(payload)
	if err != nil {
		panic(errors.Errorf("Failed to marshal ChaincodeActionPayload: %v\n", err))
	}

	action := &peer.TransactionAction{
		Header:  header,
		Payload: payloadBytes,
	}
	b.transaction.Actions = append(b.transaction.Actions, action)

	return b
}

func (b *TransactionBuilder) Build() *peer.Transaction {
	return b.transaction
}

func (b *TransactionBuilder) Marshal() []byte {
	transactionBytes, err := proto.Marshal(b.transaction)
	if err != nil {
		panic(errors.Errorf("Failed to marshal transaction: %v\n", err))
	}

	return transactionBytes
}
