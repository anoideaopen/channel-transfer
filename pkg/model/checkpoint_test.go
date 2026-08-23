package model

import (
	"context"
	"testing"

	"github.com/anoideaopen/channel-transfer/pkg/data"
	"github.com/anoideaopen/channel-transfer/pkg/data/inmem"
	"github.com/stretchr/testify/require"
)

func TestCheckPoint(t *testing.T) {
	var (
		tb = &Checkpoint{
			Channel:                 "CH1",
			Ver:                     3,
			SrcCollectFromBlockNums: 50,
		}

		db  = inmem.NewStorage()
		err error
	)

	err = db.Save(context.TODO(), tb, "01")
	require.NoError(t, err)

	err = db.Save(context.TODO(), tb, "02")
	require.NoError(t, err)

	collection, err := data.ToSlice[Checkpoint](
		db.Search(context.TODO(), &Checkpoint{}, "0"),
	)
	require.NoError(t, err)
	require.Len(t, collection, 2)

	var obj Checkpoint
	err = db.Load(context.TODO(), &obj, "02")
	require.NoError(t, err)
	require.Equal(t, obj, *tb)
}
