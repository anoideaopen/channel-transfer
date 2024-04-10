package chproducer

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/anoideaopen/channel-transfer/pkg/metrics"
	"github.com/anoideaopen/channel-transfer/pkg/model"
	"github.com/anoideaopen/foundation/core/cctransfer"
	fpb "github.com/anoideaopen/foundation/proto"
	"github.com/hyperledger/fabric-sdk-go/pkg/client/channel"
	"github.com/pkg/errors"
)

const paginationPageSize = "10"

var errTransferNotFound = ": " + cctransfer.ErrNotFound.Error()

func (h *Handler) queryChannelTransfers(ctx context.Context) ([]*fpb.CCTransfer, error) {
	executor, err := h.poolController.Executor(h.channel)
	if err != nil {
		return nil, errors.Wrap(err, "executor")
	}

	transfers := make([]*fpb.CCTransfer, 0)
	bookmark := ""

	for {
		resp, err := executor.Query(
			ctx,
			channel.Request{
				ChaincodeID: h.chaincodeID,
				Fcn:         model.QueryChannelTransfersFrom.String(),
				Args:        [][]byte{[]byte(paginationPageSize), []byte(bookmark)},
			},
			[]channel.RequestOption{},
		)
		if err != nil {
			return nil, errors.Wrap(err, "query transfers")
		}

		data := &fpb.CCTransfers{}
		err = json.Unmarshal(resp.Payload, &data)
		if err != nil {
			return nil, err
		}

		for _, transfer := range data.Ccts {
			if strings.ToLower(transfer.From) == h.channel {
				transfers = append(transfers, transfer)
			}
		}

		if data.Bookmark == "" {
			break
		}
		bookmark = data.Bookmark
	}

	return transfers, nil
}

func (h *Handler) createTransferFrom(ctx context.Context, request model.TransferRequest) (model.StatusKind, error) {
	startTime := time.Now()

	doer, err := h.poolController.Executor(h.channel)
	if err != nil {
		return model.InternalErrorTransferStatus, errors.Wrap(err, "executor")
	}

	args := [][]byte{
		[]byte(request.Request),
		[]byte(h.channel),
		[]byte(h.chaincodeID),
		[]byte(request.Transfer),
		[]byte(request.To),
		[]byte(request.Token),
		[]byte(request.Amount),
		[]byte(request.Nonce),
		[]byte(request.PublicKey),
		[]byte(request.Sign),
	}

	if request.Method == model.TxChannelTransferByAdmin.String() {
		args = append(args[:6], args[5:]...)
		args[5] = []byte(request.User)
	}

	tArgs := make([]string, 0, len(args))
	for _, arg := range args {
		tArgs = append(tArgs, string(arg))
	}
	h.log.Debugf("transfer request arguments: %+v", tArgs)

	_, err = doer.Invoke(
		ctx,
		channel.Request{
			ChaincodeID: h.chaincodeID,
			Fcn:         request.Method,
			Args:        args,
		},
		[]channel.RequestOption{},
	)
	if err != nil {
		return model.ErrorTransferFrom, errors.Wrap(err, request.Method)
	}

	h.m.TotalTransferCreated().Inc(metrics.Labels().Channel.Create(h.channel))
	h.m.TransferStageExecutionTimeDuration().Observe(
		time.Since(startTime).Seconds(),
		metrics.Labels().Channel.Create(h.channel),
		metrics.Labels().TransferStatus.Create(model.InProgressTransferFrom.String()),
	)

	return model.InProgressTransferFrom, nil
}

func (h *Handler) createTransferTo(ctx context.Context, transfer *fpb.CCTransfer) (model.StatusKind, error) {
	channelName := strings.ToLower(transfer.To)
	if status, err := h.expandTO(ctx, channelName); err != nil {
		return status, err
	}

	doer, err := h.poolController.Executor(channelName)
	if err != nil {
		return model.InternalErrorTransferStatus, errors.Wrap(err, "executor")
	}

	body, err := json.Marshal(transfer)
	if err != nil {
		return model.InternalErrorTransferStatus, errors.Wrap(err, "json marshal transfer")
	}

	_, err = doer.Invoke(
		ctx,
		channel.Request{
			ChaincodeID: channelName,
			Fcn:         model.TxCreateCCTransferTo.String(),
			Args:        [][]byte{body},
		},
		[]channel.RequestOption{},
	)
	if err != nil {
		return model.ErrorTransferTo, errors.Wrap(err, model.TxCreateCCTransferTo.String())
	}

	return model.InProgressTransferTo, nil
}

