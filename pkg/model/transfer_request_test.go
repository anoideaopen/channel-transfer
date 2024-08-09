package model

import (
	"context"
	"testing"

	"github.com/anoideaopen/channel-transfer/pkg/data"
	"github.com/anoideaopen/channel-transfer/pkg/data/inmem"
	"github.com/stretchr/testify/assert"
)

func TestTransferRequest(t *testing.T) {
	var (
		tr = &TransferRequest{
			Request:   ID("ID1"),
			Transfer:  ID("ID2"),
			User:      ID("ID3"),
			Method:    "string",
			Chaincode: "string",
			Channel:   "string",
			Nonce:     "string",
			PublicKey: "string",
			Sign:      "string",
			To:        "string",
			Token:     "string",
			Amount:    "string",
			Items: []TransferItem{
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

		db  = inmem.NewStorage()
		err error
	)

	err = db.Save(context.TODO(), tr, "01")
	assert.NoError(t, err)

	err = db.Save(context.TODO(), tr, "02")
	assert.NoError(t, err)

	err = db.Save(context.TODO(), tr, "33")
	assert.NoError(t, err)

	collection, err := data.ToSlice[TransferRequest](
		db.Search(context.TODO(), &TransferRequest{}, "0"),
	)
	assert.NoError(t, err)
	assert.Len(t, collection, 2)

	var obj TransferRequest
	err = db.Load(context.TODO(), &obj, "33")
	assert.NoError(t, err)
	assert.Equal(t, obj, *tr)
}
