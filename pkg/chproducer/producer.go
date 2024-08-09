package chproducer

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/anoideaopen/channel-transfer/pkg/data"
	"github.com/anoideaopen/channel-transfer/pkg/data/redis"
	"github.com/anoideaopen/channel-transfer/pkg/helpers/nerrors"
	"github.com/anoideaopen/channel-transfer/pkg/hlf"
	"github.com/anoideaopen/channel-transfer/pkg/logger"
	"github.com/anoideaopen/channel-transfer/pkg/metrics"
	"github.com/anoideaopen/channel-transfer/pkg/model"
	"github.com/anoideaopen/channel-transfer/pkg/transfer"
	"github.com/anoideaopen/channel-transfer/proto"
	"github.com/anoideaopen/common-component/errorshlp"
	"github.com/anoideaopen/glog"
	"github.com/go-errors/errors"
	"go.uber.org/atomic"
	"golang.org/x/sync/errgroup"
)

const checkFrequencyTransfers = time.Duration(3) * time.Second

var ErrChanClosed = errors.New("channel closed")

type BlockController interface {
	GetData() <-chan *model.BlockData
	Close()
	StreamReady() <-chan struct{}
}

type PoolController interface {
	Executor(channel string) (*hlf.ChExecutor, error)
	Has(channel string) bool
	Expand(ctx context.Context, channel string) error
	Readiness(channel string) (<-chan struct{}, error)
	Events(channel string) (<-chan struct{}, error)
}

type HealthcheckController interface {
	State(bool)
}

type Handler struct {
	channel         string
	chaincodeID     string
	ttl             time.Duration // in seconds
	activeTransfers uint
	blockStorage    *transfer.LedgerBlock
	requestStorage  *transfer.Request
	log             glog.Logger
	m               metrics.Metrics
	poolController  PoolController
	newest          <-chan model.TransferRequest // incoming api request
	inUsed          sync.Map
	launcherInWork  atomic.Bool
}

func NewHandler(
	ctx context.Context,
	chName string,
	ttl time.Duration,
	activeTransfers uint,
	storage *redis.Storage,
	poolController PoolController,
	newest <-chan model.TransferRequest,
) (*Handler, error) {
	log := glog.FromContext(ctx).With(logger.Labels{Component: logger.ComponentProducer, Channel: chName}.Fields()...)

	m := metrics.FromContext(ctx)
	m = m.CreateChild(
		metrics.Labels().Channel.Create(chName),
	)

	h := &Handler{
		channel:         chName,
		ttl:             ttl,
		activeTransfers: activeTransfers,
		poolController:  poolController,
		blockStorage:    transfer.NewLedgerBlock(storage),
		requestStorage:  transfer.NewRequest(storage),
		log:             log,
		m:               m,
		newest:          newest,
	}

	h.chaincodeID = h.channel

	return h, nil
}

func (h *Handler) Exec(ctx context.Context) error {
	group, gCtx := errgroup.WithContext(ctx)
	group.SetLimit(int(h.activeTransfers))

	ticker := time.NewTicker(checkFrequencyTransfers)
	defer ticker.Stop()

	ready, err := h.poolController.Readiness(h.channel)
	if err != nil {
		return errorshlp.WrapWithDetails(
			errors.Errorf("pool readiness: %w", err),
			nerrors.ErrTypeProducer,
			nerrors.ComponentProducer,
		)
	}
	<-ready

	h.syncAPIRequests(ctx)

	event, _ := h.poolController.Events(h.channel)

	for ctx.Err() == nil {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case request, ok := <-h.newest:
			if !ok {
				return errorshlp.WrapWithDetails(
					errors.Errorf("incoming api request: %w", ErrChanClosed),
					nerrors.ErrTypeProducer,
					nerrors.ComponentProducer,
				)
			}
			go func() {
				h.createTransfer(gCtx, request)
			}()
		case <-ticker.C:
			go h.launcher(gCtx, group)
		case <-event:
			go h.launcher(gCtx, group)
		}
	}

	return ctx.Err()
}

