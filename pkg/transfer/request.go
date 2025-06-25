package transfer

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/anoideaopen/channel-transfer/pkg/data"
	"github.com/anoideaopen/channel-transfer/pkg/data/redis"
	"github.com/anoideaopen/channel-transfer/pkg/model"
	"github.com/anoideaopen/channel-transfer/pkg/tracing"
	"github.com/anoideaopen/channel-transfer/proto"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

type Request struct {
	storage *redis.Storage
}

func NewRequest(storage *redis.Storage) *Request {
	return &Request{
		storage: storage,
	}
}

func (r *Request) TransferKeep(ctx context.Context, transferRequest model.TransferRequest) error {
	var err error
	tracebleRequest := tracing.NewTraceableRequest(&transferRequest)
	ctx, span := tracing.StartSpan(
		ctx,
		tracer,
		"transfer: TransferKeep",
		tracebleRequest,
		attribute.String("method", string(transferRequest.Method)),
	)
	defer func() {
		tracing.FinishSpan(span, err)
	}()
	if err = r.storage.Save(ctx, &transferRequest, data.Key(transferRequest.Transfer)); err != nil {
		return fmt.Errorf("save request : %w", err)
	}

	return nil
}

func (r *Request) TransferFetch(ctx context.Context, transfer model.ID) (model.TransferRequest, error) {
	var err error
	ctx, span := tracer.Start(
		ctx,
		"transfer: TransferFetch",
		trace.WithAttributes(
			attribute.String("id", string(transfer)),
		),
	)
	defer func() {
		tracing.FinishSpan(span, err)
	}()
	transferRequest := model.TransferRequest{}
	if err = r.storage.Load(ctx, &transferRequest, data.Key(transfer)); err != nil {
		return transferRequest, fmt.Errorf("load request : %w", err)
	}
	tracing.SetAttributes(span, tracing.NewTraceableRequest(&transferRequest))
	return transferRequest, nil
}

func (r *Request) TransferModify(ctx context.Context, transferRequest model.TransferRequest, ttl time.Duration) error {
	var err error
	tracebleRequest := tracing.NewTraceableRequest(&transferRequest)
	ctx, span := tracing.StartSpan(
		ctx,
		tracer,
		"transfer: TransferModify",
		tracebleRequest,
	)
	defer func() {
		tracing.FinishSpan(span, err)
	}()

	request := model.TransferRequest{}
	if err := r.storage.Load(ctx, &request, data.Key(transferRequest.Transfer)); err != nil {
		if !strings.Contains(err.Error(), data.ErrObjectNotFound.Error()) {
			return fmt.Errorf("load request : %w", err)
		}
		if err = r.storage.SaveConsideringTTL(ctx, &transferRequest, data.Key(transferRequest.Transfer), ttl); err != nil {
			return fmt.Errorf("save request : %w", err)
		}
		return nil
	}

	if transferRequest.Request != "" && request.Request != transferRequest.Request {
		request.Request = transferRequest.Request
	}
	if transferRequest.User != "" && request.User != transferRequest.User {
		request.User = transferRequest.User
	}
	if transferRequest.Method != "" && request.Method != transferRequest.Method {
		request.Method = transferRequest.Method
	}
	if transferRequest.Chaincode != "" && request.Chaincode != transferRequest.Chaincode {
		request.Chaincode = transferRequest.Chaincode
	}
	if transferRequest.Channel != "" && request.Channel != transferRequest.Channel {
		request.Channel = transferRequest.Channel
	}
	if transferRequest.Nonce != "" && request.Nonce != transferRequest.Nonce {
		request.Nonce = transferRequest.Nonce
	}
	if transferRequest.PublicKey != "" && request.PublicKey != transferRequest.PublicKey {
		request.PublicKey = transferRequest.PublicKey
	}
	if transferRequest.Sign != "" && request.Sign != transferRequest.Sign {
		request.Sign = transferRequest.Sign
	}
	if transferRequest.To != "" && request.To != transferRequest.To {
		request.To = transferRequest.To
	}
	if transferRequest.Token != "" && request.Token != transferRequest.Token {
		request.Token = transferRequest.Token
	}
	if transferRequest.Amount != "" && request.Amount != transferRequest.Amount {
		request.Amount = transferRequest.Amount
	}
	if len(transferRequest.Items) > 0 {
		request.Items = transferRequest.Items
	}
	if r.canBeChangedStatus(request.Status, transferRequest.Status) {
		request.TransferResult = transferRequest.TransferResult
	}

	if err := r.storage.SaveConsideringTTL(ctx, &request, data.Key(transferRequest.Transfer), ttl); err != nil {
		return fmt.Errorf("save request : %w", err)
	}
	return nil
}

func (r *Request) Registry(ctx context.Context) ([]*model.TransferRequest, error) {
	ctx, span := tracer.Start(
		ctx,
		"transfer: Registry",
	)
	defer func() {
		tracing.FinishSpan(span, nil)
	}()
	return data.ToSlice[model.TransferRequest](r.storage.Search(ctx, &model.TransferRequest{}, ""))
}

func (r *Request) TransferResultModify(ctx context.Context, transferID model.ID, result model.TransferResult) error {
	var err error
	ctx, span := tracer.Start(
		ctx,
		"transfer: TransferResultModify", //request: TransferFetch
		trace.WithAttributes(
			attribute.String("id", string(transferID)),
			attribute.String("new.status", result.Status),
			attribute.String("new.message", result.Message),
		),
	)
	defer func() {
		tracing.FinishSpan(span, err)
	}()
	request := model.TransferRequest{}
	if err := r.storage.Load(ctx, &request, data.Key(transferID)); err != nil {
		if !strings.Contains(err.Error(), data.ErrObjectNotFound.Error()) {
			return fmt.Errorf("load request : %w", err)
		}
		request = model.TransferRequest{Transfer: transferID, TransferResult: result}
		if err = r.storage.Save(ctx, &request, data.Key(transferID)); err != nil {
			return fmt.Errorf("save request : %w", err)
		}
	}
	tracing.SetAttributes(span, tracing.NewTraceableRequest(&request))

	if !r.canBeChangedStatus(request.Status, result.Status) {
		return nil
	}

	request.TransferResult = result

	if err := r.storage.SaveConsideringTTL(ctx, &request, data.Key(request.Transfer), redis.TTLNotTakenInto); err != nil {
		return fmt.Errorf("save request : %w", err)
	}
	return nil
}

func (r *Request) IsChangeableStatus(status string) bool {
	if status == "" ||
		status == proto.TransferStatusResponse_STATUS_IN_PROCESS.String() ||
		status == proto.TransferStatusResponse_STATUS_UNDEFINED.String() {
		return true
	}
	return false
}

func (r *Request) canBeChangedStatus(src, dst string) bool {
	if !r.IsChangeableStatus(src) {
		return false
	}
	if src == dst {
		return false
	}
	if src == proto.TransferStatusResponse_STATUS_IN_PROCESS.String() &&
		(dst == proto.TransferStatusResponse_STATUS_UNDEFINED.String() || dst == "") {
		return false
	}
	return true
}
