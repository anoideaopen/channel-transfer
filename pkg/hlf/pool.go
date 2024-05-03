package hlf

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/anoideaopen/cartridge/manager"
	"github.com/anoideaopen/channel-transfer/pkg/config"
	"github.com/anoideaopen/channel-transfer/pkg/data"
	"github.com/anoideaopen/channel-transfer/pkg/data/redis"
	"github.com/anoideaopen/channel-transfer/pkg/helpers/nerrors"
	"github.com/anoideaopen/channel-transfer/pkg/hlf/hlfprofile"
	"github.com/anoideaopen/channel-transfer/pkg/logger"
	"github.com/anoideaopen/channel-transfer/pkg/metrics"
	"github.com/anoideaopen/channel-transfer/pkg/model"
	"github.com/anoideaopen/channel-transfer/pkg/transfer"
	proto2 "github.com/anoideaopen/channel-transfer/proto"
	"github.com/anoideaopen/common-component/errorshlp"
	"github.com/anoideaopen/foundation/proto"
	"github.com/anoideaopen/glog"
	hlfcontext "github.com/hyperledger/fabric-sdk-go/pkg/common/providers/context"
	"github.com/hyperledger/fabric-sdk-go/pkg/fabsdk"
	"github.com/pkg/errors"
	"golang.org/x/sync/errgroup"
)

type Pool struct {
	userName       string
	profilePath    string
	hlfProfile     hlfprofile.HlfProfile
	opts           config.Options
	additiveToTTL  time.Duration
	cryptoManager  manager.Manager
	blocKStorage   *transfer.LedgerBlock
	checkPoint     *transfer.BlockCheckpoint
	requestStorage *transfer.Request
	fabricSDK      *fabsdk.FabricSDK

	log     glog.Logger
	m       metrics.Metrics
	streams *streams
	group   *errgroup.Group
	gCtx    context.Context
	mutex   sync.RWMutex
}

func NewPool(
	ctx context.Context,
	channels []string,
	userName string,
	opts config.Options,
	profilePath string,
	profile hlfprofile.HlfProfile,
	cryptoManager manager.Manager,
	storage *redis.Storage,
) (*Pool, error) {
	log := glog.FromContext(ctx).With(logger.Labels{Component: logger.ComponentHLFStreamsPool}.Fields()...)

	m := metrics.FromContext(ctx)

	pool := &Pool{
		profilePath:    profilePath,
		userName:       userName,
		opts:           opts,
		hlfProfile:     profile,
		cryptoManager:  cryptoManager,
		blocKStorage:   transfer.NewLedgerBlock(storage),
		checkPoint:     transfer.NewBlockCheckpoint(storage),
		requestStorage: transfer.NewRequest(storage),
		log:            log,
		m:              m,
		streams:        createStreams(ctx),
	}

	var err error
	pool.fabricSDK, err = createFabricSDK(profilePath, cryptoManager)
	if err != nil {
		return nil, errorshlp.WrapWithDetails(errors.Wrap(err, "create connection to fabric"), nerrors.ErrTypeHlf, nerrors.ComponentHLFStreamsPool)
	}

	pool.additiveToTTL = storage.TTL() - *opts.TTL
	if pool.additiveToTTL < 0 {
		return nil, errorshlp.WrapWithDetails(errors.New("upper bound of ttl is small, change ttl option"), nerrors.ErrTypeHlf, nerrors.ComponentHLFStreamsPool)
	}

	for _, channel := range channels {
		channelProvider := createChannelProvider(channel, userName, profile.OrgName, cryptoManager, pool.fabricSDK)
		err = pool.createExecutor(ctx, channel, channelProvider)
		if err != nil {
			return nil, errorshlp.WrapWithDetails(errors.Wrap(err, "create executor"), nerrors.ErrTypeHlf, nerrors.ComponentHLFStreamsPool)
		}
	}

	return pool, nil
}

func (pool *Pool) RunCollectors(ctx context.Context) error {
	defer pool.close()

	for ctx.Err() == nil {
		pool.group, pool.gCtx = errgroup.WithContext(ctx)

		pool.log.Info("start channels collectors")

		pool.streams.loop(func(key channelKey, stream *biDirectFlow) {
			if !stream.hasCollector {
				pool.group.Go(func() error {
					return pool.blockKeeper(key, stream.channelProvider)
				})
			}
		})

		if err := pool.group.Wait(); err != nil {
			pool.log.Error(errors.WithStack(err))
		}

		pool.log.Info("stop channels collectors")
	}

	return ctx.Err()
}

func (pool *Pool) close() {
	pool.streams.close()
	pool.fabricSDK.Close()
}