func (h *Handler) launcher(ctx context.Context, group *errgroup.Group) {
	if !h.launcherInWork.CAS(false, true) {
		return
	}
	defer h.launcherInWork.Store(false)

	ccTransfers, err := h.queryChannelTransfers(ctx)
	if err != nil {
		h.log.Error(errors.Errorf("stop launcher: query transfers: %w", err))
		return
	}

	counter := 0
	for _, ccTransfer := range ccTransfers {
		if _, ok := h.inUsed.Load(ccTransfer.GetId()); ok {
			continue
		}

		localTransfer := ccTransfer
		h.inUsed.Store(localTransfer.GetId(), struct{}{})
		if group.TryGo(
			func() error {
				defer h.inUsed.Delete(localTransfer.GetId())

				status, err := h.resolveStatus(ctx, localTransfer)
				if err != nil {
					h.log.Error(errors.Errorf("resolve transfer status %s: %w", localTransfer.GetId(), err))
				}
				h.restoreCompletedStatus(ctx, status, model.ID(localTransfer.GetId()))

				err = h.transferProcessing(ctx, status, localTransfer, err)
				if err != nil {
					h.log.Error(errors.Errorf("transfer processing %s: %w", localTransfer.GetId(), err))
				}

				return nil
			},
		) {
			counter++
		} else {
			h.inUsed.Delete(localTransfer.GetId())
		}
	}

	if counter > 0 {
		h.log.Debugf("%d transfers launched", counter)
	}
}

func (h *Handler) createTransfer(ctx context.Context, request model.TransferRequest) {
	var status model.StatusKind
	var err error
	if len(request.Items) > 0 {
		status, err = h.createMultiTransferFrom(ctx, request)
	} else {
		status, err = h.createTransferFrom(ctx, request)
	}
	if err != nil {
		err = errors.Errorf("create transfer: %w", err)
		request.Message = err.Error()
		h.log.Error(errorshlp.WrapWithDetails(err, nerrors.ErrTypeProducer, nerrors.ComponentProducer))
	}

	switch status {
	case model.InProgressTransferFrom:
		request.Status = proto.TransferStatusResponse_STATUS_IN_PROCESS.String()
	case model.ErrorTransferFrom, model.InternalErrorTransferStatus:
		request.Status = proto.TransferStatusResponse_STATUS_ERROR.String()
	default:
		request.Status = proto.TransferStatusResponse_STATUS_UNDEFINED.String()
	}

	h.log.Debugf("transfer modify: %s", request.Transfer)
	if err = h.requestStorage.TransferResultModify(
		ctx,
		request.Transfer,
		request.TransferResult,
	); err != nil {
		h.log.Errorf("transfer response status not saved : %s : %s", request.Transfer, err.Error())
	}
}

func (h *Handler) syncAPIRequests(ctx context.Context) {
	registry, err := h.requestStorage.Registry(ctx)
	if err != nil {
		h.log.Error(errorshlp.WrapWithDetails(errors.Errorf("scan requests: %w", err), nerrors.ErrTypeProducer, nerrors.ComponentProducer))
		return
	}

	for _, request := range registry {
		if _, ok := h.inUsed.Load(request.Transfer); ok {
			// active transfer not handle
			continue
		}

		if !h.requestStorage.IsChangeableStatus(request.Status) {
			continue
		}

		ok, err := h.queryChannelTransferFrom(ctx, strings.ToLower(h.channel), string(request.Transfer))
		if err != nil || ok {
			// metadata not removed, transfer will be processed later in the transferProcessing
			continue
		}

		channelTO := strings.ToLower(request.To)

		status, err := h.expandTO(ctx, channelTO)
		if status == model.InternalErrorTransferStatus {
			continue
		}

		if status == model.ExistsChannelTo {
			status, err = h.toBatchResponse(ctx, channelTO, string(request.Transfer))
		}

		switch status {
		case model.ToBatchNotFound, model.ErrorTransferTo, model.ErrorChannelToNotFound:
			request.TransferResult.Status = proto.TransferStatusResponse_STATUS_ERROR.String()
			if err != nil {
				request.TransferResult.Message = err.Error()
			}
		case model.CompletedTransferTo:
			request.TransferResult.Status = proto.TransferStatusResponse_STATUS_COMPLETED.String()
		case model.InternalErrorTransferStatus:
			if !errors.Is(err, data.ErrObjectNotFound) {
				continue
			}
			request.TransferResult.Status = proto.TransferStatusResponse_STATUS_CANCELED.String()
		default:
			continue
		}

		if err = h.requestStorage.TransferResultModify(ctx, request.Transfer, request.TransferResult); err != nil {
			h.log.Errorf("transfer response status not saved : %s : %s", request.Transfer, err.Error())
		}
	}
	h.log.Debug("sync api requests")
}

func (h *Handler) restoreCompletedStatus(ctx context.Context, status model.StatusKind, transferID model.ID) {
	if status != model.CompletedTransferTo &&
		status != model.CompletedTransferToDelete &&
		status != model.CommitTransferFrom {
		return
	}
	// repair completed status
	if err := h.requestStorage.TransferResultModify(
		ctx,
		transferID,
		model.TransferResult{
			Status: proto.TransferStatusResponse_STATUS_COMPLETED.String(),
		},
	); err != nil {
		h.log.Errorf("transfer response status not saved : %s : %s", transferID, err.Error())
	}
}
