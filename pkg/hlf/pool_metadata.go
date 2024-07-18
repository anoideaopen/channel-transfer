package hlf

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/anoideaopen/channel-transfer/pkg/model"
	"github.com/anoideaopen/glog"
	hlfcontext "github.com/hyperledger/fabric-sdk-go/pkg/common/providers/context"
)

const (
	startFromZero            = 0
	checkpointFrequencySaver = time.Duration(5) * time.Second
)

var (
	ErrExecutorAlreadyExists = errors.New("executor already exists")
	ErrExecutorUndefined     = errors.New("executor undefined")
	// ErrChannelUsed           = errors.New("channel already  used")
)

type channelKey string

type transactionID string

type biDirectFlow struct {
	executor        *ChExecutor
	hasCollector    bool
	transfers       map[transactionID]model.ID
	ready           chan struct{}
	once            sync.Once
	channelProvider hlfcontext.ChannelProvider
}

type streams struct {
	mtx  sync.RWMutex
	data map[channelKey]*biDirectFlow
	log  glog.Logger
}

func createStreams(ctx context.Context) *streams {
	return &streams{
		data: make(map[channelKey]*biDirectFlow),
		log:  glog.FromContext(ctx),
	}
}

func (s *streams) transferID(key channelKey, txID transactionID, id model.ID) bool {
	s.mtx.Lock()
	defer s.mtx.Unlock()
	if ps, ok := s.data[key]; ok {
		ps.transfers[txID] = id
		return true
	}
	return false
}

func (s *streams) transactionID(key channelKey, txID transactionID) (model.ID, bool) {
	s.mtx.RLock()
	defer s.mtx.RUnlock()
	if ps, ok := s.data[key]; ok {
		if transferID, ok := ps.transfers[txID]; ok {
			return transferID, true
		}
	}
	return "", false
}

func (s *streams) removeTransactionID(key channelKey, txID transactionID) {
	s.mtx.Lock()
	defer s.mtx.Unlock()
	if ps, ok := s.data[key]; ok {
		delete(ps.transfers, txID)
	}
}

func (s *streams) collector(key channelKey, inUsed bool) bool {
	s.mtx.Lock()
	ps, ok := s.data[key]
	if ok {
		ps.hasCollector = inUsed
	}
	s.mtx.Unlock()
	return ok
}

func (s *streams) ready(key channelKey) {
	s.mtx.RLock()
	if ps, ok := s.data[key]; ok {
		ps.once.Do(func() {
			close(ps.ready)
		})
	}
	s.mtx.RUnlock()
}

func (s *streams) done(key channelKey) (<-chan struct{}, bool) {
	s.mtx.RLock()
	defer s.mtx.RUnlock()
	if ps, ok := s.data[key]; ok {
		return ps.ready, true
	}
	return nil, false
}

func (s *streams) exists(key channelKey) bool {
	s.mtx.RLock()
	_, ok := s.data[key]
	s.mtx.RUnlock()
	return ok
}

func (s *streams) store(key channelKey, executor *ChExecutor, channelProvider hlfcontext.ChannelProvider) {
	if s.exists(key) {
		return
	}

	s.mtx.Lock()
	defer s.mtx.Unlock()
	_, ok := s.data[key]
	if ok {
		return
	}
	s.data[key] = &biDirectFlow{
		channelProvider: channelProvider,
		executor:        executor,
		transfers:       make(map[transactionID]model.ID),
		ready:           make(chan struct{}),
	}
}

func (s *streams) loop(f func(key channelKey, val *biDirectFlow)) {
	s.mtx.Lock()
	for k, v := range s.data {
		f(k, v)
	}
	s.mtx.Unlock()
}

func (s *streams) load(key channelKey) (ps *biDirectFlow, ok bool) {
	s.mtx.RLock()
	ps, ok = s.data[key]
	s.mtx.RUnlock()
	return
}

func (s *streams) close() {
	s.mtx.Lock()
	for key := range s.data {
		if s.data[key].executor != nil {
			s.data[key].executor.Close()
		}
		for id := range s.data[key].transfers {
			delete(s.data[key].transfers, id)
		}
		delete(s.data, key)
	}
	s.mtx.Unlock()
}
