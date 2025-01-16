package hlf

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/anoideaopen/channel-transfer/pkg/config"
	"github.com/anoideaopen/channel-transfer/pkg/data"
	"github.com/anoideaopen/channel-transfer/pkg/data/redis"
	"github.com/anoideaopen/channel-transfer/pkg/helpers/methods"
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
	"github.com/go-errors/errors"
	hlfcontext "github.com/hyperledger/fabric-sdk-go/pkg/common/providers/context"
	"github.com/hyperledger/fabric-sdk-go/pkg/fabsdk"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"google.golang.org/grpc/backoff"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"
)

type Pool struct {
	userName       string
	profilePath    string
	hlfProfile     hlfprofile.HlfProfile
	opts           config.Options
	additiveToTTL  time.Duration
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
	events  map[string]chan struct{}
}

func NewPool(
	ctx context.Context,
	channels []config.Channel,
	userName string,
	opts config.Options,
	profilePath string,
	profile hlfprofile.HlfProfile,
	storage *redis.Storage,
) (*Pool, error) {
	log := glog.FromContext(ctx).With(logger.Labels{Component: logger.ComponentHLFStreamsPool}.Fields()...)

	m := metrics.FromContext(ctx)

	pool := &Pool{
		profilePath:    profilePath,
		userName:       userName,
		opts:           opts,
		hlfProfile:     profile,
		blocKStorage:   transfer.NewLedgerBlock(storage),
		checkPoint:     transfer.NewBlockCheckpoint(storage),
		requestStorage: transfer.NewRequest(storage),
		log:            log,
		m:              m,
		streams:        createStreams(ctx),
		events:         make(map[string]chan struct{}),
	}

	var err error
	pool.fabricSDK, err = createFabricSDK(profilePath)
	if err != nil {
		return nil, errorshlp.WrapWithDetails(
			fmt.Errorf("create connection to fabric: %w", err),
			nerrors.ErrTypeHlf,
			nerrors.ComponentHLFStreamsPool,
		)
	}

	pool.additiveToTTL = storage.TTL() - *opts.TTL
	if pool.additiveToTTL < 0 {
		return nil, errorshlp.WrapWithDetails(errors.New("upper bound of ttl is small, change ttl option"), nerrors.ErrTypeHlf, nerrors.ComponentHLFStreamsPool)
	}

	// map of gRPC clients for every gRPC address in channels config
	gRPCClients := make(map[string]*grpc.ClientConn)
	for _, channel := range channels {
		channelProvider := createChannelProvider(channel.Name, userName, profile.OrgName, pool.fabricSDK)

		// if the channel has batcher options, create the client for it
		var gRPCClient *grpc.ClientConn = nil
		if channel.TaskExecutor != nil {
			var ok bool
			// check if a client for the gRPC address already created
			gRPCClient, ok = gRPCClients[channel.TaskExecutor.AddressGRPC]
			if !ok {
				// if not, create the new one
				gRPCClient, err = createGRPCClient(channel.TaskExecutor)
				if err != nil {
					return nil, errorshlp.WrapWithDetails(fmt.Errorf("create gRPC client: %w", err), nerrors.ErrTypeHlf, nerrors.ComponentHLFStreamsPool)
				}
				gRPCClients[channel.TaskExecutor.AddressGRPC] = gRPCClient
			}
		}

		err = pool.createExecutor(ctx, channel.Name, channelProvider, gRPCClient)
		if err != nil {
			return nil, errorshlp.WrapWithDetails(
				fmt.Errorf("create executor: %w", err),
				nerrors.ErrTypeHlf,
				nerrors.ComponentHLFStreamsPool,
			)
		}

		pool.events[channel.Name] = make(chan struct{}, 1)
	}

	return pool, nil
}

func newGRPCClient(opts *config.TaskExecutor) (*grpc.ClientConn, error) {
	kacp := keepalive.ClientParameters{
		Time:                10 * time.Second, // send pings every 10 seconds if there is no activity
		Timeout:             time.Second,      // wait 1 second for ping ack before considering the connection dead
		PermitWithoutStream: true,             // send pings even without active streams
	}

	bOff := backoff.Config{
		BaseDelay:  1.0 * time.Second,
		Multiplier: 1.6,
		Jitter:     0.2,
		MaxDelay:   30 * time.Second,
	}

	dialOpts := []grpc.DialOption{
		grpc.WithKeepaliveParams(kacp),
		grpc.WithConnectParams(grpc.ConnectParams{Backoff: bOff}),
	}

	if opts.TLSConfig() != nil {
		dialOpts = append(dialOpts, grpc.WithTransportCredentials(credentials.NewTLS(opts.TLSConfig())))
	} else {
		dialOpts = append(dialOpts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	}

	client, err := grpc.NewClient(opts.AddressGRPC, dialOpts...)
	if err != nil {
		return nil, err
	}

	return client, err
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
			pool.log.Error(errors.New(err))
		}

		pool.log.Info("stop channels collectors")
	}

	return ctx.Err()
}

