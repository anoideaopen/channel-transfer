package transfer

import (
	"context"
	"fmt"

	"github.com/anoideaopen/channel-transfer/pkg/data"
	"github.com/anoideaopen/channel-transfer/pkg/data/redis"
	"github.com/anoideaopen/channel-transfer/pkg/model"
)

type Metadata struct {
	storage *redis.Storage
}

func NewMetadata(storage *redis.Storage) *Metadata {
	return &Metadata{
		storage: storage,
	}
}

func (md *Metadata) MetadataSave(ctx context.Context, metadataModel model.Metadata, transferID model.ID) error {
	if err := md.storage.Save(ctx, &metadataModel, data.Key(transferID)); err != nil {
		return fmt.Errorf("save metadata: %w", err)
	}

	return nil
}

func (md *Metadata) MetadataLoad(ctx context.Context, transferID model.ID) (model.Metadata, error) {
	metadataModel := model.Metadata{}
	if err := md.storage.Load(ctx, &metadataModel, data.Key(transferID)); err != nil {
		return metadataModel, fmt.Errorf("load metadata: %w", err)
	}

	return metadataModel, nil
}
