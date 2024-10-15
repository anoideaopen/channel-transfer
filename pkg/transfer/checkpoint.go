package transfer

import (
	"context"
	"fmt"

	"github.com/anoideaopen/channel-transfer/pkg/data"
	"github.com/anoideaopen/channel-transfer/pkg/data/redis"
	"github.com/anoideaopen/channel-transfer/pkg/model"
)

type BlockCheckpoint struct {
	storage *redis.Storage
}

func NewBlockCheckpoint(storage *redis.Storage) *BlockCheckpoint {
	return &BlockCheckpoint{
		storage: storage,
	}
}

func (ckp *BlockCheckpoint) CheckpointSave(ctx context.Context, checkpoint model.Checkpoint) (model.Checkpoint, error) {
	if checkpoint.Channel == "" || checkpoint.SrcCollectFromBlockNums == 0 || checkpoint.Ver == 0 {
		return checkpoint, nil
	}
	if err := ckp.storage.Save(ctx, &checkpoint, data.Key(checkpoint.Channel)); err != nil {
		return model.Checkpoint{}, fmt.Errorf("save checkpoint : %w", err)
	}

	return checkpoint, nil
}

func (ckp *BlockCheckpoint) CheckpointLoad(ctx context.Context, id model.ID) (model.Checkpoint, error) {
	checkpoint := model.Checkpoint{}
	if err := ckp.storage.Load(ctx, &checkpoint, data.Key(id)); err != nil {
		return model.Checkpoint{}, fmt.Errorf("load checkpoint : %w", err)
	}

	return checkpoint, nil
}