func (pool *Pool) createExecutor(ctx context.Context, channel string, channelProvider hlfcontext.ChannelProvider) error {
	if pool.streams.exists(channelKey(channel)) {
		return ErrExecutorAlreadyExists
	}

	executor, err := createChExecutor(
		ctx,
		channel,
		channelProvider,
		ExecuteOptions{
			ExecuteTimeout:       *pool.opts.ExecuteTimeout,
			RetryExecuteAttempts: pool.opts.RetryExecuteAttempts,
			RetryExecuteMaxDelay: *pool.opts.RetryExecuteMaxDelay,
			RetryExecuteDelay:    *pool.opts.RetryExecuteDelay,
		},
	)
	if err != nil {
		pool.m.TotalReconnectsToFabric().Inc(metrics.Labels().Channel.Create(channel))
		return errors.Wrap(err, "start executor")
	}

	pool.streams.store(channelKey(channel), executor, channelProvider)

	return nil
}

func (pool *Pool) Executor(channel string) (*ChExecutor, error) {
	ps, ok := pool.streams.load(channelKey(channel))
	if !ok {
		return nil, errorshlp.WrapWithDetails(ErrExecutorUndefined, nerrors.ErrTypeHlf, nerrors.ComponentHLFStreamsPool)
	}
	return ps.executor, nil
}

func (pool *Pool) Has(channel string) bool {
	return pool.streams.exists(channelKey(channel))
}

func (pool *Pool) Expand(ctx context.Context, channel string) error {
	pool.mutex.Lock()
	defer pool.mutex.Unlock()

	key := channelKey(channel)

	if pool.streams.exists(key) {
		return nil
	}

	channelProvider := createChannelProvider(channel, pool.userName, pool.hlfProfile.OrgName, pool.cryptoManager, pool.fabricSDK)

	err := pool.createExecutor(ctx, channel, channelProvider)
	if err != nil {
		return errorshlp.WrapWithDetails(err, nerrors.ErrTypeHlf, nerrors.ComponentHLFStreamsPool)
	}

	if pool.group != nil {
		pool.group.Go(func() error {
			return pool.blockKeeper(key, channelProvider)
		})
		ready, err := pool.Readiness(channel)
		if err != nil {
			return errorshlp.WrapWithDetails(errors.Wrap(err, "pool readiness"), nerrors.ErrTypeHlf, nerrors.ComponentHLFStreamsPool)
		}
		<-ready
	}

	return nil
}

func (pool *Pool) blockchainHeight(key channelKey) (*uint64, error) {
	if ps, ok := pool.streams.load(key); ok {
		return ps.executor.BlockchainHeight(pool.gCtx)
	}

	return nil, ErrExecutorUndefined
}

//nolint:funlen
func (pool *Pool) blockKeeper(key channelKey, provider hlfcontext.ChannelProvider) error {
	blockNumber := uint64(startFromZero)
	checkPointVersion := int64(0)

	checkPoint, err := pool.checkPoint.CheckpointLoad(pool.gCtx, model.ID(key))
	if err != nil {
		if !errors.Is(err, data.ErrObjectNotFound) {
			pool.log.Error(errors.Wrapf(err, "load checkpoint of %s", string(key)))
		}
	} else {
		blockNumber = checkPoint.SrcCollectFromBlockNums
		checkPointVersion = checkPoint.Ver
	}

	collector := createChCollector(pool.gCtx, string(key), blockNumber, provider, pool.opts.BatchTxPreimagePrefix, pool.opts.CollectorsBufSize)
	defer func() {
		collector.Close()
		_ = pool.streams.collector(key, false)
	}()

	if !pool.streams.collector(key, true) {
		return fmt.Errorf("streams buffer : channel %s not found in buffer", string(key))
	}

	bcHeight, err := pool.blockchainHeight(key)
	if err != nil {
		return errors.Wrapf(err, "create channel %s hasCollector : get blockchain height", string(key))
	}
	readiness := func() {
		if *bcHeight-blockNumber <= 1 {
			pool.streams.ready(key)
		}
	}
	readiness()

	saver := time.NewTicker(checkpointFrequencySaver)
	defer saver.Stop()

	pool.m.TotalReconnectsToFabric().Inc(metrics.Labels().Channel.Create(string(key)))

	for pool.gCtx.Err() == nil {
		select {
		case <-pool.gCtx.Done():
			return errors.WithStack(pool.gCtx.Err())
		case block, ok := <-collector.GetData():
			if !ok {
				return errors.Wrapf(fmt.Errorf("hasCollector of chan %s closed", string(key)), "get hasCollector data")
			}
			if err = pool.storeTransfer(key, *block); err != nil {
				return errors.Wrap(err, "store block to redis")
			}
			blockNumber = block.BlockNum
			readiness()
		case <-saver.C:
			go func() {
				nCheckPoint, err := pool.checkPoint.CheckpointSave(
					pool.gCtx,
					model.Checkpoint{
						Ver:                     checkPointVersion,
						Channel:                 model.ID(key),
						SrcCollectFromBlockNums: blockNumber,
					},
				)
				if err != nil {
					pool.log.Errorf("save checkpoint of %s : %s", string(key), err.Error())
				} else {
					checkPointVersion = nCheckPoint.Ver
				}
			}()
		}
	}

	return errors.WithStack(pool.gCtx.Err())
}

