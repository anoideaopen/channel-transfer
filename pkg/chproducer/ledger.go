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
	"github.com/go-errors/errors"
	"github.com/hyperledger/fabric-sdk-go/pkg/client/channel"
)

const paginationPageSize = "10"

var errTransferNotFound = ": " + cctransfer.ErrNotFound.Error()

func (h *Handler) queryChannelTransfers(ctx context.Context) ([]*fpb.CCTransfer, error) {
	executor, err := h.poolController.Executor(h.channel)
	if err != nil {
		return nil, errors.Errorf("executor: %w", err)
	}

	transfers := make([]*fpb.CCTransfer, 0)
	bookmark := ""
	h.log.Debugf("query channel transfers from, channel %s", h.chaincodeID)

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
			return nil, errors.Errorf("query transfers: %w", err)
		}

		data := &fpb.CCTransfers{}
		err = json.Unmarshal(resp.Payload, &data)
		if err != nil {
			return nil, err
		}

		for _, transfer := range data.GetCcts() {
			if strings.ToLower(transfer.GetFrom()) == h.channel {
				transfers = append(transfers, transfer)
			}
		}

		if data.GetBookmark() == "" {
			break
		}
		bookmark = data.GetBookmark()
	}

	return transfers, nil
}

func (h *Handler) createTransferFrom(ctx context.Context, request model.TransferRequest) (model.StatusKind, error) {
	if request.Token == "" {
		return model.InternalErrorTransferStatus, errors.New("executor: token is not specified")
	}
	if request.Amount == "" {
		return model.InternalErrorTransferStatus, errors.New("executor: amount is not specified")
	}

	startTime := time.Now()

	h.log.Debugf("create cc transfer from, channel %s, id %s, method %s", h.chaincodeID, request.Transfer, request.Method)
	doer, err := h.poolController.Executor(h.channel)
	if err != nil {
		return model.InternalErrorTransferStatus, errors.Errorf("executor: %w", err)
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
		return model.ErrorTransferFrom, errors.Errorf("%s: %w", request.Method, err)
	}

	h.m.TotalTransferCreated().Inc(metrics.Labels().Channel.Create(h.channel))
	h.m.TransferStageExecutionTimeDuration().Observe(
		time.Since(startTime).Seconds(),
		metrics.Labels().Channel.Create(h.channel),
		metrics.Labels().TransferStatus.Create(model.InProgressTransferFrom.String()),
	)

	return model.InProgressTransferFrom, nil
}

