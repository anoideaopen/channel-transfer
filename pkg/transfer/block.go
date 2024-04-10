package transfer

import (
	"context"
	"fmt"
	"path"
	"time"

	"github.com/anoideaopen/channel-transfer/pkg/data"
	"github.com/anoideaopen/channel-transfer/pkg/data/redis"
	"github.com/anoideaopen/channel-transfer/pkg/model"
)

type LedgerBlock struct {
	storage *redis.Storage
}

func NewLedgerBlock(storage *redis.Storage) *LedgerBlock {
	return &LedgerBlock{
		storage: storage,
	}
}

func (lb *LedgerBlock) BlockSave(ctx context.Context, transferBlock model.TransferBlock, ttl time.Duration) error {
	if err := lb.storage.SaveConsideringTTL(ctx, &transferBlock, data.Key(lb.Key(transferBlock.Channel, transferBlock.Transfer)), ttl); err != nil {
		return fmt.Errorf("save block : %w", err)
	}
	return nil
}

func (lb *LedgerBlock) BlockLoad(ctx context.Context, id model.ID) (model.TransferBlock, error) {
	transferBlock := model.TransferBlock{}
	if err := lb.storage.Load(ctx, &transferBlock, data.Key(id)); err != nil {
		return transferBlock, fmt.Errorf("load request : %w", err)
	}
	return transferBlock, nil
}

func (lb *LedgerBlock) Key(channel model.ID, transfer model.ID) model.ID {
	return model.ID(path.Join(string(channel), string(transfer)))
}
