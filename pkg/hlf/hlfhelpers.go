package hlf

import (
	"github.com/go-errors/errors"
	"github.com/hyperledger/fabric-sdk-go/pkg/common/errors/status"
	hlfcontext "github.com/hyperledger/fabric-sdk-go/pkg/common/providers/context"
	"github.com/hyperledger/fabric-sdk-go/pkg/common/providers/core"
	"github.com/hyperledger/fabric-sdk-go/pkg/core/config"
	"github.com/hyperledger/fabric-sdk-go/pkg/fabsdk"
)

func configBackendsToProvider(cb []core.ConfigBackend) func() ([]core.ConfigBackend, error) {
	return func() ([]core.ConfigBackend, error) {
		return cb, nil
	}
}

// IsEndorsementMismatchErr checks if the error is a mismatch.
func IsEndorsementMismatchErr(err error) bool {
	if err == nil {
		return false
	}
	s, ok := status.FromError(err)
	return ok && status.Code(s.Code) == status.EndorsementMismatch
}

// IsConnectionFailedErr checks if the error is a network connection attempt from the SDK fails
func IsConnectionFailedErr(err error) bool {
	if err == nil {
		return false
	}
	s, ok := status.FromError(err)
	return ok && status.Code(s.Code) == status.ConnectionFailed
}

func createFabricSDK(
	connectionProfile string,
) (*fabsdk.FabricSDK, error) {
	configBackends, err := config.FromFile(connectionProfile)()
	if err != nil {
		return nil, errors.Errorf("reads sdk config: %w", err)
	}

	fabricSDK, err := fabsdk.New(configBackendsToProvider(configBackends))
	if err != nil {
		return nil, errors.New(err)
	}

	return fabricSDK, nil
}

func createChannelProvider(
	channelName string,
	userName string,
	orgName string,
	fsdk *fabsdk.FabricSDK,
) hlfcontext.ChannelProvider {
	contextOpts := []fabsdk.ContextOption{fabsdk.WithOrg(orgName)}
	contextOpts = append(contextOpts, fabsdk.WithUser(userName))

	return fsdk.ChannelContext(channelName, contextOpts...)
}