func (h *Handler) createMultiTransferFrom(ctx context.Context, request model.TransferRequest) (model.StatusKind, error) {
	if len(request.Items) == 0 {
		return model.InternalErrorTransferStatus, errors.New("executor: items is empty")
	}

	startTime := time.Now()

	h.log.Debugf("create cc transfer from, channel %s, id %s, method %s", h.chaincodeID, request.Transfer, request.Method)
	doer, err := h.poolController.Executor(h.channel)
	if err != nil {
		return model.InternalErrorTransferStatus, errors.Errorf("executor: %w", err)
	}

	items, err := json.Marshal(request.Items)
	if err != nil {
		return model.InternalErrorTransferStatus, errors.Errorf("executor: %w", err)
	}

	args := [][]byte{
		[]byte(request.Request),
		[]byte(h.channel),
		[]byte(h.chaincodeID),
		[]byte(request.Transfer),
		[]byte(request.To),
		items,
		[]byte(request.Nonce),
		[]byte(request.PublicKey),
		[]byte(request.Sign),
	}

	if request.Method == model.TxChannelMultiTransferByAdmin.String() {
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
		return model.ErrorTransferFrom, errors.Errorf("%s: %w", request.Method, err)
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
	channelName := strings.ToLower(transfer.GetTo())
	h.log.Debugf("create cc transfer to, channel %s, id %s", channelName, transfer.GetId())

	if status, err := h.expandTO(ctx, channelName); err != nil {
		return status, err
	}

	doer, err := h.poolController.Executor(channelName)
	if err != nil {
		return model.InternalErrorTransferStatus, errors.Errorf("executor: %w", err)
	}

	body, err := json.Marshal(transfer)
	if err != nil {
		return model.InternalErrorTransferStatus, errors.Errorf("json marshal transfer: %w", err)
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
		return model.InternalErrorTransferStatus, errors.Errorf("%s: %w", model.TxCreateCCTransferTo.String(), err)
	}

	return model.InProgressTransferTo, nil
}

func (h *Handler) cancelTransferFrom(ctx context.Context, transferID string) (model.StatusKind, error) {
	h.log.Debugf("cancel cc transfer from, channel %s, id %s", h.channel, transferID)

	if err := h.invoke(ctx, h.channel, h.chaincodeID, model.TxCancelCCTransferFrom, transferID); err != nil {
		if strings.Contains(err.Error(), errTransferNotFound) {
			h.log.Error(errors.Errorf("cancel transfer: %w", err))
			return model.Canceled, nil
		}

		return model.InternalErrorTransferStatus, err
	}

	return model.Canceled, nil
}

func (h *Handler) commitTransferFrom(ctx context.Context, transferID string) (model.StatusKind, error) {
	h.log.Debugf("commit cc transfer from, channel %s, id %s", h.channel, transferID)

	if err := h.invoke(ctx, h.channel, h.chaincodeID, model.NbTxCommitCCTransferFrom, transferID); err != nil {
		if strings.Contains(err.Error(), errTransferNotFound) {
			h.log.Error(errors.Errorf("commit transfer: %w", err))
			return model.Canceled, nil
		}
		return model.InternalErrorTransferStatus, err
	}

	return model.CommitTransferFrom, nil
}

func (h *Handler) deleteTransferFrom(ctx context.Context, transferID string) (model.StatusKind, error) {
	h.log.Debugf("delete cc transfer from, channel %s, id %s", h.channel, transferID)

	if err := h.invoke(ctx, h.channel, h.chaincodeID, model.NbTxDeleteCCTransferFrom, transferID); err != nil {
		if strings.Contains(err.Error(), errTransferNotFound) {
			h.log.Error(errors.Errorf("delete transfer: %w", err))
			return model.CompletedTransferFromDelete, nil
		}
		return model.InternalErrorTransferStatus, err
	}

	return model.CompletedTransferFromDelete, nil
}

func (h *Handler) deleteTransferTo(ctx context.Context, channelName string, transferID string) (model.StatusKind, error) {
	h.log.Debugf("delete cc transfer to, channel %s, id %s", channelName, transferID)

	if err := h.invoke(ctx, channelName, channelName, model.NbTxDeleteCCTransferTo, transferID); err != nil {
		if strings.Contains(err.Error(), errTransferNotFound) {
			h.log.Error(errors.Errorf("delete transfer: %w", err))
			return model.CompletedTransferToDelete, nil
		}
		return model.InternalErrorTransferStatus, err
	}

	return model.CompletedTransferToDelete, nil
}

func (h *Handler) invoke(ctx context.Context, channelName string, chaincodeID string, method model.TransactionKind, transferID string) error {
	doer, err := h.poolController.Executor(channelName)
	if err != nil {
		return errors.Errorf("executor: %w", err)
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
		return errors.Errorf("%s: %w", method.String(), err)
	}

	return nil
}

func (h *Handler) queryChannelTransferTo(ctx context.Context, channelName string, transferID string) (bool, error) {
	executor, err := h.poolController.Executor(channelName)
	if err != nil {
		return false, errors.Errorf("expand: %w", err)
	}

	h.log.Debugf("query channel transfer to, channel %s, id %s", channelName, transferID)
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
		return false, errors.Errorf("expand: %w", err)
	}

	h.log.Debugf("query channel transfer from, channel %s, id %s", channelName, transferID)
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
