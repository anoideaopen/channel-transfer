package model

import (
	"context"
	"testing"

	"github.com/anoideaopen/channel-transfer/pkg/data"
	"github.com/anoideaopen/channel-transfer/pkg/data/inmem"
	"github.com/stretchr/testify/require"
)

func TestTransferBlock(t *testing.T) {
	var (
		tb = &TransferBlock{
			Channel:  ID("CH1"),
			Transfer: ID("ID1"),
			Transactions: []Transaction{
				{
					BlockNum:       34,
					TxID:           "63c54afa562f7e8c6c0f352eade8f8d77ae26017a09e4e30b450d79ba50bd403",
					FuncName:       "deleteCCTransferTo",
					Args:           [][]byte{[]byte("deleteCCTransferTo"), []byte("57214401-4713-4b24-ad44-e2f5935e408d")},
					TimeNs:         1660058499519,
					ValidationCode: 0,
					BatchResponse:  nil,
					Response:       nil,
				},
			},
		}

		db  = inmem.NewStorage()
		err error
	)

	err = db.Save(context.TODO(), tb, "01")
	require.NoError(t, err)

	err = db.Save(context.TODO(), tb, "02")
	require.NoError(t, err)

	collection, err := data.ToSlice[TransferBlock](
		db.Search(context.TODO(), &TransferBlock{}, "0"),
	)
	require.NoError(t, err)
	require.Len(t, collection, 2)

	var obj TransferBlock
	err = db.Load(context.TODO(), &obj, "02")
	require.NoError(t, err)
	require.Equal(t, obj, *tb)
}
