package transfer

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/anoideaopen/channel-transfer/pkg/data/redis"
	"github.com/anoideaopen/channel-transfer/pkg/model"
	redis2 "github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
)

func TestBlockCheckpoint(t *testing.T) {
	storage, err := redis.NewStorage(
		redis2.NewUniversalClient(&redis2.UniversalOptions{
			Addrs: []string{miniredis.RunT(t).Addr()},
		}),
		time.Hour,
		"test",
	)
	assert.NoError(t, err)

	checkpoint := model.Checkpoint{
		Channel:                 "cc",
		Ver:                     1,
		SrcCollectFromBlockNums: 1,
	}

	blockCheckpoint := NewBlockCheckpoint(storage)

	got, err := blockCheckpoint.CheckpointSave(context.TODO(), checkpoint)
	assert.NoError(t, err)

	got.SrcCollectFromBlockNums++
	checkpoint.SrcCollectFromBlockNums++
	checkpoint.Ver++

	got, err = blockCheckpoint.CheckpointSave(context.TODO(), got)
	assert.NoError(t, err)
	assert.Equal(t, checkpoint, got)

	got, err = blockCheckpoint.CheckpointLoad(context.TODO(), checkpoint.Channel)
	assert.NoError(t, err)
	assert.Equal(t, int64(2), got.Ver)
}
