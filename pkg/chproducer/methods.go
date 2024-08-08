package chproducer

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/anoideaopen/channel-transfer/pkg/data"
	"github.com/anoideaopen/channel-transfer/pkg/metrics"
	"github.com/anoideaopen/channel-transfer/pkg/model"
	"github.com/anoideaopen/channel-transfer/proto"
	fpb "github.com/anoideaopen/foundation/proto"
	"github.com/avast/retry-go/v4"
	"github.com/go-errors/errors"
)

const (
	repeatAttempt   = 5
	sleepAttempt    = 200 * time.Millisecond
	maxSleepAttempt = 500 * time.Millisecond
)

var (
	errBatchToNotFound   = errors.New("batch TO response not found")
	errBatchFromNotFound = errors.New("batch FROM response not found")
)

type failureTag string

const (
	expiredTransferTag failureTag = "expired transfer"
	transferFromError  failureTag = "transfer from"
	transferToError    failureTag = "transfer to"
)

var re = regexp.MustCompile(`no channel peers configured for channel \[`)

//nolint:funlen,gocognit,gocyclo
func (h *Handler) transferProcessing(ctx context.Context, initStatus model.StatusKind, transfer *fpb.CCTransfer, lastErr error) error {
	state := make(chan model.StatusKind, 1)
	defer close(state)

	state <- initStatus

	h.log.Infof("%s transfer processing started", transfer.GetId())

	var (
		failTag    failureTag
		lastStatus = initStatus
		startTime  = time.Now()
	)
	defer func() {
		if lastStatus == initStatus {
			return
		}
		h.m.TotalInWorkTransfer().Add(
			-1,
			metrics.Labels().Channel.Create(h.channel),
			metrics.Labels().TransferStatus.Create(lastStatus.String()),
		)
		if lastStatus != model.InternalErrorTransferStatus {
			h.m.TimeDurationCompleteTransferBeforeResponding().Observe(time.Since(startTime).Seconds(), metrics.Labels().Channel.Create(h.channel))
			h.m.TransferExecutionTimeDuration().Observe(time.Since(time.Unix(0, transfer.GetTimeAsNanos())).Seconds(), metrics.Labels().Channel.Create(h.channel))
		}
	}()

	for {
		select {
		case <-ctx.Done():
			h.log.Infof("%s transfer stopped", transfer.GetId())
			return ctx.Err()
		case status, ok := <-state:
			if !ok {
				break
			}

			if status != lastStatus {
				h.m.TotalInWorkTransfer().Add(
					-1,
					metrics.Labels().Channel.Create(h.channel),
					metrics.Labels().TransferStatus.Create(lastStatus.String()),
				)
			}
			h.m.TotalInWorkTransfer().Add(
				1,
				metrics.Labels().Channel.Create(h.channel),
				metrics.Labels().TransferStatus.Create(status.String()),
			)
			lastStatus = status

			status, ok, lastErr = h.hasBeenTransferExpired(ctx, status, transfer, lastErr)
			if ok {
				failTag = expiredTransferTag
			}

			h.log.Debugf("event status %s, id %s, channel from %s, channel to %s",
				status.String(), transfer.GetId(), transfer.GetFrom(), transfer.GetTo(),
			)

			switch status {
			case model.InProgressTransferFrom:
				if lastErr != nil {
					h.log.Infof("%s transfer processing stopped with error : %s", transfer.GetId(), lastErr.Error())
					return lastErr
				}
				status, lastErr = h.fromBatchResponse(ctx, transfer.GetId())
				if lastErr != nil {
					lastErr = errors.Errorf("'FROM' batch response: %w", lastErr)
				}
			case model.FromBatchNotFound:
				status = model.InternalErrorTransferStatus
			case model.CompletedTransferFrom:
				status, lastErr = h.createTransferTo(ctx, transfer)
				if lastErr != nil {
					lastErr = errors.Errorf("create transfer: %w", lastErr)
				}
			case model.InProgressTransferTo:
				status, lastErr = h.toBatchResponse(ctx, strings.ToLower(transfer.GetTo()), transfer.GetId())
				if lastErr != nil {
					lastErr = errors.Errorf("'TO' batch response: %w", lastErr)
				}
			case model.CompletedTransferTo:
				// the second transaction has passed, the tokens have been transferred, the client can dispose of them
				if err := h.requestStorage.TransferResultModify(
					ctx,
					model.ID(transfer.GetId()),
					model.TransferResult{
						Status: proto.TransferStatusResponse_STATUS_COMPLETED.String(),
					},
				); err != nil {
					h.log.Errorf("transfer response status not saved : %s : %s", transfer.GetId(), err.Error())
				}
				status, lastErr = h.commitTransferFrom(ctx, transfer.GetId())
				if lastErr != nil {
					lastErr = errors.Errorf("commit transfer: %w", lastErr)
				}
			case model.ErrorChannelToNotFound:
				if err := h.requestStorage.TransferResultModify(
					ctx,
					model.ID(transfer.GetId()),
					model.TransferResult{
						Status:  proto.TransferStatusResponse_STATUS_ERROR.String(),
						Message: message(lastErr),
					},
				); err != nil {
					h.log.Errorf("transfer response status not saved : %s : %s", transfer.GetId(), err.Error())
				}
				// cancel transfer
				status, lastErr = h.cancelTransfer(ctx, transfer.GetId(), lastErr)
				failTag = transferToError
			case model.ErrorTransferTo:
				failTag = transferToError
				status, lastErr = h.deleteTransferTo(ctx, strings.ToLower(transfer.GetTo()), transfer.GetId())
				if lastErr != nil {
					lastErr = errors.Errorf("delete transfer: %w", lastErr)
				}
			case model.CommitTransferFrom:
				// we continue technical operations with transfer
				status, lastErr = h.deleteTransferTo(ctx, strings.ToLower(transfer.GetTo()), transfer.GetId())
				if lastErr != nil {
					lastErr = errors.Errorf("delete transfer: %w", lastErr)
				}
			case model.ToBatchNotFound:
				status = model.InternalErrorTransferStatus
			case model.CompletedTransferToDelete:
				status, lastErr = h.deleteTransferFrom(ctx, transfer.GetId())
				if lastErr != nil {
					lastErr = errors.Errorf("delete transfer: %w", lastErr)
				}
			case model.CompletedTransferFromDelete:
				status = model.Completed
			case model.InternalErrorTransferStatus:
				return fmt.Errorf("%s transfer processing stopped with : %w", transfer.GetId(), lastErr)
			case model.ErrorTransferFrom:
				// fix error
				if err := h.requestStorage.TransferResultModify(
					ctx,
					model.ID(transfer.GetId()),
					model.TransferResult{
						Status:  proto.TransferStatusResponse_STATUS_ERROR.String(),
						Message: message(lastErr),
					},
				); err != nil {
					h.log.Errorf("transfer response status not saved : %s : %s", transfer.GetId(), err.Error())
				}
				// cancel transfer
				status, lastErr = h.cancelTransfer(ctx, transfer.GetId(), lastErr)
				failTag = transferFromError
			case model.Completed:
				h.m.TotalSuccessTransfer().Inc(metrics.Labels().Channel.Create(h.channel))
				h.log.Infof("%s transfer processing completed", transfer.GetId())
				return nil
			case model.Canceled:
				h.m.TotalFailureTransfer().Inc(metrics.Labels().Channel.Create(h.channel), metrics.Labels().FailTransferTag.Create(string(failTag)))
				if err := h.requestStorage.TransferResultModify(
					ctx,
					model.ID(transfer.GetId()),
					model.TransferResult{
						Status:  proto.TransferStatusResponse_STATUS_CANCELED.String(),
						Message: message(lastErr),
					},
				); err != nil {
					h.log.Errorf("transfer response status not saved : %s : %s", transfer.GetId(), err.Error())
				}
				message := transfer.GetId() + " transfer processing canceled"
				if lastErr != nil {
					message = transfer.GetId() + " transfer processing canceled with error : " + lastErr.Error()
				}
				h.log.Info(message)
				return nil
			}

			if lastErr != nil {
				h.log.Error(lastErr)
			}

			h.m.TransferStageExecutionTimeDuration().Observe(
				time.Since(startTime).Seconds(),
				metrics.Labels().Channel.Create(h.channel),
				metrics.Labels().TransferStatus.Create(status.String()),
			)

			state <- status
		}
	}
}

