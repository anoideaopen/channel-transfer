package hlf

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewConnectionProfileWithPath(t *testing.T) {
	profile, err := NewConnectionProfile("path")
	require.NoError(t, err)

	require.Equal(t, "path", profile.Path)
	require.Empty(t, profile.Raw)
}

func TestNewConnectionProfileWithRaw(t *testing.T) {
	testHlfConnectionYaml := "connection.yaml"
	t.Setenv(fmt.Sprintf(ConnectionYamlEnvName), testHlfConnectionYaml)

	profile, err := NewConnectionProfile("")
	require.NoError(t, err)

	require.Equal(t, testHlfConnectionYaml, profile.Raw)
}

func TestNewConnectionProfileWithInvalidRaw(t *testing.T) {
	_, err := NewConnectionProfile("")
	require.Error(t, err)
	require.Contains(t, err.Error(), "hlf connection profile not found")
}
