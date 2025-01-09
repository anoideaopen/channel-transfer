package model

import (
	"encoding/json"

	"github.com/anoideaopen/channel-transfer/pkg/data"
	"github.com/anoideaopen/foundation/proto"
	"github.com/hyperledger/fabric-protos-go/peer"
)

type CompositeKeysPrefixes struct {
	BatchPrefix string
}

type Transaction struct {
	Channel        string
	BlockNum       uint64
	TxID           string
	FuncName       string
	Args           [][]byte
	TimeNs         uint64
	ValidationCode int32
	IsExecutorTask bool
	BatchResponse  *proto.TxResponse
	Response       *peer.Response
}

type TransferBlock struct {
	Channel      ID
	Transfer     ID
	Transactions []Transaction
}

func (tb *TransferBlock) MarshalBinary() (data []byte, err error) {
	return json.Marshal(tb)
}

func (tb *TransferBlock) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, tb)
}

func (tb *TransferBlock) Clone() data.Object {
	tbCopy := *tb
	return &tbCopy
}

func (tb *TransferBlock) Instance() data.Type {
	return data.InstanceOf(tb)
}

type BlockData struct {
	BlockNum uint64
	Txs      []Transaction
}

func (d *BlockData) IsEmpty() bool {
	return len(d.Txs) == 0
}

func (d *BlockData) ItemsCount() uint {
	return uint(len(d.Txs))
}
