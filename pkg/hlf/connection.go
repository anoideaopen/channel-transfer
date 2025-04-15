package hlf

import (
	"os"

	"github.com/anoideaopen/channel-transfer/pkg/config"
	"github.com/anoideaopen/channel-transfer/pkg/hlf/hlfprofile"
	"github.com/go-errors/errors"
	"github.com/hyperledger/fabric-sdk-go/pkg/fabsdk"
)

const ConnectionYamlEnvName = config.EnvPrefix + "_HLF_CONNECTION_YAML"

type ConnectionProfile struct {
	Path string
	Raw  string
}

func NewConnectionProfile(path string) (ConnectionProfile, error) {
	if path != "" {
		return ConnectionProfile{
			Path: path,
			Raw:  "",
		}, nil
	}

	if raw, ok := os.LookupEnv(ConnectionYamlEnvName); ok {
		return ConnectionProfile{
			Path: "",
			Raw:  raw,
		}, nil
	}

	return ConnectionProfile{}, errors.Errorf("hlf connection profile not found")
}

func fabricSDKFromConnectionProfile(profile ConnectionProfile) (*fabsdk.FabricSDK, error) {
	if profile.Raw != "" {
		return createFabricSDKFromRaw([]byte(profile.Raw))
	}
	return createFabricSDKFromFile(profile.Path)
}

func hlfProfileFromConnectionProfile(profile ConnectionProfile) (hlfprofile.HlfProfile, error) {
	var (
		hlfProfile *hlfprofile.HlfProfile
		err        error
	)
	if profile.Raw != "" {
		hlfProfile, err = hlfprofile.ParseProfileBytes([]byte(profile.Raw))
		if err != nil {
			return hlfprofile.HlfProfile{}, errors.Errorf("failed parsing profile from raw: %w", err)
		}
		return *hlfProfile, nil
	}

	hlfProfile, err = hlfprofile.ParseProfileFromFile(profile.Path)
	if err != nil {
		return hlfprofile.HlfProfile{}, errors.Errorf("failed parsing profile from %s: %w", profile.Path, err)
	}

	return *hlfProfile, nil
}
