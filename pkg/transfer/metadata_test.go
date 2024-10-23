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

func TestMetadata(t *testing.T) {
	storage, err := redis.NewStorage(
		redis2.NewUniversalClient(&redis2.UniversalOptions{
			Addrs: []string{miniredis.RunT(t).Addr()},
		}),
		time.Hour,
		"test",
	)
	assert.NoError(t, err)

	metadata := NewMetadata(storage)

	mdlMetadata := model.Metadata{
		"traceId": "test-trace-id",
		"spanId":  "test-span-id",
	}

	t.Run("saving metadata", func(t *testing.T) {
		err = metadata.MetadataSave(context.TODO(), mdlMetadata, "01")
		assert.NoError(t, err)
	})

	t.Run("loading metadata", func(t *testing.T) {
		var loadedMetadata model.Metadata
		loadedMetadata, err = metadata.MetadataLoad(context.TODO(), "01")
		assert.NoError(t, err)
		assert.Equal(t, mdlMetadata, loadedMetadata)
	})
}