func (pool *Pool) storeTransfer(key channelKey, block model.BlockData) error {
	if len(block.Txs) == 0 {
		return nil
	}

	transferBlocks := transfer.LedgerBlockToTransferBlock(string(key), block)

	for transferID, transferBlock := range transferBlocks {
		ttl := redis.TTLNotTakenInto
		canBeStored := false

		for _, transaction := range transferBlock.Transactions {
			if transaction.BatchResponse != nil {
				continue
			}

			upperBound := time.Unix(0, int64(transaction.TimeNs)).Add(*pool.opts.TTL + pool.additiveToTTL)
			if time.Now().After(upperBound) {
				return nil
			}

			ttl = time.Until(upperBound)

			if !pool.streams.transferID(key, transactionID(transaction.TxID), transferID) {
				return fmt.Errorf("streams buffer : channel %s not found", string(key))
			}

			if transaction.FuncName == model.TxChannelTransferByCustomer.String() ||
				transaction.FuncName == model.TxChannelTransferByAdmin.String() {
				canBeStored = true
			}
		}

		if transferID == "" {
			if err := pool.updateBatchResponse(key, block.Txs); err != nil {
				return err
			}
			continue
		}

		if canBeStored {
			if err := pool.syncTransferRequest(*transferBlock, ttl); err != nil {
				return errors.Wrap(err, "sync transfer request")
			}
		}

		err := pool.blocKStorage.BlockSave(pool.gCtx, *transferBlock, ttl)
		if err != nil {
			return err
		}
	}
	return nil
}

//nolint:gocognit,funlen
func (pool *Pool) updateBatchResponse(key channelKey, transactions []model.Transaction) error {
	batchResponses := make(map[string]*proto.TxResponse)
	transferIDs := make([]model.ID, 0, len(transactions))

	for _, tx := range transactions {
		if tid, ok := pool.streams.transactionID(key, transactionID(tx.TxID)); ok {
			transferIDs = append(transferIDs, tid)
			if tx.BatchResponse != nil {
				batchResponses[tx.TxID] = tx.BatchResponse
			}
		}
	}

	for _, transferID := range transferIDs {
		transferBlock, err := pool.blocKStorage.BlockLoad(pool.gCtx, pool.blocKStorage.Key(model.ID(key), transferID))
		if err != nil {
			return err
		}

		for i := range transferBlock.Transactions {
			for _, tx := range transactions {
				if tx.TxID != transferBlock.Transactions[i].TxID {
					continue
				}

				response, ok := batchResponses[tx.TxID]
				if !ok {
					continue
				}

				transferBlock.Transactions[i].BatchResponse = response

				pool.streams.removeTransactionID(key, transactionID(tx.TxID))

				if transferBlock.Transactions[i].FuncName != model.TxChannelTransferByCustomer.String() &&
					transferBlock.Transactions[i].FuncName != model.TxChannelTransferByAdmin.String() {
					continue
				}

				if response.Error == nil {
					continue
				}

				// the execution of the transaction preimage in batch ended with an error
				if err = pool.requestStorage.TransferResultModify(
					pool.gCtx,
					transferBlock.Transfer,
					model.TransferResult{
						Status:  proto2.TransferStatusResponse_STATUS_ERROR.String(),
						Message: response.Error.GetError(),
					},
				); err != nil {
					pool.log.Errorf("transfer response status not saved : %s : %s", transferBlock.Transfer, err.Error())
				}
			}
		}

		if err = pool.blocKStorage.BlockSave(pool.gCtx, transferBlock, redis.TTLNotTakenInto); err != nil {
			return err
		}
	}

	return nil
}

func (pool *Pool) syncTransferRequest(block model.TransferBlock, ttl time.Duration) error {
	return pool.requestStorage.TransferModify(pool.gCtx, transfer.BlockToRequest(block), ttl)
}

func (pool *Pool) Readiness(channel string) (<-chan struct{}, error) {
	ready, ok := pool.streams.done(channelKey(channel))
	if ok {
		return ready, nil
	}

	return nil, errors.New("processed channel not found")
}
