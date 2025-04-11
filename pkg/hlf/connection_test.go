package hlf

import (
	"encoding/base64"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewConnectionProfileWithPath(t *testing.T) {
	profile, err := NewConnectionProfile("path", "")
	require.NoError(t, err)

	require.Equal(t, "path", profile.Path)
	require.Nil(t, profile.Raw)
}

func TestNewConnectionProfileWithRaw(t *testing.T) {
	raw := base64.StdEncoding.EncodeToString([]byte("raw"))

	profile, err := NewConnectionProfile("path", raw)
	require.NoError(t, err)

	require.Equal(t, "raw", string(profile.Raw))
}

func TestNewConnectionProfileWithInvalidRaw(t *testing.T) {
	_, err := NewConnectionProfile("path", "invalid")
	require.Error(t, err)
}
