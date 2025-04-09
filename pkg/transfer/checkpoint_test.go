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

var (
	channelName = model.ID("cc")
	version1    = int64(1)
	version2    = int64(2)
)

func TestBlockCheckpoint(t *testing.T) {
	storage, err := redis.NewStorage(
		context.Background(),
		redis2.NewUniversalClient(&redis2.UniversalOptions{
			Addrs: []string{miniredis.RunT(t).Addr()},
		}),
		time.Hour,
		"test",
	)
	assert.NoError(t, err)

	blockCheckpoint := NewBlockCheckpoint(storage)

	checkpoint := model.Checkpoint{
		Channel:                 channelName,
		Ver:                     version1,
		SrcCollectFromBlockNums: uint64(version1),
	}

	resultCheckPoint := model.Checkpoint{}
	t.Run("saving initial checkpoint", func(t *testing.T) {
		resultCheckPoint, err = blockCheckpoint.CheckpointSave(context.TODO(), checkpoint)
		assert.NoError(t, err)
	})

	resultCheckPoint.SrcCollectFromBlockNums++
	resultCheckPoint.Ver++
	checkpoint.SrcCollectFromBlockNums = uint64(version2)
	checkpoint.Ver = version2

	t.Run("saving new checkpoint", func(t *testing.T) {
		resultCheckPoint, err = blockCheckpoint.CheckpointSave(context.TODO(), checkpoint)
		assert.NoError(t, err)
		assert.Equal(t, checkpoint, resultCheckPoint)
	})

	t.Run("loading checkpoint", func(t *testing.T) {
		resultCheckPoint, err = blockCheckpoint.CheckpointLoad(context.TODO(), checkpoint.Channel)
		assert.NoError(t, err)
		assert.Equal(t, version2, resultCheckPoint.Ver)
		assert.Equal(t, uint64(version2), resultCheckPoint.SrcCollectFromBlockNums)
	})
}
