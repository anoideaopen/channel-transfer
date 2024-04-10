package demultiplexer

import (
	"context"
	"sync"

	"github.com/anoideaopen/channel-transfer/pkg/helpers/nerrors"
	"github.com/anoideaopen/channel-transfer/pkg/logger"
	"github.com/anoideaopen/channel-transfer/pkg/model"
	"github.com/anoideaopen/common-component/errorshlp"
	"github.com/newity/glog"
	"github.com/pkg/errors"
)

type chKey string

type Demultiplexer struct {
	mutex sync.RWMutex

	incoming   <-chan model.TransferRequest
	channels   map[chKey]chan model.TransferRequest
	chanBuffer int
	log        glog.Logger
}

func NewDemultiplexer(ctx context.Context, in <-chan model.TransferRequest, bufferSize int) (*Demultiplexer, error) {
	if bufferSize == 0 {
		return nil, errorshlp.WrapWithDetails(
			errors.New("invalid buffer size for channels - must be great than 0"),
			nerrors.ErrTypeInternal,
			nerrors.ComponentDemultiplexer)
	}

	log := glog.FromContext(ctx)
	log = log.With(logger.Labels{Component: logger.ComponentDemultiplexer}.Fields()...)

	dm := &Demultiplexer{
		incoming:   in,
		chanBuffer: bufferSize,
		channels:   make(map[chKey]chan model.TransferRequest),
		log:        log,
	}

	return dm, nil
}

func (dm *Demultiplexer) Run(ctx context.Context) error {
	for ctx.Err() == nil {
		select {
		case <-ctx.Done():
			return errors.WithStack(ctx.Err())
		case item, ok := <-dm.incoming:
			if !ok {
				return errors.WithStack(errors.New("incoming channel closed"))
			}
			dm.output(item)
		}
	}

	return errors.WithStack(ctx.Err())
}

func (dm *Demultiplexer) output(data model.TransferRequest) {
	key := chKey(data.Channel)

	channel := dm.specifyChannel(key)
	if len(channel) >= dm.chanBuffer {
		// Походу подписчик завис или тормоз
		dm.unSubscribe(key)
		dm.log.Error(errorshlp.WrapWithDetails(errors.New("channel buffer overflow. flush it"), nerrors.ErrTypeAPI, nerrors.ComponentDemultiplexer))
		return
	}
	channel <- data
}

func (dm *Demultiplexer) specifyChannel(key chKey) chan model.TransferRequest {
	dm.mutex.RLock()
	channel, ok := dm.channels[key]
	dm.mutex.RUnlock()
	if ok {
		return channel
	}

	// subscribe
	channel = make(chan model.TransferRequest, dm.chanBuffer)

	dm.mutex.Lock()
	dm.channels[key] = channel
	dm.mutex.Unlock()

	dm.log.Infof("subscribe to request channel %s", string(key))

	return channel
}

func (dm *Demultiplexer) unSubscribe(key chKey) {
	dm.mutex.RLock()
	channel, ok := dm.channels[key]
	dm.mutex.RUnlock()
	if ok {
		close(channel)

		dm.mutex.Lock()
		delete(dm.channels, key)
		dm.mutex.Unlock()

		dm.log.Infof("unsubscribed of request channel %s", string(key))
	}
}

func (dm *Demultiplexer) Stream(chName string) <-chan model.TransferRequest {
	return dm.specifyChannel(chKey(chName))
}
