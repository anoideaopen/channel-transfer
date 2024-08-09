package model

import (
	"encoding/json"

	"github.com/anoideaopen/channel-transfer/pkg/data"
)

// ID is defined as a separate data type for requests to get the status by ID.
type ID string

// TransferResult is the result of processing a request to transfer funds from
// one channel to another. The statuses are symmetrical to [proto].
// TODO: Custom statuses should be added later.
type TransferResult struct {
	Status  string
	Message string
}

type TransferItem struct {
	Token  string `json:"token"`
	Amount string `json:"amount"`
}

// TransferRequest contains the internal representation of a request to transfer
// funds from one channel to another. This structure is filled from the request
// and enters the queue for processing.
type TransferRequest struct {
	TransferResult

	Request   ID
	Transfer  ID
	User      ID
	Method    string
	Chaincode string
	Channel   string
	Nonce     string
	PublicKey string
	Sign      string
	To        string
	Token     string
	Amount    string
	Items     []TransferItem
}

func (tr *TransferRequest) MarshalBinary() (data []byte, err error) {
	return json.Marshal(tr)
}

func (tr *TransferRequest) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, tr)
}

// Clone should create an exact copy of the object, located in a different
// memory location from the original. This is necessary in case of cache
// optimization to avoid marshalling the object in some cases. Clone is also
// used as a template for finding an object in the repository.
func (tr *TransferRequest) Clone() data.Object {
	trCopy := *tr
	return &trCopy
}

// Instance should return a unique object type to share namespace between
// the stored data. In the simplest case, you can return the type name via
// InstanceOf, but keep in mind that you need to preserve compatibility or
// provide for migration when refactoring.
func (tr *TransferRequest) Instance() data.Type {
	return data.InstanceOf(tr)
}
