package hlf

import (
	"encoding/base64"

	"github.com/anoideaopen/channel-transfer/pkg/hlf/hlfprofile"
	"github.com/go-errors/errors"
	"github.com/hyperledger/fabric-sdk-go/pkg/fabsdk"
)

type ConnectionProfile struct {
	Path string
	Raw  []byte
}

func NewConnectionProfile(path string, base64Encoded string) (ConnectionProfile, error) {
	var raw []byte
	if base64Encoded != "" {
		var err error
		raw, err = base64.StdEncoding.DecodeString(base64Encoded)
		if err != nil {
			return ConnectionProfile{}, err
		}
	}
	return ConnectionProfile{
		Path: path,
		Raw:  raw,
	}, nil
}

func fabricSDKFromConnectionProfile(profile ConnectionProfile) (*fabsdk.FabricSDK, error) {
	if profile.Raw != nil {
		return createFabricSDKFromRaw(profile.Raw)
	}
	return createFabricSDKFromFile(profile.Path)
}

func hlfProfileFromConnectionProfile(profile ConnectionProfile) (hlfprofile.HlfProfile, error) {
	var (
		hlfProfile *hlfprofile.HlfProfile
		err        error
	)
	if profile.Raw != nil {
		hlfProfile, err = hlfprofile.ParseProfileBytes(profile.Raw)
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
