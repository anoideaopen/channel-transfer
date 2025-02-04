package chaincode

import (
	"github.com/go-errors/errors"
	"github.com/hyperledger/fabric-protos-go-apiv2/peer"
	"google.golang.org/protobuf/proto"
)

type ChaincodeInvocationSpecBuilder struct {
	chaincodeSpec *peer.ChaincodeInvocationSpec
}

func NewChaincodeInvocationSpecBuilder() *ChaincodeInvocationSpecBuilder {
	return &ChaincodeInvocationSpecBuilder{
		chaincodeSpec: &peer.ChaincodeInvocationSpec{},
	}
}

func (b *ChaincodeInvocationSpecBuilder) SetChaincodeSpec(spec *peer.ChaincodeSpec) *ChaincodeInvocationSpecBuilder {
	b.chaincodeSpec.ChaincodeSpec = spec
	return b
}

func (b *ChaincodeInvocationSpecBuilder) Build() *peer.ChaincodeInvocationSpec {
	return b.chaincodeSpec
}

func (b *ChaincodeInvocationSpecBuilder) Marshal() []byte {
	actionBytes, err := proto.Marshal(b.chaincodeSpec)
	if err != nil {
		panic(errors.Errorf("Failed to marshal ChaincodeInvocationSpec: %v\n", err))
	}

	return actionBytes
}
