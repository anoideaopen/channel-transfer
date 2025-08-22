package telemetry

import (
	"testing"

	"github.com/anoideaopen/channel-transfer/pkg/config"
	"github.com/stretchr/testify/require"
)

const (
	serviceName    = "channel-transfer"
	tlsCACertWrong = "asdasd"
	tlsCACertValid = "LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSUNKekNDQWMyZ0F3SUJBZ0lVR2NSTDZkZ1d0UG4zK3VSRXpvcENiaEpwVU53d0NnWUlLb1pJemowRUF3SXcKY0RFTE1Ba0dBMVVFQmhNQ1ZWTXhGekFWQmdOVkJBZ1REazV2Y25Sb0lFTmhjbTlzYVc1aE1ROHdEUVlEVlFRSApFd1pFZFhKb1lXMHhHVEFYQmdOVkJBb1RFRzl5WnpFdVpYaGhiWEJzWlM1amIyMHhIREFhQmdOVkJBTVRFMk5oCkxtOXlaekV1WlhoaGJYQnNaUzVqYjIwd0hoY05NalV3TXpFd01UZ3lOekF3V2hjTk5EQXdNekEyTVRneU56QXcKV2pCd01Rc3dDUVlEVlFRR0V3SlZVekVYTUJVR0ExVUVDQk1PVG05eWRHZ2dRMkZ5YjJ4cGJtRXhEekFOQmdOVgpCQWNUQmtSMWNtaGhiVEVaTUJjR0ExVUVDaE1RYjNKbk1TNWxlR0Z0Y0d4bExtTnZiVEVjTUJvR0ExVUVBeE1UClkyRXViM0puTVM1bGVHRnRjR3hsTG1OdmJUQlpNQk1HQnlxR1NNNDlBZ0VHQ0NxR1NNNDlBd0VIQTBJQUJJQ2sKazFDRDhGdHZZajNJU1JENVJ3NEQ4SkxscEwrN3dmUW4reHRrTm5lSW4xQU1WWXNQalZTYkN0MHRUSStqc0V5VQptcDRyWVg4SVRlZ0FySnRuRlYralJUQkRNQTRHQTFVZER3RUIvd1FFQXdJQkJqQVNCZ05WSFJNQkFmOEVDREFHCkFRSC9BZ0VCTUIwR0ExVWREZ1FXQkJUZnhDeWJ3U3pFcVZEa0tvZzdMdXRyZFc1ODNqQUtCZ2dxaGtqT1BRUUQKQWdOSUFEQkZBaUVBNCtYeUJWRVFUVnhCRytJTks0bmFPbG4wSURSOWViMXdyUTJMVFMxUDZTRUNJRXhJdkd5SQpqeEZ0UkRNRTNnTGkyS1dOa1pETU1RbTdNdytnVHBlNVZIMXoKLS0tLS1FTkQgQ0VSVElGSUNBVEUtLS0tLQo="
)

func TestInit(t *testing.T) {
	err := InitTracing(t.Context(), nil, serviceName)
	require.Error(t, err)
	require.Equal(t, err.Error(), "tracing collector configuration is nil")

	require.NoError(t, InitTracing(t.Context(), &config.Collector{}, serviceName))

	err = InitTracing(
		t.Context(),
		&config.Collector{
			TLSCA: tlsCACertWrong,
		}, serviceName)
	require.Error(t, err)
	require.Contains(t, err.Error(), "nitialize trace exporter: failed to get TLS config")

	err = InitTracing(
		t.Context(),
		&config.Collector{
			AuthorizationHeaderKey:   "Authorization",
			AuthorizationHeaderValue: "Bearer token",
		}, serviceName)
	require.Error(t, err)
	require.Contains(t, err.Error(), "TLSCA must be set if authorization headers are provided")

	err = InitTracing(
		t.Context(),
		&config.Collector{
			AuthorizationHeaderKey:   "Authorization",
			AuthorizationHeaderValue: "Bearer token",
			TLSCA:                    tlsCACertValid,
		}, serviceName)
	require.NoError(t, err)
}