func (h *Handler) cancelTransferFrom(ctx context.Context, transferID string) (model.StatusKind, error) {
	if err := h.invoke(ctx, h.channel, h.chaincodeID, model.TxCancelCCTransferFrom, transferID); err != nil {
		if strings.Contains(err.Error(), errTransferNotFound) {
			h.log.Error(errors.Wrap(err, "cancel transfer"))
			return model.Canceled, nil
		}

		return model.InternalErrorTransferStatus, err
	}

	return model.Canceled, nil
}

func (h *Handler) commitTransferFrom(ctx context.Context, transferID string) (model.StatusKind, error) {
	if err := h.invoke(ctx, h.channel, h.chaincodeID, model.NbTxCommitCCTransferFrom, transferID); err != nil {
		if strings.Contains(err.Error(), errTransferNotFound) {
			h.log.Error(errors.Wrap(err, "commit transfer"))
			return model.Canceled, nil
		}
		return model.InternalErrorTransferStatus, err
	}

	return model.CommitTransferFrom, nil
}

func (h *Handler) deleteTransferFrom(ctx context.Context, transferID string) (model.StatusKind, error) {
	if err := h.invoke(ctx, h.channel, h.chaincodeID, model.NbTxDeleteCCTransferFrom, transferID); err != nil {
		if strings.Contains(err.Error(), errTransferNotFound) {
			h.log.Error(errors.Wrap(err, "delete transfer"))
			return model.CompletedTransferFromDelete, nil
		}
		return model.InternalErrorTransferStatus, err
	}

	return model.CompletedTransferFromDelete, nil
}

func (h *Handler) deleteTransferTo(ctx context.Context, channelName string, transferID string) (model.StatusKind, error) {
	if err := h.invoke(ctx, channelName, channelName, model.NbTxDeleteCCTransferTo, transferID); err != nil {
		if strings.Contains(err.Error(), errTransferNotFound) {
			h.log.Error(errors.Wrap(err, "delete transfer"))
			return model.CompletedTransferToDelete, nil
		}
		return model.InternalErrorTransferStatus, err
	}

	return model.CompletedTransferToDelete, nil
}

func (h *Handler) invoke(ctx context.Context, channelName string, chaincodeID string, method model.TransactionKind, transferID string) error {
	doer, err := h.poolController.Executor(channelName)
	if err != nil {
		return errors.Wrap(err, "executor")
	}

	_, err = doer.Invoke(
		ctx,
		channel.Request{
			ChaincodeID: chaincodeID,
			Fcn:         method.String(),
			Args:        [][]byte{[]byte(transferID)},
		},
		[]channel.RequestOption{},
	)
	if err != nil {
		return errors.Wrap(err, method.String())
	}

	return nil
}

func (h *Handler) queryChannelTransferTo(ctx context.Context, channelName string, transferID string) (bool, error) {
	executor, err := h.poolController.Executor(channelName)
	if err != nil {
		return false, errors.Wrap(err, "expand")
	}

	_, err = executor.Query(
		ctx,
		channel.Request{
			ChaincodeID: channelName,
			Fcn:         model.QueryChannelTransferTo.String(),
			Args:        [][]byte{[]byte(transferID)},
		},
		[]channel.RequestOption{},
	)
	if err != nil && strings.Contains(err.Error(), errTransferNotFound) {
		return false, nil
	}

	if err != nil {
		return false, err
	}

	return true, nil
}

func (h *Handler) queryChannelTransferFrom(ctx context.Context, channelName string, transferID string) (bool, error) {
	executor, err := h.poolController.Executor(channelName)
	if err != nil {
		return false, errors.Wrap(err, "expand")
	}

	_, err = executor.Query(
		ctx,
		channel.Request{
			ChaincodeID: channelName,
			Fcn:         model.QueryChannelTransferFrom.String(),
			Args:        [][]byte{[]byte(transferID)},
		},
		[]channel.RequestOption{},
	)
	if err != nil && strings.Contains(err.Error(), errTransferNotFound) {
		return false, nil
	}

	if err != nil {
		return false, err
	}

	return true, nil
}
