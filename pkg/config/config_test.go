package config

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

const testConfigName = "config_test.yaml"

func TestGetConfigSimple(t *testing.T) {
	t.Setenv(fmt.Sprintf("%s_CONFIG", EnvPrefix), testConfigName)

	c, err := getConfig()
	require.NoError(t, err)
	require.NotNil(t, c)

	require.Equal(t, "info", c.LogLevel)

	require.Len(t, c.Channels, 3)
}

func TestGetConfigOverrideEnv(t *testing.T) {
	t.Setenv(fmt.Sprintf("%s_LOGLEVEL", EnvPrefix), "myval")
	t.Setenv(fmt.Sprintf("%s_CONFIG", EnvPrefix), testConfigName)

	c, err := getConfig()
	require.NoError(t, err)
	require.NotNil(t, c)
	require.Equal(t, "myval", c.LogLevel)
}

func TestValidateConfig(t *testing.T) {
	t.Setenv(fmt.Sprintf("%s_CONFIG", EnvPrefix), testConfigName)

	c, err := getConfig()
	require.NoError(t, err)
	err = validateConfig(c)
	require.NoError(t, err)
}

func TestExecuteOptions(t *testing.T) {
	defExecuteTimeout := time.Duration(100)

	defOpts := Options{
		ExecuteTimeout: &defExecuteTimeout,
	}

	executeTimeout := time.Duration(10)

	// 1. full ExecOptions
	fullExecOptions := Options{
		ExecuteTimeout: &executeTimeout,
	}

	et, err := fullExecOptions.EffExecuteTimeout(defOpts)
	require.EqualValues(t, *fullExecOptions.ExecuteTimeout, et)
	require.NoError(t, err)

	// 2. empty ExecOptions
	emptyExecOptions := Options{}
	et, err = emptyExecOptions.EffExecuteTimeout(defOpts)
	require.EqualValues(t, *defOpts.ExecuteTimeout, et)
	require.NoError(t, err)

	// 3. check that we don't override default values occasionally
	require.EqualValues(t, defExecuteTimeout, *defOpts.ExecuteTimeout)
}
