package transfer

import (
	"context"
	"testing"

	"github.com/anoideaopen/channel-transfer/pkg/data"
	"github.com/anoideaopen/channel-transfer/pkg/model"
	"github.com/anoideaopen/channel-transfer/pkg/transfer/mock"
	dto "github.com/anoideaopen/channel-transfer/proto"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
)

func TestAPIServerTransfer(t *testing.T) {
	requests := make(chan model.TransferRequest, 50)
	defer close(requests)

	var (
		channels      = []string{"ch1", "ch2"}
		ctx           = context.Background()
		tracingCtx, _ = tracer.Start(ctx, "fake span for tracing context generation") //nolint:spancheck
		ctrl          = gomock.NewController(t)
		mc            = mock.NewMockRequestController(ctrl)
		srv           = NewAPIServer(ctx, requests, mc, channels)
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
			Channel:   inCustomer.GetGenerals().GetChannel(),
			Request:   model.ID(inCustomer.GetGenerals().GetRequestId()),
			Transfer:  model.ID(inCustomer.GetIdTransfer()),
			Method:    inCustomer.GetGenerals().GetMethodName(),
			Nonce:     inCustomer.GetGenerals().GetNonce(),
			Sign:      inCustomer.GetGenerals().GetSign(),
			Chaincode: inCustomer.GetGenerals().GetChaincode(),
			PublicKey: inCustomer.GetGenerals().GetPublicKey(),
			To:        "ch2",
			Token:     "ch1",
			Amount:    "1",
			Status:    "STATUS_IN_PROCESS",
			Metadata:  make(model.TransferMetadata),
		}

		outCustomer = &dto.TransferStatusResponse{
			IdTransfer: inCustomer.GetIdTransfer(),
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
			Channel:   inAdmin.GetGenerals().GetChannel(),
			Request:   model.ID(inAdmin.GetGenerals().GetRequestId()),
			Transfer:  model.ID(inAdmin.GetIdTransfer()),
			User:      model.ID(inAdmin.GetAddress()),
			Method:    inAdmin.GetGenerals().GetMethodName(),
			Nonce:     inCustomer.GetGenerals().GetNonce(),
			Sign:      inCustomer.GetGenerals().GetSign(),
			Chaincode: inCustomer.GetGenerals().GetChaincode(),
			PublicKey: inCustomer.GetGenerals().GetPublicKey(),
			To:        "ch2",
			Token:     "ch2",
			Amount:    "1",
			Status:    "STATUS_IN_PROCESS",
			Metadata:  make(model.TransferMetadata),
		}

		outAdmin = &dto.TransferStatusResponse{
			IdTransfer: inAdmin.GetIdTransfer(),
			Status:     dto.TransferStatusResponse_STATUS_IN_PROCESS,
		}
	)

	gomock.InOrder(
		mc.EXPECT().TransferKeep(tracingCtx, mdlCustomer).Return(nil),
		mc.EXPECT().TransferKeep(tracingCtx, mdlAdmin).Return(nil),
	)

	resp, err := srv.TransferByCustomer(ctx, inCustomer)
	require.NoError(t, err)
	require.Equal(t, resp, outCustomer)

	resp, err = srv.TransferByAdmin(ctx, inAdmin)
	require.NoError(t, err)
	require.Equal(t, resp, outAdmin)

	inCustomer.Generals.Channel = "ch3"
	resp, err = srv.TransferByCustomer(ctx, inCustomer)
	require.Error(t, err)
	require.Equal(t, &dto.TransferStatusResponse{
		IdTransfer: inCustomer.GetIdTransfer(),
		Status:     dto.TransferStatusResponse_STATUS_ERROR,
		Message:    "parse transfer request: " + ErrBadChannel.Error(),
	}, resp)
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
			IdTransfer: in.GetIdTransfer(),
			Status:     dto.TransferStatusResponse_STATUS_IN_PROCESS,
		}

		mdl = model.TransferRequest{
			Request: model.ID(in.GetIdTransfer()),
			Method:  "test3",
			Status:  "STATUS_IN_PROCESS",
		}
	)

	gomock.InOrder(
		mc.EXPECT().TransferFetch(ctx, mdl.Request).Return(mdl, nil),
		mc.EXPECT().TransferFetch(ctx, mdl.Request).Return(model.TransferRequest{}, data.ErrObjectNotFound),
	)

	resp, err := srv.TransferStatus(ctx, in)
	require.NoError(t, err)
	require.Equal(t, resp, out)

	// --------

	_, err = srv.TransferStatus(ctx, in)
	require.ErrorContains(t, err, data.ErrObjectNotFound.Error())
}

