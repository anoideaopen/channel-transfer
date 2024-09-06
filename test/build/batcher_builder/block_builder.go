package batcher_builder

import (
	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric-protos-go/common"
	"github.com/pkg/errors"
)

type BlockBuilder struct {
	block *common.Block
}

func NewBlockBuilder() *BlockBuilder {
	return &BlockBuilder{
		block: &common.Block{
			Header:   &common.BlockHeader{},
			Data:     &common.BlockData{},
			Metadata: &common.BlockMetadata{},
		},
	}
}

func (b *BlockBuilder) SetHeader(number uint64, previousHash, dataHash []byte) *BlockBuilder {
	b.block.Header = &common.BlockHeader{
		Number:       number,
		PreviousHash: previousHash,
		DataHash:     dataHash,
	}

	return b
}

func (b *BlockBuilder) AddData(data *common.Envelope) *BlockBuilder {
	dataBytes, err := proto.Marshal(data)
	if err != nil {
		panic(errors.Errorf("failed to marshal data: %w", err))
	}

	b.block.Data.Data = append(b.block.Data.Data, dataBytes)

	return b
}

func (b *BlockBuilder) AddMetadata(index int, value []byte, signature []byte, signatureHeader []byte, identifierHeader []byte) *BlockBuilder {
	if len(b.block.Metadata.Metadata) <= index {
		b.block.Metadata.Metadata = append(b.block.Metadata.Metadata, make([][]byte, index-len(b.block.Metadata.Metadata)+1)...)
	}

	metadata := &common.Metadata{
		Value:      value,
		Signatures: []*common.MetadataSignature{},
	}

	metadata.Signatures = append(metadata.Signatures, &common.MetadataSignature{
		Signature:        signature,
		SignatureHeader:  signatureHeader,
		IdentifierHeader: identifierHeader,
	})

	if value == nil {
		metadata = &common.Metadata{}
		b.block.Metadata.Metadata[index] = []byte{0}
	} else {
		metadataBytes, err := proto.Marshal(metadata)
		if err != nil {
			panic(errors.Errorf("failed to marshal metadata: %v\n", err))
		}
		b.block.Metadata.Metadata[index] = metadataBytes
	}

	return b
}

func (b *BlockBuilder) Build() *common.Block {
	return b.block
}