func (pool *Pool) close() {
	pool.streams.close()
	pool.fabricSDK.Close()
}

func (pool *Pool) createExecutor(
	ctx context.Context,
	channel string,
	channelProvider hlfcontext.ChannelProvider,
	gRPCClient *grpc.ClientConn,
) error {
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
		return fmt.Errorf("start executor: %w", err)
	}

	if gRPCClient != nil {
		executor.gRPCExecutor = &gRPCExecutor{
			Channel: channel,
			Client:  gRPCClient,
		}
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

	channelProvider := createChannelProvider(channel, pool.userName, pool.hlfProfile.OrgName, pool.fabricSDK)

	err := pool.createExecutor(ctx, channel, channelProvider, nil)
	if err != nil {
		return errorshlp.WrapWithDetails(err, nerrors.ErrTypeHlf, nerrors.ComponentHLFStreamsPool)
	}

	if pool.group != nil {
		pool.group.Go(func() error {
			return pool.blockKeeper(key, channelProvider)
		})
		ready, err := pool.Readiness(channel)
		if err != nil {
			return errorshlp.WrapWithDetails(
				fmt.Errorf("pool readiness: %w", err),
				nerrors.ErrTypeHlf,
				nerrors.ComponentHLFStreamsPool,
			)
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
	var blockNumber uint64

	checkPoint, err := pool.checkPoint.CheckpointLoad(pool.gCtx, model.ID(key))
	if err != nil {
		if !errors.Is(err, data.ErrObjectNotFound) {
			pool.log.Error(fmt.Errorf("load checkpoint of %s: %w", string(key), err))
		}
	} else {
		blockNumber = checkPoint.SrcCollectFromBlockNums
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
		return fmt.Errorf("create channel %s hasCollector, get blockchain height: %w", string(key), err)
	}
	readiness := func() {
		if *bcHeight-blockNumber <= 1 {
			pool.streams.ready(key)
		}
	}
	readiness()

	pool.m.TotalReconnectsToFabric().Inc(metrics.Labels().Channel.Create(string(key)))

	for pool.gCtx.Err() == nil {
		select {
		case <-pool.gCtx.Done():
			return errors.New(pool.gCtx.Err())
		case block, ok := <-collector.GetData():
			if !ok {
				return errors.Errorf("hasCollector of chan %s closed: %s", string(key), "get hasCollector data")
			}
			pool.log.Debugf("store block: %d", block.BlockNum)
			if err = pool.storeTransfer(key, *block); err != nil {
				return fmt.Errorf("store block to redis: %w", err)
			}
			blockNumber = block.BlockNum
			readiness()
			// saving event checkpoint
			checkPoint = pool.saveCheckPoint(checkPoint, key, blockNumber)
		}
	}

	return errors.New(pool.gCtx.Err())
}

func (pool *Pool) saveCheckPoint(oldCheckPoint model.Checkpoint, key channelKey, blockNumber uint64) model.Checkpoint {
	nCheckPoint, err := pool.checkPoint.CheckpointSave(
		pool.gCtx,
		model.Checkpoint{
			Ver:                     oldCheckPoint.Ver + 1,
			Channel:                 model.ID(key),
			SrcCollectFromBlockNums: blockNumber,
		},
	)
	if err != nil {
		pool.log.Errorf("save checkpoint of %s : %s", string(key), err.Error())
		return oldCheckPoint
	}

	return nCheckPoint
}

//nolint:gocognit
func (pool *Pool) storeTransfer(key channelKey, block model.BlockData) error {
	if len(block.Txs) == 0 {
		return nil
	}

	transferBlocks := transfer.LedgerBlockToTransferBlock(string(key), block)

	for transferID, transferBlock := range transferBlocks {
		var (
			ttl         = redis.TTLNotTakenInto
			canBeStored = false
			isSendEvent = false
			isFullTx    = false
		)

		for _, transaction := range transferBlock.Transactions {
			isCreateMethod := methods.IsTransferFromMethod(transaction.FuncName)
			isFullTx = isFullTx || transaction.BatchResponse != nil && transaction.TimeNs != 0

			if isCreateMethod && transaction.BatchResponse != nil && !isSendEvent {
				isSendEvent = true
				pool.sendEvent(string(key))
			}

			if transaction.BatchResponse != nil && !isFullTx {
				continue
			}

			upperBound := time.Unix(0, int64(transaction.TimeNs)).Add(*pool.opts.TTL + pool.additiveToTTL)
			if time.Now().After(upperBound) {
				return nil
			}

			ttl = time.Until(upperBound)

			if !isFullTx && !pool.streams.transferID(key, transactionID(transaction.TxID), transferID) {
				return fmt.Errorf("streams buffer : channel %s not found", string(key))
			}

			canBeStored = isCreateMethod
		}

		if transferID == "" {
			if err := pool.updateBatchResponse(key, block.Txs); err != nil {
				return err
			}
			continue
		}

		if canBeStored {
			if err := pool.syncTransferRequest(*transferBlock, ttl); err != nil {
				return fmt.Errorf("sync transfer request: %w", err)
			}
		}

		pool.log.Debugf("block save in storeTransfer %s, channel %s",
			transferBlock.Transfer, transferBlock.Channel,
		)
		if err := pool.blocKStorage.BlockSave(pool.gCtx, *transferBlock, ttl); err != nil {
			return err
		}

		if isFullTx {
			pool.updateTransferResult(transferBlock)
		}
	}
	return nil
}

func (pool *Pool) updateTransferResult(transferBlock *model.TransferBlock) {
	for _, tx := range transferBlock.Transactions {
		if !methods.IsTransferFromMethod(tx.FuncName) {
			continue
		}

		response := tx.BatchResponse

		if response.GetError() == nil {
			if err := pool.requestStorage.TransferResultModify(
				pool.gCtx,
				transferBlock.Transfer,
				model.TransferResult{
					Status: proto2.TransferStatusResponse_STATUS_IN_PROCESS.String(),
				},
			); err != nil {
				pool.log.Errorf("transfer result no updated : %s : %s", transferBlock.Transfer, err.Error())
			}
			continue
		}

		// the execution of the transaction ended with an error
		if err := pool.requestStorage.TransferResultModify(
			pool.gCtx,
			transferBlock.Transfer,
			model.TransferResult{
				Status:  proto2.TransferStatusResponse_STATUS_ERROR.String(),
				Message: response.GetError().GetError(),
			},
		); err != nil {
			pool.log.Errorf("transfer result no updated : %s : %s", transferBlock.Transfer, err.Error())
		}
	}
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

				if !methods.IsTransferFromMethod(transferBlock.Transactions[i].FuncName) {
					continue
				}

				if response.GetError() == nil {
					continue
				}

				// the execution of the transaction preimage in batch ended with an error
				if err = pool.requestStorage.TransferResultModify(
					pool.gCtx,
					transferBlock.Transfer,
					model.TransferResult{
						Status:  proto2.TransferStatusResponse_STATUS_ERROR.String(),
						Message: response.GetError().GetError(),
					},
				); err != nil {
					pool.log.Errorf("transfer response status not saved : %s : %s", transferBlock.Transfer, err.Error())
				}
			}
		}

		pool.log.Debugf("block save in updateBatchResponse %s, channel %s",
			transferBlock.Transfer, transferBlock.Channel,
		)
		if err = pool.blocKStorage.BlockSave(pool.gCtx, transferBlock, redis.TTLNotTakenInto); err != nil {
			return err
		}
	}

	return nil
}

func (pool *Pool) syncTransferRequest(block model.TransferBlock, ttl time.Duration) error {
	request, err := transfer.BlockToRequest(block)
	if err != nil {
		return err
	}
	return pool.requestStorage.TransferModify(pool.gCtx, request, ttl)
}

func (pool *Pool) Readiness(channel string) (<-chan struct{}, error) {
	ready, ok := pool.streams.done(channelKey(channel))
	if ok {
		return ready, nil
	}

	return nil, errors.New("processed channel not found")
}

func (pool *Pool) Events(channel string) (<-chan struct{}, error) {
	ev, ok := pool.events[channel]
	if ok {
		return ev, nil
	}

	return nil, errors.New("processed channel not found")
}

func (pool *Pool) sendEvent(channel string) {
	ev, ok := pool.events[channel]
	if !ok {
		return
	}

	select {
	case ev <- struct{}{}:
	default:
	}
}

func createGRPCClient(options *config.TaskExecutor) (*grpc.ClientConn, error) {
	gRPCClient, err := newGRPCClient(options)
	if err != nil {
		return nil, fmt.Errorf("create grpc client: %w", err)
	}

	return gRPCClient, nil
}