func TestAPIServerMultiTransfer(t *testing.T) {
	requests := make(chan model.TransferRequest, 50)
	defer close(requests)

	var (
		channels      = []string{"ch1", "ch2"}
		ctx           = context.Background()
		tracingCtx, _ = tracer.Start(ctx, "fake span for tracing context generation") //nolint:spancheck
		ctrl          = gomock.NewController(t)
		mc            = mock.NewMockRequestController(ctrl)
		srv           = NewAPIServer(ctx, requests, mc, channels)
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
			Channel:   inCustomer.GetGenerals().GetChannel(),
			Request:   model.ID(inCustomer.GetGenerals().GetRequestId()),
			Transfer:  model.ID(inCustomer.GetIdTransfer()),
			Method:    inCustomer.GetGenerals().GetMethodName(),
			Nonce:     inCustomer.GetGenerals().GetNonce(),
			Sign:      inCustomer.GetGenerals().GetSign(),
			Chaincode: inCustomer.GetGenerals().GetChaincode(),
			PublicKey: inCustomer.GetGenerals().GetPublicKey(),
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
			Status:   "STATUS_IN_PROCESS",
			Metadata: make(model.TransferMetadata),
		}

		outCustomer = &dto.TransferStatusResponse{
			IdTransfer: inCustomer.GetIdTransfer(),
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
			Channel:   inAdmin.GetGenerals().GetChannel(),
			Request:   model.ID(inAdmin.GetGenerals().GetRequestId()),
			Transfer:  model.ID(inAdmin.GetIdTransfer()),
			User:      model.ID(inAdmin.GetAddress()),
			Method:    inAdmin.GetGenerals().GetMethodName(),
			Nonce:     inCustomer.GetGenerals().GetNonce(),
			Sign:      inCustomer.GetGenerals().GetSign(),
			Chaincode: inCustomer.GetGenerals().GetChaincode(),
			PublicKey: inCustomer.GetGenerals().GetPublicKey(),
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
			Status:   "STATUS_IN_PROCESS",
			Metadata: make(model.TransferMetadata),
		}

		outAdmin = &dto.TransferStatusResponse{
			IdTransfer: inAdmin.GetIdTransfer(),
			Status:     dto.TransferStatusResponse_STATUS_IN_PROCESS,
		}
	)

	gomock.InOrder(
		mc.EXPECT().TransferKeep(tracingCtx, mdlCustomer).Return(nil),
		mc.EXPECT().TransferKeep(tracingCtx, mdlAdmin).Return(nil),
	)

	resp, err := srv.MultiTransferByCustomer(ctx, inCustomer)
	require.NoError(t, err)
	require.Equal(t, resp, outCustomer)

	resp, err = srv.MultiTransferByAdmin(ctx, inAdmin)
	require.NoError(t, err)
	require.Equal(t, resp, outAdmin)

	inCustomer.Generals.Channel = "ch3"
	resp, err = srv.MultiTransferByCustomer(ctx, inCustomer)
	require.Error(t, err)
	require.Equal(t, &dto.TransferStatusResponse{
		IdTransfer: inCustomer.GetIdTransfer(),
		Status:     dto.TransferStatusResponse_STATUS_ERROR,
		Message:    "parse transfer request: " + ErrBadChannel.Error(),
	}, resp)
}
