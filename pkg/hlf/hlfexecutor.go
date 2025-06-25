package hlf

import (
	"context"
	"fmt"
	"time"

	"github.com/anoideaopen/channel-transfer/pkg/telemetry"
	"github.com/anoideaopen/channel-transfer/pkg/tracing"
	"github.com/go-errors/errors"
	pb "github.com/hyperledger/fabric-protos-go-apiv2/peer"
	"github.com/hyperledger/fabric-sdk-go/pkg/client/channel"
	"github.com/hyperledger/fabric-sdk-go/pkg/client/channel/invoke"
	"github.com/hyperledger/fabric-sdk-go/pkg/client/common/filter"
	"github.com/hyperledger/fabric-sdk-go/pkg/client/ledger"
	"github.com/hyperledger/fabric-sdk-go/pkg/common/errors/status"
	chctx "github.com/hyperledger/fabric-sdk-go/pkg/common/providers/context"
	"github.com/hyperledger/fabric-sdk-go/pkg/common/providers/fab"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

var tracer = otel.Tracer("pkg/hlf:hlfexecutor")

type ExecuteOptions struct {
	ExecuteTimeout       time.Duration
	RetryExecuteAttempts uint
	RetryExecuteMaxDelay time.Duration
	RetryExecuteDelay    time.Duration
}

type hlfExecutor struct {
	chClient  *channel.Client
	chCtx     chctx.Channel
	execOpts  ExecuteOptions
	ldgClient *ledger.Client
}

func (he *hlfExecutor) invoke(ctx context.Context, request channel.Request, options []channel.RequestOption) (channel.Response, error) {

	var (
		err  error
		resp channel.Response
	)

	options = append(options, channel.WithParentContext(ctx))

	var argsListForTracing string
	for _, arg := range request.Args {
		argsListForTracing += string(arg) + ", "
	}

	ctx, span := tracer.Start(ctx,
		"hlfexecutor: invoke",
		trace.WithAttributes(
			attribute.String("invoke.args", fmt.Sprintf("%+v\n", argsListForTracing)),
		),
	)
	defer func() {
		tracing.FinishSpan(span, err)
	}()

	executeTimeout := he.chCtx.EndpointConfig().Timeout(fab.Execute)
	if he.execOpts.ExecuteTimeout > 0 {
		executeTimeout = he.execOpts.ExecuteTimeout
	}
	options = append(options, channel.WithTimeout(fab.Execute, executeTimeout))

	// use all channel peers from connection profile in the peers selection algorithm
	options = append(options,
		channel.WithTargetFilter(
			filter.NewEndpointFilter(
				he.chCtx, filter.EndorsingPeer)))

	h := invoke.NewSelectAndEndorseHandler(
		invoke.NewEndorsementValidationHandler(
			invoke.NewSignatureValidationHandler(
				&commitTxHandler{
					ctx:      ctx,
					execOpts: he.execOpts,
				},
			),
		),
	)

	request.TransientMap = telemetry.TransientMapFromContext(ctx)
	resp, err = he.chClient.InvokeHandler(h, request, options...)
	if err != nil {
		err = errors.Errorf("invoke failed: %w", err)
	}

	return resp, err
}

func (he *hlfExecutor) query(ctx context.Context, request channel.Request, options []channel.RequestOption) (channel.Response, error) {

	var (
		err  error
		resp channel.Response
	)

	options = append(options, channel.WithParentContext(ctx))

	var argsListForTracing string
	for _, arg := range request.Args {
		argsListForTracing += string(arg) + ", "
	}

	ctx, span := tracer.Start(ctx,
		"hlfexecutor: query",
		trace.WithAttributes(
			attribute.String("query.args", fmt.Sprintf("%+v\n", argsListForTracing)),
		),
	)
	defer func() {
		tracing.FinishSpan(span, err)
	}()

	executeTimeout := he.chCtx.EndpointConfig().Timeout(fab.Execute)
	if he.execOpts.ExecuteTimeout > 0 {
		executeTimeout = he.execOpts.ExecuteTimeout
	}
	options = append(options, channel.WithTimeout(fab.Execute, executeTimeout))

	// use all channel peers from connection profile in the peers selection algorithm
	options = append(options,
		channel.WithTargetFilter(
			filter.NewEndpointFilter(
				he.chCtx, filter.EndorsingPeer)))

	request.TransientMap = telemetry.TransientMapFromContext(ctx)

	resp, err = he.chClient.Query(request, options...)
	if err != nil {
		err = errors.Errorf("query failed: %w", err)
	}

	return resp, err
}

// CommitTxHandler for committing transactions
type commitTxHandler struct {
	ctx      context.Context
	execOpts ExecuteOptions
}

// Handle handles commit tx
func (cth *commitTxHandler) Handle(reqCtx *invoke.RequestContext, clientCtx *invoke.ClientContext) {
	// register tx event
	reg, statusNotifier, err := clientCtx.
		EventService.RegisterTxStatusEvent(
		string(reqCtx.Response.TransactionID))
	if err != nil {
		reqCtx.Error = errors.Errorf("error registering for TxStatus event: %w", err)
		return
	}
	defer clientCtx.EventService.Unregister(reg)

	tx, err := clientCtx.Transactor.CreateTransaction(
		fab.TransactionRequest{
			Proposal:          reqCtx.Response.Proposal,
			ProposalResponses: reqCtx.Response.Responses,
		})
	if err != nil {
		reqCtx.Error = errors.Errorf("createTransaction failed: %w", err)
		return
	}

	if _, err = clientCtx.Transactor.SendTransaction(tx); err != nil {
		reqCtx.Error = errors.Errorf("sendTransaction failed: %w", err)
		return
	}

	select {
	case txStatus := <-statusNotifier:
		reqCtx.Response.BlockNumber = txStatus.BlockNumber
		reqCtx.Response.TxValidationCode = txStatus.TxValidationCode

		if txStatus.TxValidationCode != pb.TxValidationCode_VALID {
			reqCtx.Error = errors.New(
				status.New(status.EventServerStatus, int32(txStatus.TxValidationCode),
					"received invalid transaction", nil))
		}
		return
	case <-cth.ctx.Done():
		reqCtx.Error = errors.New(
			status.New(status.ClientStatus, status.Unknown.ToInt32(),
				"Execute didn't receive block event (context done)", nil))
		return
	case <-reqCtx.Ctx.Done():
		reqCtx.Error = errors.New(
			status.New(status.ClientStatus, status.Timeout.ToInt32(),
				"Execute didn't receive block event", nil))
		return
	}
}

func (he *hlfExecutor) blockchainHeight(options []ledger.RequestOption) (*uint64, error) {
	bi, err := he.ldgClient.QueryInfo(options...)
	if err != nil {
		return nil, err
	}

	return &bi.BCI.Height, nil
}
