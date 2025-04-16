package hlfprofile

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseProfileFromFile(t *testing.T) {
	hlfProfile, err := ParseProfileFromFile("connection.yaml")
	require.Nil(t, err)
	require.NotNil(t, hlfProfile)

	require.EqualValues(t, "anoideaopen", hlfProfile.OrgName)
	require.EqualValues(t, "anoideaopenMSP", hlfProfile.MspID)
	require.EqualValues(t, "dev-data/hlf-test-stage-04/crypto/backend@anoideaopen.anoideaopen-test-04.ledger.scientificideas.org/msp/signcerts", hlfProfile.CredentialStorePath)
	require.EqualValues(t, "dev-data/hlf-test-stage-04/crypto/backend@anoideaopen.anoideaopen-test-04.ledger.scientificideas.org/msp", hlfProfile.CryptoStorePath)
}

func TestParseProfileFromBase64(t *testing.T) {
	raw, err := os.ReadFile("connection.yaml")
	require.Nil(t, err)

	hlfProfile, err := ParseProfileBytes(raw)
	require.Nil(t, err)
	require.NotNil(t, hlfProfile)

	require.EqualValues(t, "anoideaopen", hlfProfile.OrgName)
	require.EqualValues(t, "anoideaopenMSP", hlfProfile.MspID)
	require.EqualValues(t, "dev-data/hlf-test-stage-04/crypto/backend@anoideaopen.anoideaopen-test-04.ledger.scientificideas.org/msp/signcerts", hlfProfile.CredentialStorePath)
	require.EqualValues(t, "dev-data/hlf-test-stage-04/crypto/backend@anoideaopen.anoideaopen-test-04.ledger.scientificideas.org/msp", hlfProfile.CryptoStorePath)
}
