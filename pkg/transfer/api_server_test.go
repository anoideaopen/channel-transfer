package transfer

import (
	"context"
	"testing"

	"github.com/anoideaopen/channel-transfer/pkg/data"
	"github.com/anoideaopen/channel-transfer/pkg/model"
	"github.com/anoideaopen/channel-transfer/pkg/transfer/mock"
	dto "github.com/anoideaopen/channel-transfer/proto"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func TestAPIServerTransfer(t *testing.T) {
	requests := make(chan model.TransferRequest, 50)
	defer close(requests)

	var (
		channels = []string{"ch1", "ch2"}
		ctx      = context.Background()
		ctrl     = gomock.NewController(t)
		mc       = mock.NewMockRequestController(ctrl)
		srv      = NewAPIServer(ctx, requests, mc, channels)
	)

	var (
		gpCustomer = &dto.GeneralParams{
			RequestId:  "ID1",
			MethodName: model.TxChannelTransferByCustomer.String(),
			Channel:    channels[0],
			Chaincode:  "chaincode",
			Sign:       "sign",
			Nonce:      "1234567890",
			PublicKey:  "a1",
		}

		inCustomer = &dto.TransferBeginCustomerRequest{
			Generals:   gpCustomer,
			IdTransfer: "T1",
			ChannelTo:  "ch2",
			Token:      "ch1",
			Amount:     "1",
		}

		mdlCustomer = model.TransferRequest{
			Channel:   inCustomer.Generals.Channel,
			Request:   model.ID(inCustomer.Generals.RequestId),
			Transfer:  model.ID(inCustomer.IdTransfer),
			Method:    inCustomer.Generals.MethodName,
			Nonce:     inCustomer.Generals.Nonce,
			Sign:      inCustomer.Generals.Sign,
			Chaincode: inCustomer.Generals.Chaincode,
			PublicKey: inCustomer.Generals.PublicKey,
			To:        "ch2",
			Token:     "ch1",
			Amount:    "1",
			TransferResult: model.TransferResult{
				Status: "STATUS_IN_PROCESS",
			},
		}

		outCustomer = &dto.TransferStatusResponse{
			IdTransfer: inCustomer.IdTransfer,
			Status:     dto.TransferStatusResponse_STATUS_IN_PROCESS,
		}

		gpAdmin = &dto.GeneralParams{
			RequestId:  "ID2",
			MethodName: model.TxChannelTransferByAdmin.String(),
			Channel:    channels[0],
			Chaincode:  "chaincode",
			Sign:       "sign",
			Nonce:      "1234567890",
			PublicKey:  "a1",
		}
		inAdmin = &dto.TransferBeginAdminRequest{
			Generals:   gpAdmin,
			IdTransfer: "T2",
			ChannelTo:  "ch2",
			Address:    "UserID",
			Token:      "ch2",
			Amount:     "1",
		}

		mdlAdmin = model.TransferRequest{
			Channel:   inAdmin.Generals.Channel,
			Request:   model.ID(inAdmin.Generals.RequestId),
			Transfer:  model.ID(inAdmin.IdTransfer),
			User:      model.ID(inAdmin.Address),
			Method:    inAdmin.Generals.MethodName,
			Nonce:     inCustomer.Generals.Nonce,
			Sign:      inCustomer.Generals.Sign,
			Chaincode: inCustomer.Generals.Chaincode,
			PublicKey: inCustomer.Generals.PublicKey,
			To:        "ch2",
			Token:     "ch2",
			Amount:    "1",
			TransferResult: model.TransferResult{
				Status: "STATUS_IN_PROCESS",
			},
		}

		outAdmin = &dto.TransferStatusResponse{
			IdTransfer: inAdmin.IdTransfer,
			Status:     dto.TransferStatusResponse_STATUS_IN_PROCESS,
		}
	)

	gomock.InOrder(
		mc.EXPECT().TransferKeep(ctx, mdlCustomer).Return(nil),
		mc.EXPECT().TransferKeep(ctx, mdlAdmin).Return(nil),
	)

	resp, err := srv.TransferByCustomer(ctx, inCustomer)
	assert.NoError(t, err)
	assert.Equal(t, resp, outCustomer)

	resp, err = srv.TransferByAdmin(ctx, inAdmin)
	assert.NoError(t, err)
	assert.Equal(t, resp, outAdmin)

	inCustomer.Generals.Channel = "ch3"
	resp, err = srv.TransferByCustomer(ctx, inCustomer)
	assert.Error(t, err)
	assert.Equal(t, resp, &dto.TransferStatusResponse{
		IdTransfer: inCustomer.IdTransfer,
		Status:     dto.TransferStatusResponse_STATUS_ERROR,
		Message:    "parse transfer request: " + ErrBadChannel.Error(),
	})
}

func TestAPIServerTransferStatus(t *testing.T) {
	requests := make(chan model.TransferRequest, 50)
	defer close(requests)

	var (
		ctx  = context.Background()
		ctrl = gomock.NewController(t)
		mc   = mock.NewMockRequestController(ctrl)
		srv  = NewAPIServer(ctx, requests, mc, nil)
	)

	var (
		in = &dto.TransferStatusRequest{
			IdTransfer: "ID3",
		}

		out = &dto.TransferStatusResponse{
			IdTransfer: in.IdTransfer,
			Status:     dto.TransferStatusResponse_STATUS_IN_PROCESS,
		}

		mdl = model.TransferRequest{
			Request: model.ID(in.IdTransfer),
			Method:  "test3",
			TransferResult: model.TransferResult{
				Status: "STATUS_IN_PROCESS",
			},
		}
	)

	gomock.InOrder(
		mc.EXPECT().TransferFetch(ctx, mdl.Request).Return(mdl, nil),
		mc.EXPECT().TransferFetch(ctx, mdl.Request).Return(model.TransferRequest{}, data.ErrObjectNotFound),
	)

	resp, err := srv.TransferStatus(ctx, in)
	assert.NoError(t, err)
	assert.Equal(t, resp, out)

	// --------

	_, err = srv.TransferStatus(ctx, in)
	assert.ErrorContains(t, err, data.ErrObjectNotFound.Error())
}

func TestAPIServerMultiTransfer(t *testing.T) {
	requests := make(chan model.TransferRequest, 50)
	defer close(requests)

	var (
		channels = []string{"ch1", "ch2"}
		ctx      = context.Background()
		ctrl     = gomock.NewController(t)
		mc       = mock.NewMockRequestController(ctrl)
		srv      = NewAPIServer(ctx, requests, mc, channels)
	)

	var (
		gpCustomer = &dto.GeneralParams{
			RequestId:  "ID1",
			MethodName: model.TxChannelMultiTransferByCustomer.String(),
			Channel:    channels[0],
			Chaincode:  "chaincode",
			Sign:       "sign",
			Nonce:      "1234567890",
			PublicKey:  "a1",
		}

		inCustomer = &dto.MultiTransferBeginCustomerRequest{
			Generals:   gpCustomer,
			IdTransfer: "T1",
			ChannelTo:  "ch2",
			Items: []*dto.TransferItem{
				{
					Token:  "ch1_1",
					Amount: "1",
				},
				{
					Token:  "ch1_2",
					Amount: "1",
				},
			},
		}

		mdlCustomer = model.TransferRequest{
			Channel:   inCustomer.Generals.Channel,
			Request:   model.ID(inCustomer.Generals.RequestId),
			Transfer:  model.ID(inCustomer.IdTransfer),
			Method:    inCustomer.Generals.MethodName,
			Nonce:     inCustomer.Generals.Nonce,
			Sign:      inCustomer.Generals.Sign,
			Chaincode: inCustomer.Generals.Chaincode,
			PublicKey: inCustomer.Generals.PublicKey,
			To:        "ch2",
			Items: []model.TransferItem{
				{
					Token:  "ch1_1",
					Amount: "1",
				},
				{
					Token:  "ch1_2",
					Amount: "1",
				},
			},
			TransferResult: model.TransferResult{
				Status: "STATUS_IN_PROCESS",
			},
		}

		outCustomer = &dto.TransferStatusResponse{
			IdTransfer: inCustomer.IdTransfer,
			Status:     dto.TransferStatusResponse_STATUS_IN_PROCESS,
		}

		gpAdmin = &dto.GeneralParams{
			RequestId:  "ID2",
			MethodName: model.TxChannelMultiTransferByAdmin.String(),
			Channel:    channels[0],
			Chaincode:  "chaincode",
			Sign:       "sign",
			Nonce:      "1234567890",
			PublicKey:  "a1",
		}
		inAdmin = &dto.MultiTransferBeginAdminRequest{
			Generals:   gpAdmin,
			IdTransfer: "T2",
			ChannelTo:  "ch2",
			Address:    "UserID",
			Items: []*dto.TransferItem{
				{
					Token:  "ch2_1",
					Amount: "1",
				},
				{
					Token:  "ch2_2",
					Amount: "1",
				},
			},
		}

		mdlAdmin = model.TransferRequest{
			Channel:   inAdmin.Generals.Channel,
			Request:   model.ID(inAdmin.Generals.RequestId),
			Transfer:  model.ID(inAdmin.IdTransfer),
			User:      model.ID(inAdmin.Address),
			Method:    inAdmin.Generals.MethodName,
			Nonce:     inCustomer.Generals.Nonce,
			Sign:      inCustomer.Generals.Sign,
			Chaincode: inCustomer.Generals.Chaincode,
			PublicKey: inCustomer.Generals.PublicKey,
			To:        "ch2",
			Items: []model.TransferItem{
				{
					Token:  "ch2_1",
					Amount: "1",
				},
				{
					Token:  "ch2_2",
					Amount: "1",
				},
			},
			TransferResult: model.TransferResult{
				Status: "STATUS_IN_PROCESS",
			},
		}

		outAdmin = &dto.TransferStatusResponse{
			IdTransfer: inAdmin.IdTransfer,
			Status:     dto.TransferStatusResponse_STATUS_IN_PROCESS,
		}
	)

	gomock.InOrder(
		mc.EXPECT().TransferKeep(ctx, mdlCustomer).Return(nil),
		mc.EXPECT().TransferKeep(ctx, mdlAdmin).Return(nil),
	)

	resp, err := srv.MultiTransferByCustomer(ctx, inCustomer)
	assert.NoError(t, err)
	assert.Equal(t, resp, outCustomer)

	resp, err = srv.MultiTransferByAdmin(ctx, inAdmin)
	assert.NoError(t, err)
	assert.Equal(t, resp, outAdmin)

	inCustomer.Generals.Channel = "ch3"
	resp, err = srv.MultiTransferByCustomer(ctx, inCustomer)
	assert.Error(t, err)
	assert.Equal(t, resp, &dto.TransferStatusResponse{
		IdTransfer: inCustomer.IdTransfer,
		Status:     dto.TransferStatusResponse_STATUS_ERROR,
		Message:    "parse transfer request: " + ErrBadChannel.Error(),
	})
}
