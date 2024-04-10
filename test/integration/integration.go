package test

import (
	"context"
	"os"

	client "github.com/anoideaopen/channel-transfer/test/integration/clienthttp"
	"google.golang.org/grpc/metadata"

	"github.com/go-openapi/runtime"
	httptransport "github.com/go-openapi/runtime/client"
	"github.com/go-openapi/strfmt"
)

var (
	transferCli *client.CrossChanelTransfer
	clientCtx   context.Context
	authOpts    func(c *runtime.ClientOperation)
	targetGrpc  string
	token       string
	ctx         context.Context
)

func init() {
	ctx = context.Background()

	transport := httptransport.New(os.Getenv("CHANNEL_TRANSFER_HTTP"), "", nil)
	transferCli = client.New(transport, strfmt.Default)

	token = os.Getenv("HLF_PROXY_AUTH_TOKEN")
	clientCtx = metadata.NewOutgoingContext(ctx, metadata.Pairs("authorization", token))

	auth := httptransport.APIKeyAuth("authorization", "header", token)
	authOpts = func(c *runtime.ClientOperation) {
		c.AuthInfo = auth
	}

	targetGrpc = os.Getenv("CHANNEL_TRANSFER_GRPC")
}
