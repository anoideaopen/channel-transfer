package hlf

import (
	"github.com/anoideaopen/cartridge"
	"github.com/anoideaopen/cartridge/manager"
	"github.com/hyperledger/fabric-sdk-go/pkg/common/errors/status"
	hlfcontext "github.com/hyperledger/fabric-sdk-go/pkg/common/providers/context"
	"github.com/hyperledger/fabric-sdk-go/pkg/common/providers/core"
	"github.com/hyperledger/fabric-sdk-go/pkg/core/config"
	"github.com/hyperledger/fabric-sdk-go/pkg/fabsdk"
	"github.com/pkg/errors"
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
	cryptoManager manager.Manager,
) (*fabsdk.FabricSDK, error) {
	configBackends, err := config.FromFile(connectionProfile)()
	if err != nil {
		return nil, errors.Wrap(err, "reads sdk config")
	}

	var connectOpts []fabsdk.Option
	if cryptoManager != nil {
		connectOpts, err = cartridge.NewConnector(cryptoManager, cartridge.NewVaultConnectProvider(configBackends...)).Opts()
		if err != nil {
			return nil, errors.Wrap(err, "creates sdk connect options array")
		}
	}

	fabricSDK, err := fabsdk.New(configBackendsToProvider(configBackends), connectOpts...)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return fabricSDK, nil
}

func createChannelProvider(
	channelName string,
	userName string,
	orgName string,
	cryptoManager manager.Manager,
	fsdk *fabsdk.FabricSDK,
) hlfcontext.ChannelProvider {
	contextOpts := []fabsdk.ContextOption{fabsdk.WithOrg(orgName)}

	if cryptoManager != nil {
		contextOpts = append(contextOpts, fabsdk.WithIdentity(cryptoManager.SigningIdentity()))
	} else {
		contextOpts = append(contextOpts, fabsdk.WithUser(userName))
	}

	return fsdk.ChannelContext(channelName, contextOpts...)
}
