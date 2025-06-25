package tracing

import (
	"context"
	"fmt"

	"github.com/anoideaopen/channel-transfer/pkg/model"
	"github.com/anoideaopen/channel-transfer/proto"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

var FinishSpan = func(span trace.Span, err error) {
	if span == nil {
		return
	}
	if err == nil {
		span.SetStatus(codes.Ok, "")
	} else {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	}
	span.End()
}

type BasicTransferDataGetter interface {
	GetIdTransfer() string
	GetGenerals() *proto.GeneralParams
	GetChannelTo() string
}

type TransferDataGetter interface {
	GetToken() string
	GetAmount() string
	BasicTransferDataGetter
}

type ItemDataGetter interface {
	GetItems() []*proto.TransferItem
}

type MultiTransferDataGetter struct {
	ItemDataGetter
	BasicTransferDataGetter
}

func (m *MultiTransferDataGetter) GetToken() string {
	itemList := make([]string, 0, len(m.GetItems()))
	for _, item := range m.GetItems() {
		itemList = append(itemList, item.GetToken())
	}
	return fmt.Sprintf("%+v", itemList)
}

func (m *MultiTransferDataGetter) GetAmount() string {
	itemList := make([]string, 0, len(m.GetItems()))
	for _, item := range m.GetItems() {
		itemList = append(itemList, item.GetAmount())
	}
	return fmt.Sprintf("%+v", itemList)
}

// StartSpan creates a new span with the provided context and tracer.
// It is a client responsibility to call FinishSpan when the span is no longer needed.
func StartSpan(
	ctx context.Context,
	tracer trace.Tracer,
	spanName string,
	req TransferDataGetter,
	attributes ...attribute.KeyValue,
) (context.Context, trace.Span) {
	attributes = append(attributes,
		attribute.String("id", req.GetIdTransfer()),
		attribute.String("from", req.GetGenerals().GetChannel()),
		attribute.String("to", req.GetChannelTo()),
		attribute.String("token", req.GetToken()),
		attribute.String("amount", req.GetAmount()),
	)
	return tracer.Start(ctx, //nolint:spancheck
		spanName,
		trace.WithAttributes(
			attributes...,
		),
	)
}

func SetAttributes(
	span trace.Span,
	req TransferDataGetter,
) {
	span.SetAttributes(
		attribute.String("id", req.GetIdTransfer()),
		attribute.String("from", req.GetGenerals().GetChannel()),
		attribute.String("to", req.GetChannelTo()),
		attribute.String("token", req.GetToken()),
		attribute.String("amount", req.GetAmount()),
	)
}

var _ TransferDataGetter = (*TraceableRequest)(nil)

type TraceableRequest struct {
	*model.TransferRequest
}

func (t *TraceableRequest) GetIdTransfer() string {
	return string(t.Transfer)
}

func (t *TraceableRequest) GetGenerals() *proto.GeneralParams {
	return &proto.GeneralParams{
		RequestId:  string(t.Request),
		MethodName: t.Method,
		Chaincode:  t.Chaincode,
		Channel:    t.Channel,
		Nonce:      t.Nonce,
		PublicKey:  t.PublicKey,
		Sign:       t.Sign,
	}
}

func (t *TraceableRequest) GetChannelTo() string {
	return t.To
}

func (t *TraceableRequest) GetToken() string {
	if len(t.Items) > 0 {
		itemList := make([]string, 0, len(t.Items))
		for _, item := range t.Items {
			itemList = append(itemList, item.Token)
		}
		return fmt.Sprintf("%+v", itemList)
	}
	return t.Token
}

func (t *TraceableRequest) GetAmount() string {
	if len(t.Items) > 0 {
		itemList := make([]string, 0, len(t.Items))
		for _, item := range t.Items {
			itemList = append(itemList, item.Amount)
		}
		return fmt.Sprintf("%+v", itemList)
	}
	return t.Amount
}

func NewTraceableRequest(tr *model.TransferRequest) *TraceableRequest {
	return &TraceableRequest{
		TransferRequest: tr,
	}
}
