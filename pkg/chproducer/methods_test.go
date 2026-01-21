package chproducer

import (
	"testing"

	"github.com/anoideaopen/channel-transfer/pkg/model"
	"github.com/stretchr/testify/require"
)

func TestShouldSkipToBatchNotFoundStatus(t *testing.T) {
	// Must skip ToBatchNotFound (prevents double-spending when TransferTo exists)
	require.True(t, shouldSkipToBatchNotFoundStatus(model.ToBatchNotFound))

	// Must NOT skip for other statuses
	require.False(t, shouldSkipToBatchNotFoundStatus(model.InternalErrorTransferStatus))
	require.False(t, shouldSkipToBatchNotFoundStatus(model.CompletedTransferTo))
}
