package model

import (
	"context"
	"testing"

	"github.com/anoideaopen/channel-transfer/pkg/data"
	"github.com/anoideaopen/channel-transfer/pkg/data/inmem"
	"github.com/stretchr/testify/assert"
)

func TestMetadata(t *testing.T) {
	var (
		m = &Metadata{
			"traceId": "0123456789",
			"spanId":  "9876543210",
		}

		db  = inmem.NewStorage()
		err error
	)

	err = db.Save(context.TODO(), m, "01")
	assert.NoError(t, err)

	err = db.Save(context.TODO(), m, "02")
	assert.NoError(t, err)

	collection, err := data.ToSlice[Metadata](
		db.Search(context.TODO(), &Metadata{}, "0"),
	)
	assert.NoError(t, err)
	assert.Len(t, collection, 2)

	var obj Metadata
	err = db.Load(context.TODO(), &obj, "02")
	assert.NoError(t, err)
	assert.Equal(t, obj, *m)
}
