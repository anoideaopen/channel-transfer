package transfer

import (
	"context"
	"errors"
	"fmt"

	"github.com/anoideaopen/channel-transfer/pkg/data"
	"github.com/anoideaopen/channel-transfer/pkg/data/redis"
	"github.com/anoideaopen/channel-transfer/pkg/model"
	"github.com/anoideaopen/glog"
)

type BlockCheckpoint struct {
	storage *redis.Storage
}

func NewBlockCheckpoint(storage *redis.Storage) *BlockCheckpoint {
	return &BlockCheckpoint{
		storage: storage,
	}
}

func (ckp *BlockCheckpoint) CheckpointSave(ctx context.Context, checkpoint model.Checkpoint, log glog.Logger) (model.Checkpoint, error) {
	existsCheckPoint := model.Checkpoint{}
	if err := ckp.storage.Load(ctx, &existsCheckPoint, data.Key(checkpoint.Channel)); err != nil { //nolint:nestif
		if !errors.Is(err, data.ErrObjectNotFound) {
			return model.Checkpoint{}, fmt.Errorf("save checkpoint : %w", err)
		}
		if log != nil {
			log.Debug("PFI8 ", err, " ", data.Key(checkpoint.Channel))
		}
	} else {
		if log != nil {
			log.Debug("PFI5 ", checkpoint.Ver, " ", existsCheckPoint.Ver, " ", data.Key(checkpoint.Channel))
		}
		if existsCheckPoint.Ver != checkpoint.Ver {
			return model.Checkpoint{}, data.ErrVersionMismatch
		}
		checkpoint.Ver++
		if log != nil {
			log.Debug("PFI6 ", checkpoint.Ver, " ", data.Key(checkpoint.Channel))
		}
	}

	if log != nil {
		log.Debug("PFI7 ", checkpoint.Ver, " ", data.Key(checkpoint.Channel))
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
