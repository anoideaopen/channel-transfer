package hlf

import (
	"context"
	"fmt"

	"github.com/anoideaopen/channel-transfer/pkg/logger"
	"github.com/anoideaopen/glog"
	"github.com/google/uuid"
	"github.com/hyperledger/fabric-sdk-go/pkg/common/options"
	contextApi "github.com/hyperledger/fabric-sdk-go/pkg/common/providers/context"
	"github.com/hyperledger/fabric-sdk-go/pkg/common/providers/fab"
	"github.com/hyperledger/fabric-sdk-go/pkg/fab/events/client"
	clientdispatcher "github.com/hyperledger/fabric-sdk-go/pkg/fab/events/client/dispatcher"
	"github.com/hyperledger/fabric-sdk-go/pkg/fab/events/client/peerresolver"
	"github.com/hyperledger/fabric-sdk-go/pkg/fab/events/deliverclient"
	"github.com/hyperledger/fabric-sdk-go/pkg/fab/events/deliverclient/seek"
	"github.com/hyperledger/fabric-sdk-go/pkg/fab/events/service"
	"github.com/hyperledger/fabric-sdk-go/pkg/fab/events/service/dispatcher"
	"github.com/pkg/errors"
)

// Events stores data about block subscription without gaps
type Events struct {
	registration    fab.Registration
	service         fab.EventService
	blockEvents     <-chan *fab.BlockEvent
	clientID        string
	log             glog.Logger
	channelProvider contextApi.ChannelProvider
}

// TargetPeersResolver for filter peers after discoveryService find them - choose only peer from array
type TargetPeersResolver struct {
	targetPeersMap map[string]bool
}

// NewTargetPeersResolver create and make map from array for faster find element
func NewTargetPeersResolver(targetPeers []string) TargetPeersResolver {
	targetPeersMap := make(map[string]bool, len(targetPeers))

	for _, targetPeer := range targetPeers {
		targetPeersMap[targetPeer] = true
	}
	return TargetPeersResolver{
		targetPeersMap: targetPeersMap,
	}
}

// Resolve only peer from map targetPeersMap
func (tpp TargetPeersResolver) Resolve(peers []fab.Peer) (fab.Peer, error) {
	for _, peer := range peers {
		if tpp.targetPeersMap[peer.URL()] {
			return peer, nil
		}
	}
	return nil, errors.Errorf("peer not found in target Peers %v", tpp.targetPeersMap)
}

// ShouldDisconnect always returns false (will not disconnect a connected peer)
func (TargetPeersResolver) ShouldDisconnect(_ []fab.Peer, _ fab.Peer) bool {
	return false
}

// SubscribeEventBlock - subscription to events without pass blocks
func SubscribeEventBlock(ctx context.Context, channelProvider contextApi.ChannelProvider, fromBlock uint64, channel string, targetPeers []string) (*Events, error) {
	log := glog.FromContext(ctx).With(logger.Labels{Component: logger.ComponentSubscribeEvent, Channel: channel}.Fields()...)

	channelContext, err := channelProvider()
	if err != nil {
		return nil, errors.WithMessage(err, "failed to create channel context")
	}

	if channelContext.ChannelService() == nil {
		return nil, errors.New("channel service not initialized")
	}

	if fromBlock > 0 {
		fromBlock--
	}

	clientID := fmt.Sprintf("%s-%s", channel, uuid.New().String())
	opts := []options.Opt{
		client.WithBlockEvents(),
		client.WithID(clientID),
		deliverclient.WithSeekType(seek.FromBlock),
		deliverclient.WithBlockNum(fromBlock),
		dispatcher.WithEventConsumerTimeout(0),
	}

	if len(targetPeers) > 0 {
		opts = append(opts, clientdispatcher.WithPeerResolver(func(service.Dispatcher, contextApi.Client, string, ...options.Opt) peerresolver.Resolver {
			return NewTargetPeersResolver(targetPeers)
		}))
	}

	eventService, err := channelContext.ChannelService().EventService(opts...)
	if err != nil {
		return nil, errors.WithMessage(err, "event service creation failed")
	}

	registration, bEvent, err := eventService.RegisterBlockEvent()
	if err != nil {
		return nil, errors.WithMessage(err, "register block event failed")
	}

	return &Events{
		registration:    registration,
		service:         eventService,
		blockEvents:     bEvent,
		clientID:        clientID,
		log:             log,
		channelProvider: channelProvider,
	}, nil
}

func (e *Events) UnsubscribeEventBlock() {
	/*
		skd doesn't use context
		to prevent case when dispatcher event channel is blocked by
		current channel re.blockEvents, we clean it until this channel will close.
		and don't block base program
	*/
	go func() {
		for range e.blockEvents {
		}
	}()

	e.service.Unregister(e.registration)

	e.deleteEventService()
}

func (e *Events) deleteEventService() {
	channelContext, err := e.channelProvider()
	if err != nil {
		e.log.Infof("sdkComps chProvider return error: %s", err)
		return
	}
	cs := channelContext.ChannelService()

	var opts []options.Opt
	opts = append(opts, client.WithBlockEvents())
	if e.clientID != "" {
		opts = append(opts, client.WithID(e.clientID))
	}
	err = cs.DeleteEventService(opts...)
	if err != nil {
		e.log.Infof("delete event service error: %s", err)
		return
	}
}

func (e *Events) BlockEvent() <-chan *fab.BlockEvent {
	return e.blockEvents
}
