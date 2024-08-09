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

func TestRequest(t *testing.T) {
	storage, err := redis.NewStorage(
		redis2.NewUniversalClient(&redis2.UniversalOptions{
			Addrs: []string{miniredis.RunT(t).Addr()},
		}),
		time.Hour,
		"test",
	)
	assert.NoError(t, err)

	user := model.ID("ID3")

	transferRequest := model.TransferRequest{
		Request:   model.ID("ID1"),
		Transfer:  model.ID("ID2"),
		User:      user,
		Method:    "string",
		Chaincode: "string",
		Channel:   "string",
		Nonce:     "string",
		PublicKey: "string",
		Sign:      "string",
		To:        "string",
		Token:     "string",
		Amount:    "string",
		Items: []model.TransferItem{
			{
				Token:  "string",
				Amount: "string",
			},
			{
				Token:  "string",
				Amount: "string",
			},
		},
	}

	request := NewRequest(storage)

	err = request.TransferKeep(context.TODO(), transferRequest)
	assert.NoError(t, err)

	got, err := request.TransferFetch(context.TODO(), transferRequest.Transfer)
	assert.NoError(t, err)
	assert.Equal(t, user, got.User)
}
