package hlf

import (
	"context"
	"time"

	"github.com/anoideaopen/channel-transfer/pkg/helpers/nerrors"
	"github.com/anoideaopen/channel-transfer/pkg/metrics"
	"github.com/anoideaopen/common-component/errorshlp"
	"github.com/avast/retry-go/v4"
	"github.com/hyperledger/fabric-sdk-go/pkg/client/channel"
	"github.com/hyperledger/fabric-sdk-go/pkg/client/ledger"
	hlfcontext "github.com/hyperledger/fabric-sdk-go/pkg/common/providers/context"
	"github.com/newity/glog"
	"github.com/pkg/errors"
)

type executor interface {
	invoke(ctx context.Context, req channel.Request, options []channel.RequestOption) (channel.Response, error)
	query(ctx context.Context, req channel.Request, options []channel.RequestOption) (channel.Response, error)
	blockchainHeight(options []ledger.RequestOption) (*uint64, error)
}

type ChExecutor struct {
	// args
	log    glog.Logger
	m      metrics.Metrics
	chName string

	// init
	executor             executor
	retryExecuteAttempts uint
	retryExecuteMaxDelay time.Duration
	retryExecuteDelay    time.Duration
}

func createChExecutor(
	ctx context.Context,
	chName string,
	channelProvider hlfcontext.ChannelProvider,
	execOpts ExecuteOptions,
) (*ChExecutor, error) {
	log := glog.FromContext(ctx)
	m := metrics.FromContext(ctx)

	chExec := &ChExecutor{
		log:                  log,
		m:                    m,
		chName:               chName,
		retryExecuteAttempts: execOpts.RetryExecuteAttempts,
		retryExecuteMaxDelay: execOpts.RetryExecuteMaxDelay,
		retryExecuteDelay:    execOpts.RetryExecuteDelay,
	}

	if err := chExec.initExecutor(channelProvider, execOpts); err != nil {
		return nil, errorshlp.WrapWithDetails(err, nerrors.ErrTypeHlf, nerrors.ComponentExecutor)
	}

	return chExec, nil
}

func (che *ChExecutor) initExecutor(chProvider hlfcontext.ChannelProvider, execOpts ExecuteOptions) error {
	var opts []channel.ClientOption

	chClient, err := channel.New(chProvider, opts...)
	if err != nil {
		return errors.Wrap(err, "create channel client instance")
	}

	chCtx, err := chProvider()
	if err != nil {
		return errors.Wrap(err, "create channel client context")
	}

	ldgCli, err := ledger.New(chProvider)
	if err != nil {
		return errors.Wrap(err, "create channel ledger instance")
	}

	che.executor = &hlfExecutor{
		chClient:  chClient,
		chCtx:     chCtx,
		execOpts:  execOpts,
		ldgClient: ldgCli,
	}

	return nil
}

func (che *ChExecutor) Invoke(ctx context.Context, req channel.Request, options []channel.RequestOption) (channel.Response, error) {
	return che.executeWithRetry(ctx, func() (channel.Response, error) {
		return che.executor.invoke(ctx, req, options)
	})
}

func (che *ChExecutor) Query(ctx context.Context, req channel.Request, options []channel.RequestOption) (channel.Response, error) {
	return che.executeWithRetry(ctx, func() (channel.Response, error) {
		return che.executor.query(ctx, req, options)
	})
}

func (che *ChExecutor) Close() {
	che.executor = nil
}

func (che *ChExecutor) executeWithRetry(ctx context.Context, f func() (channel.Response, error)) (channel.Response, error) {
	var resp channel.Response
	err := retry.Do(func() error {
		r, err := f()
		if err != nil {
			return err
		}
		resp = r
		return nil
	},
		retry.LastErrorOnly(true),
		retry.Attempts(che.retryExecuteAttempts),
		retry.Delay(che.retryExecuteDelay),
		retry.MaxDelay(che.retryExecuteMaxDelay),
		retry.RetryIf(isExecuteErrorRecoverable),
		retry.Context(ctx),
	)
	if err != nil {
		return resp, errorshlp.WrapWithDetails(errors.WithStack(err),
			nerrors.ErrTypeHlf, nerrors.ComponentExecutor)
	}

	return resp, nil
}

func isExecuteErrorRecoverable(e error) bool {
	return IsConnectionFailedErr(e) || IsEndorsementMismatchErr(e)
}

func (che *ChExecutor) BlockchainHeight(ctx context.Context) (*uint64, error) {
	return che.executor.blockchainHeight(
		[]ledger.RequestOption{
			ledger.WithParentContext(ctx),
		},
	)
}
