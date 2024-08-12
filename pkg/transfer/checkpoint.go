package transfer

import (
	"context"
	"errors"
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
	existsCheckPoint := model.Checkpoint{}
	if err := ckp.storage.Load(ctx, &existsCheckPoint, data.Key(checkpoint.Channel)); err != nil {
		if !errors.Is(err, data.ErrObjectNotFound) {
			return model.Checkpoint{}, fmt.Errorf("save checkpoint : %w", err)
		}
	} else {
		if checkpoint.Ver > existsCheckPoint.Ver {
			return model.Checkpoint{}, data.ErrVersionMismatch
		}
		if checkpoint.Ver < existsCheckPoint.Ver {
			checkpoint.Ver = existsCheckPoint.Ver
		}
		checkpoint.Ver++
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