func (h *Handler) hasBeenTransferExpired(ctx context.Context, status model.StatusKind, transfer *fpb.CCTransfer, lastErr error) (model.StatusKind, bool, error) {
	if h.expiredTransfer(transfer) && isTheStatusExpired(status) {
		status, lastErr = h.cancelTransfer(ctx, transfer.GetId(), lastErr)
		return status, true, lastErr
	}
	return status, false, lastErr
}

func isTheStatusExpired(status model.StatusKind) bool {
	switch status {
	case model.InProgressTransferFrom,
		model.FromBatchNotFound,
		model.CompletedTransferFrom,
		model.ToBatchNotFound:
		return true
	default:
	}
	return false
}

func (h *Handler) cancelTransfer(ctx context.Context, transferID string, lastErr error) (model.StatusKind, error) {
	status, err := h.cancelTransferFrom(ctx, transferID)
	if err != nil {
		if lastErr != nil {
			err = errors.Errorf("%s: %w", err.Error(), lastErr)
		}
	} else {
		if lastErr != nil {
			err = lastErr
		}
	}
	return status, err
}

func (h *Handler) resolveStatus(ctx context.Context, transfer *fpb.CCTransfer) (model.StatusKind, error) {
	channelName := strings.ToLower(transfer.GetTo())
	if status, err := h.expandTO(ctx, channelName); err != nil {
		return status, err
	}

	hasTransferTo, err := h.queryChannelTransferTo(ctx, channelName, transfer.GetId())
	if err != nil {
		if h.expiredTransfer(transfer) {
			return model.CompletedTransferFrom, errors.Errorf("query transfer: %w", err)
		}
		return model.InternalErrorTransferStatus, errors.Errorf("query transfer: %w", err)
	}
	if hasTransferTo {
		if status, err := h.toBatchResponse(ctx, strings.ToLower(transfer.GetTo()), transfer.GetId()); err != nil {
			return status, err
		}
	}

	if !transfer.GetIsCommit() {
		if hasTransferTo {
			return model.CompletedTransferTo, nil
		}
		return h.fromBatchResponse(ctx, transfer.GetId())
	}

	if hasTransferTo {
		return model.CommitTransferFrom, nil
	}

	return model.CompletedTransferToDelete, nil
}

