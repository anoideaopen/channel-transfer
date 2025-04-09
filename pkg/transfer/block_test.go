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

func TestBlock(t *testing.T) {
	storage, err := redis.NewStorage(
		context.Background(),
		redis2.NewUniversalClient(&redis2.UniversalOptions{
			Addrs: []string{miniredis.RunT(t).Addr()},
		}),
		time.Hour,
		"test",
	)
	assert.NoError(t, err)

	transactions := []model.Transaction{
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
	}

	transferBlock := model.TransferBlock{
		Channel:      model.ID("ID1"),
		Transfer:     model.ID("ID2"),
		Transactions: transactions,
	}

	ledgerBlock := NewLedgerBlock(storage)

	err = ledgerBlock.BlockSave(context.TODO(), transferBlock, redis.TTLNotTakenInto)
	assert.NoError(t, err)

	got, err := ledgerBlock.BlockLoad(context.TODO(), ledgerBlock.Key(transferBlock.Channel, transferBlock.Transfer))
	assert.NoError(t, err)
	assert.Equal(t, transactions, got.Transactions)
}
