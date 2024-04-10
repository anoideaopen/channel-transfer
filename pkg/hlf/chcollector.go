package hlf

import (
	"context"
	"sync"
	"time"

	"github.com/anoideaopen/channel-transfer/pkg/hlf/parser"
	"github.com/anoideaopen/channel-transfer/pkg/logger"
	"github.com/anoideaopen/channel-transfer/pkg/metrics"
	"github.com/anoideaopen/channel-transfer/pkg/model"
	hlfcontext "github.com/hyperledger/fabric-sdk-go/pkg/common/providers/context"
	"github.com/hyperledger/fabric-sdk-go/pkg/common/providers/fab"
	"github.com/newity/glog"
)

type chCollector struct {
	log                glog.Logger
	m                  metrics.Metrics
	srcChName          string
	batchPrefix        string
	initStartFrom      uint64
	delayAfterSrcError time.Duration

	cancel context.CancelFunc
	wgLoop *sync.WaitGroup
	stream chan *model.BlockData
}

const (
	defaultDelayAfterSrcError = 5 * time.Second
)

func createChCollector(
	ctx context.Context,
	channel string,
	startFrom uint64,
	channelProvider hlfcontext.ChannelProvider,
	batchPrefix string,
	streamBufSize uint,
) *chCollector {
	log := glog.FromContext(ctx).With(logger.Labels{Component: logger.ComponentCollector, Channel: channel}.Fields()...)

	m := metrics.FromContext(ctx).CreateChild(metrics.Labels().Channel.Create(channel))

	chColl := &chCollector{
		log:                log,
		m:                  m,
		srcChName:          channel,
		initStartFrom:      startFrom,
		batchPrefix:        batchPrefix,
		wgLoop:             &sync.WaitGroup{},
		delayAfterSrcError: defaultDelayAfterSrcError,
		stream:             make(chan *model.BlockData, streamBufSize),
	}

	loopCtx, cancel := context.WithCancel(ctx)
	chColl.cancel = cancel
	chColl.wgLoop.Add(1)

	go chColl.loopProxy(loopCtx, channelProvider)

	return chColl
}

func (cc *chCollector) loopProxy(ctx context.Context, channelProvider hlfcontext.ChannelProvider) {
	defer func() {
		cc.log.Info("stop collector")
		cc.wgLoop.Done()
	}()

	cc.log.Info("start collector")
	startFrom := cc.initStartFrom
	for ctx.Err() == nil {
		chEvent, err := SubscribeEventBlock(ctx, channelProvider, startFrom, cc.srcChName, nil)
		if err != nil {
			cc.log.Error(err, "subscribe to channel events")
			delayOrCancel(ctx, cc.delayAfterSrcError)
			continue
		}

		cc.m.FabricConnectionStatus().Add(1, metrics.Labels().Channel.Create(cc.srcChName))

		lastPushedBlockNum := cc.loopProxyByEvents(ctx, startFrom, chEvent.BlockEvent())
		if lastPushedBlockNum != nil {
			startFrom = *lastPushedBlockNum + 1
		}

		chEvent.UnsubscribeEventBlock()

		cc.m.FabricConnectionStatus().Add(-1, metrics.Labels().Channel.Create(cc.srcChName))

		delayOrCancel(ctx, cc.delayAfterSrcError)
	}
}

func (cc *chCollector) loopProxyByEvents(ctx context.Context, startedFrom uint64, events <-chan *fab.BlockEvent) *uint64 {
	var lastPushedBlockNum *uint64

	expectedBlockNum := startedFrom
	isFirstBlockGot := false
	blockParser := parser.NewParser(cc.log, cc.srcChName, cc.batchPrefix)

	for {
		select {
		case <-ctx.Done():
			return lastPushedBlockNum
		case ev, ok := <-events:
			if !ok {
				cc.log.Error("source events channel closed")
				return lastPushedBlockNum
			}

			if ev.Block == nil || ev.Block.Header == nil {
				cc.log.Error("got event block: invalid block!!!")
				return lastPushedBlockNum
			}

			blockNum := ev.Block.Header.Number
			cc.log.Debugf("got event block: %v", blockNum)

			// It is allowed to receive (startedFrom - 1) once when we start receiving blocks
			if !isFirstBlockGot && blockNum != startedFrom && startedFrom > 0 && blockNum == startedFrom-1 {
				cc.log.Infof("got event block: %v, expected: %v. Skip it because we subscribe with -1 gap", blockNum, expectedBlockNum)
				isFirstBlockGot = true
				continue
			}

			if blockNum < expectedBlockNum {
				cc.log.Errorf("got event block: %v, expected: %v. The block was already handled, skip it", blockNum, expectedBlockNum)
				continue
			}

			// It is not allowed, might skip unhandled blocks
			if blockNum > expectedBlockNum {
				cc.log.Errorf("got event block: %v, expected: %v. The block num is greater than expected", blockNum, expectedBlockNum)
				return lastPushedBlockNum
			}

			data, err := blockParser.ExtractData(ev.Block)
			if err != nil {
				cc.log.Errorf("got event block: %d, extract block data : %s", blockNum, err.Error())
				return lastPushedBlockNum
			}

			select {
			case <-ctx.Done():
				return lastPushedBlockNum
			case cc.stream <- data:
				lastPushedBlockNum = &ev.Block.Header.Number
				expectedBlockNum++
			}
		}
	}
}

func (cc *chCollector) Close() {
	if cc.cancel != nil {
		cc.cancel()
	}
	cc.wgLoop.Wait()
	close(cc.stream)
}

func (cc *chCollector) GetData() <-chan *model.BlockData {
	return cc.stream
}

func delayOrCancel(ctx context.Context, delay time.Duration) {
	select {
	case <-time.After(delay):
	case <-ctx.Done():
	}
}
