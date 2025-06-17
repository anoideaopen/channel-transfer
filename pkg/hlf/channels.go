package hlf

import (
	"fmt"

	"github.com/anoideaopen/channel-transfer/pkg/config"
	"google.golang.org/grpc"
)

// channelsInfoProvider stores and provides data about channels and
// configuration of connection to them
type channelsInfoProvider struct {
	servedChannels       []string                    // list of channel names the channel transfer serves
	channelsParams       map[string]config.Channel   // channel connection info mapped to channel names
	gRPCClientsByAddress map[string]*grpc.ClientConn // grpc clients mapped to grpc addresses
	gRPCClientByChannels map[string]*grpc.ClientConn // grpc clients mapped to channel names
}

func newChannelsInfoProvider(allChannels []config.Channel, servedChannels []string) (*channelsInfoProvider, error) {
	info := &channelsInfoProvider{
		servedChannels:       servedChannels,
		channelsParams:       make(map[string]config.Channel),
		gRPCClientsByAddress: make(map[string]*grpc.ClientConn),
		gRPCClientByChannels: make(map[string]*grpc.ClientConn),
	}

	noServedChannels := len(info.servedChannels) == 0

	for _, param := range allChannels {
		info.channelsParams[param.Name] = param
		if noServedChannels {
			// if no served channels provided, serve all channels from the full list
			info.servedChannels = append(info.servedChannels, param.Name)
		}
	}

	// check if the list of served channels contains channels which are not in the full list
	for _, channel := range info.servedChannels {
		_, ok := info.channelsParams[channel]
		if !ok {
			return nil, fmt.Errorf("no configuration for channel %s", channel)
		}
	}

	return info, nil
}

func (info *channelsInfoProvider) getGRPCClientByChannelName(channel string) (*grpc.ClientConn, error) {
	client, ok := info.gRPCClientByChannels[channel]
	if ok {
		return client, nil
	}

	var (
		gRPCClient *grpc.ClientConn
		err        error
	)
	chParams := info.channelsParams[channel]

	if chParams.TaskExecutor != nil {
		// check if a client for the gRPC address already created
		gRPCClient, ok = info.gRPCClientsByAddress[chParams.TaskExecutor.AddressGRPC]
		if !ok {
			// if not, create the new one
			gRPCClient, err = createGRPCClient(chParams.TaskExecutor)
			if err != nil {
				return nil, fmt.Errorf("create gRPC client: %w", err)
			}
			info.gRPCClientsByAddress[chParams.TaskExecutor.AddressGRPC] = gRPCClient
		}
		info.gRPCClientByChannels[channel] = gRPCClient
	}

	return gRPCClient, nil
}

func (info *channelsInfoProvider) ServedChannels() []string {
	return info.servedChannels
}
