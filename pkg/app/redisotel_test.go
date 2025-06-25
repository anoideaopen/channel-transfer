package app

import (
	"testing"

	"github.com/redis/go-redis/extra/redisotel/v9"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

func TestRedisotel(t *testing.T) {
	rdb := redis.NewUniversalClient(&redis.UniversalOptions{})
	err := redisotel.InstrumentTracing(nil)
	require.Error(t, err)
	require.Equal(t, "redisotel: <nil> not supported", err.Error())
	require.NoError(t, redisotel.InstrumentTracing(rdb))
	err = redisotel.InstrumentMetrics(nil)
	require.Error(t, err)
	require.Equal(t, "redisotel: <nil> not supported", err.Error())
	require.NoError(t, redisotel.InstrumentMetrics(rdb))
}