func (h *Handler) expiredTransfer(transfer *fpb.CCTransfer) bool {
	return time.Until(time.Unix(0, transfer.GetTimeAsNanos()).Add(h.ttl)) <= 0
}

func (h *Handler) fromBatchResponse(ctx context.Context, transferID string) (model.StatusKind, error) {
	var (
		batchResponse *fpb.TxResponse
		m             model.StatusKind
	)

	err := retry.Do(func() error {
		blocks, err := h.responseWithAttempt(ctx, h.channel, transferID)
		h.log.Debugf("find batchResponse in fromBatchResponse: %s; error: %v", transferID, err)
		if err != nil {
			err = errors.Errorf("batch FROM: %w", err)
			if strings.Contains(err.Error(), data.ErrObjectNotFound.Error()) {
				m = model.FromBatchNotFound
				return err
			}
			m = model.InternalErrorTransferStatus
			return err
		}

		for _, transaction := range blocks.Transactions {
			if (transaction.FuncName == model.TxChannelTransferByCustomer.String() ||
				transaction.FuncName == model.TxChannelTransferByAdmin.String() ||
				transaction.FuncName == model.TxChannelMultiTransferByCustomer.String() ||
				transaction.FuncName == model.TxChannelMultiTransferByAdmin.String()) &&
				transaction.BatchResponse != nil {
				batchResponse = transaction.BatchResponse
				return nil
			}
		}

		return errBatchFromNotFound
	},
		retry.LastErrorOnly(true),
		retry.Attempts(repeatAttempt),
		retry.Delay(sleepAttempt),
		retry.MaxDelay(maxSleepAttempt),
		retry.Context(ctx),
	)
	if err != nil {
		if errors.Is(err, errBatchFromNotFound) && batchResponse == nil {
			return model.InternalErrorTransferStatus, errBatchFromNotFound
		}
		return m, err
	}

	if batchResponse.GetError().GetCode() != 0 ||
		len(batchResponse.GetError().GetError()) != 0 {
		// delete transfer
		return model.ErrorTransferFrom, errors.New(batchResponse.GetError().GetError())
	}

	return model.CompletedTransferFrom, nil
}

