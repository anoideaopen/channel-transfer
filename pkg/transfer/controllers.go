package transfer

import (
	"context"

	"github.com/anoideaopen/channel-transfer/pkg/model"
)

// RequestController is responsible for implementing the processing of user and
// administrator requests to perform transfer operations from one channel to
// another.
//
//go:generate mockgen -source controllers.go -destination mock/controllers.go -package mock -mock_names RequestController=MockRequestController
type RequestController interface {
	// TransferKeep storage a request to transfer funds from one channel to
	// another.
	TransferKeep(context.Context, model.TransferRequest) error

	// TransferFetch retrieves the transfer by ID with updated status and the
	// result of processing the transfer request.
	TransferFetch(context.Context, model.ID) (model.TransferRequest, error)
}

//go:generate mockgen -source controllers.go -destination mock/controllers.go -package mock -mock_names BlockController=MockBlockController
type BlockController interface {
	// BlockSave storages the block of ledger data.
	BlockSave(context.Context, model.TransferBlock) error

	// BlockLoad retrieves the block of ledger data.
	BlockLoad(context.Context, model.ID) (model.TransferBlock, error)
}

//go:generate mockgen -source controllers.go -destination mock/controllers.go -package mock -mock_names CheckpointController=MockCheckpointController
type CheckpointController interface {
	// CheckpointSave saves the checkpoint of the processed data of the ledger channel.
	CheckpointSave(context.Context, model.Checkpoint) (model.Checkpoint, error)

	// CheckpointLoad retrieves the checkpoint of the processed data of the ledger channel.
	CheckpointLoad(context.Context, model.ID) (model.Checkpoint, error)
}
