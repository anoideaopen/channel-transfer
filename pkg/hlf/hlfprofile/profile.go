package hlfprofile

import (
	"os"

	"github.com/go-errors/errors"
	"gopkg.in/yaml.v2"
)

type HlfProfile struct {
	OrgName             string
	MspID               string
	CredentialStorePath string
	CryptoStorePath     string
}

type ConnProfile struct {
	Client struct {
		Org             string `yaml:"organization"`
		CredentialStore struct {
			Path        string `yaml:"path"`
			CryptoStore struct {
				Path string `yaml:"path"`
			} `yaml:"cryptoStore"`
		} `yaml:"credentialStore"`
	} `yaml:"client"`
	Organizations map[string]struct {
		Mspid string `yaml:"mspid"`
	} `yaml:"organizations"`
}

func ParseProfileFromFile(profilePath string) (*HlfProfile, error) {
	b, err := os.ReadFile(profilePath)
	if err != nil {
		return nil, errors.Errorf("error read connection profile path: %w", err)
	}

	return ParseProfileBytes(b)
}

func ParseProfileBytes(profileBytes []byte) (*HlfProfile, error) {
	cp := ConnProfile{}
	if err := yaml.Unmarshal(profileBytes, &cp); err != nil {
		return nil, errors.Errorf("error unmarshal connection profile: %w", err)
	}

	org, ok := cp.Organizations[cp.Client.Org]
	if !ok {
		return nil, errors.Errorf("cannot find mspid for org %s: %w", cp.Client.Org,
			errors.New("error unmarshal connection profile"))
	}

	res := &HlfProfile{
		OrgName:             cp.Client.Org,
		MspID:               org.Mspid,
		CredentialStorePath: cp.Client.CredentialStore.Path,
		CryptoStorePath:     cp.Client.CredentialStore.CryptoStore.Path,
	}

	return res, nil
}