func (h *Handler) toBatchResponse(ctx context.Context, channelName string, transferID string) (model.StatusKind, error) {
	var (
		batchResponse *fpb.TxResponse
		m             model.StatusKind
	)

	err := retry.Do(func() error {
		blocks, err := h.responseWithAttempt(ctx, channelName, transferID)
		h.log.Debugf("find batchResponse in toBatchResponse: %s; error: %v", transferID, err)
		if err != nil {
			err = errors.Errorf("batch TO: %w", err)
			if strings.Contains(err.Error(), data.ErrObjectNotFound.Error()) {
				m = model.ToBatchNotFound
				return err
			}
			m = model.InternalErrorTransferStatus
			return err
		}

		for _, transaction := range blocks.Transactions {
			if transaction.FuncName == model.TxCreateCCTransferTo.String() &&
				transaction.BatchResponse != nil {
				batchResponse = transaction.BatchResponse
				return nil
			}
		}

		return errBatchToNotFound
	},
		retry.LastErrorOnly(true),
		retry.Attempts(repeatAttempt),
		retry.Delay(sleepAttempt),
		retry.MaxDelay(maxSleepAttempt),
		retry.Context(ctx),
	)
	if err != nil {
		if errors.Is(err, errBatchToNotFound) && batchResponse == nil {
			return model.InternalErrorTransferStatus, errBatchToNotFound
		}
		return m, err
	}

	if batchResponse.GetError().GetCode() == 0 &&
		len(batchResponse.GetError().GetError()) == 0 {
		return model.CompletedTransferTo, nil
	}
	if batchResponse.GetError().GetError() == "id transfer already exists" {
		return model.CompletedTransferTo, nil
	}

	// delete transfer
	return model.ErrorTransferTo, errors.New(batchResponse.GetError().GetError())
}

func (h *Handler) responseWithAttempt(ctx context.Context, channel string, transferID string) (model.TransferBlock, error) {
	var resp model.TransferBlock
	err := retry.Do(func() error {
		blocks, err := h.blockStorage.BlockLoad(ctx, h.blockStorage.Key(model.ID(channel), model.ID(transferID)))
		h.log.Debugf("block load in responseWithAttempt: %s; error: %v", transferID, err)
		if err != nil {
			resp = model.TransferBlock{}
			return err
		}
		resp = blocks
		return nil
	},
		retry.LastErrorOnly(true),
		retry.Attempts(repeatAttempt),
		retry.Delay(sleepAttempt),
		retry.MaxDelay(maxSleepAttempt),
		retry.RetryIf(func(err error) bool {
			if err == nil {
				return false
			}
			return strings.Contains(err.Error(), data.ErrObjectNotFound.Error())
		}),
		retry.Context(ctx),
	)

	return resp, err
}

func (h *Handler) expandTO(ctx context.Context, channelTO string) (model.StatusKind, error) {
	if !h.poolController.Has(channelTO) {
		err := h.poolController.Expand(ctx, channelTO)
		if err != nil {
			if errMsg := re.FindString(err.Error()); errMsg != "" {
				return model.ErrorChannelToNotFound, errors.Errorf("expand : channel TO: %w", err)
			}
			return model.InternalErrorTransferStatus, errors.Errorf("expand: %w", err)
		}
	}

	return model.ExistsChannelTo, nil
}

func message(err error) string {
	if err != nil {
		return err.Error()
	}
	return ""
}
