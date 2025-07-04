//nolint:spancheck
package transfer

import (
	"context"
	"fmt"
	"strings"

	"github.com/anoideaopen/channel-transfer/pkg/data"
	"github.com/anoideaopen/channel-transfer/pkg/model"
	"github.com/anoideaopen/channel-transfer/pkg/telemetry"
	"github.com/anoideaopen/channel-transfer/pkg/tracing"
	dto "github.com/anoideaopen/channel-transfer/proto"
	"github.com/anoideaopen/glog"
	"github.com/go-errors/errors"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const StatusOptionFilterName = "excludeStatus"

// Errors related to request processing.
var (
	ErrBadRequest        = errors.New("bad request")
	ErrInvalidStatusCode = errors.New("invalid status code")
	ErrBadChannel        = errors.New("channel not in configuration list")
	ErrPubKey            = errors.New("public key of the signer of the request undefined")
	ErrNonce             = errors.New("nonce undefined")
	ErrChaincode         = errors.New("chaincode undefined")
	ErrMethod            = errors.New("method name undefined")
	ErrUnknownMethod     = errors.New("unknown method name")
	ErrSign              = errors.New("sign undefined")
	tracer               = otel.Tracer("pkg/transfer")
)

// APIServer implements the logic for processing user requests and serves as
// the application server layer for working with the business logic of the app.
type APIServer struct {
	dto.UnimplementedAPIServer
	ctrl           RequestController
	output         chan<- model.TransferRequest
	actualChannels map[string]struct{}
	log            glog.Logger
}

// NewAPIServer creates a new instance of the structure with the specified
// controller.
func NewAPIServer(
	ctx context.Context,
	output chan<- model.TransferRequest,
	ctrl RequestController,
	actualChannels []string,
) *APIServer {
	server := &APIServer{
		ctrl:           ctrl,
		output:         output,
		actualChannels: make(map[string]struct{}),
		log:            glog.FromContext(ctx),
	}
	for _, channel := range actualChannels {
		server.actualChannels[channel] = struct{}{}
	}
	return server
}

// TransferByCustomer registers a new transfer from one channel to another on
// behalf of the contract administrator, using business logic and maps the
// result to DTO request objects.
func (api *APIServer) TransferByCustomer(
	ctx context.Context,
	req *dto.TransferBeginCustomerRequest,
) (*dto.TransferStatusResponse, error) {
	var (
		err error
		tr  model.TransferRequest
	)

	if req == nil {
		return nil, ErrBadRequest
	}

	log := glog.FromContext(ctx)
	log.Set(
		glog.Field{K: "transfer.id", V: req.GetIdTransfer()},
		glog.Field{K: "transfer.from", V: req.GetGenerals().GetChannel()},
		glog.Field{K: "transfer.to", V: req.GetChannelTo()},
		glog.Field{K: "transfer.token", V: req.GetToken()},
	)
	md := telemetry.TransferMetadataFromContext(ctx)
	ctx, span := tracing.StartSpan(
		ctx,
		tracer,
		"api_server: TransferByCustomer",
		req,
	)
	defer func() {
		tracing.FinishSpan(span, err)
	}()
	log.Debug("transferByCustomer request received")
	tr, err = dtoBeginCustomerToModelTransferRequest(req, api.actualChannels)
	if err != nil {
		err = errors.Errorf("parse transfer request: %w", err)
		return &dto.TransferStatusResponse{
				IdTransfer: req.GetIdTransfer(),
				Status:     dto.TransferStatusResponse_STATUS_ERROR,
				Message:    err.Error(),
			}, status.Error(
				codes.InvalidArgument,
				err.Error(),
			)
	}
	tr.Metadata = md

	if err = api.ctrl.TransferKeep(ctx, tr); err != nil {
		return nil, fmt.Errorf(
			"[APIServer] failed to save transfer request: %w",
			err,
		)
	}

	api.output <- tr

	return &dto.TransferStatusResponse{
		IdTransfer: string(tr.Transfer),
		Status:     dto.TransferStatusResponse_STATUS_IN_PROCESS,
	}, nil
}

// TransferByAdmin registers a new transfer from one channel to another on
// behalf of the contract administrator, using business logic and maps the
// result to DTO request objects.
func (api *APIServer) TransferByAdmin(
	ctx context.Context,
	req *dto.TransferBeginAdminRequest,
) (*dto.TransferStatusResponse, error) {
	var err error

	if req == nil {
		return nil, ErrBadRequest
	}

	log := glog.FromContext(ctx)
	log.Set(
		glog.Field{K: "transfer.id", V: req.GetIdTransfer()},
		glog.Field{K: "transfer.from", V: req.GetGenerals().GetChannel()},
		glog.Field{K: "transfer.to", V: req.GetChannelTo()},
		glog.Field{K: "transfer.token", V: req.GetToken()},
	)
	md := telemetry.TransferMetadataFromContext(ctx)
	ctx, span := tracing.StartSpan(
		ctx,
		tracer,
		"api_server: TransferByAdmin",
		req,
	)
	defer func() {
		tracing.FinishSpan(span, err)
	}()
	log.Debug("transferByAdmin request received")
	tr, err := dtoBeginAdminToModelTransferRequest(req, api.actualChannels)
	if err != nil {
		err = errors.Errorf("parse transfer request: %w", err)
		return &dto.TransferStatusResponse{
				IdTransfer: req.GetIdTransfer(),
				Status:     dto.TransferStatusResponse_STATUS_ERROR,
				Message:    err.Error(),
			}, status.Error(
				codes.InvalidArgument,
				err.Error(),
			)
	}
	tr.Metadata = md

	if err = api.ctrl.TransferKeep(ctx, tr); err != nil {
		return nil, fmt.Errorf(
			"[APIServer] failed to save transfer request: %w",
			err,
		)
	}

	api.output <- tr

	return &dto.TransferStatusResponse{
		IdTransfer: string(tr.Transfer),
		Status:     dto.TransferStatusResponse_STATUS_IN_PROCESS,
	}, nil
}

// MultiTransferByCustomer registers a new transfer from one channel to another on
// behalf of the contract administrator, using business logic and maps the
// result to DTO request objects.
func (api *APIServer) MultiTransferByCustomer(
	ctx context.Context,
	req *dto.MultiTransferBeginCustomerRequest,
) (*dto.TransferStatusResponse, error) {
	var err error

	if req == nil {
		return nil, ErrBadRequest
	}

	log := glog.FromContext(ctx)
	log.Set(
		glog.Field{K: "transfer.id", V: req.GetIdTransfer()},
		glog.Field{K: "transfer.from", V: req.GetGenerals().GetChannel()},
		glog.Field{K: "transfer.to", V: req.GetChannelTo()},
		glog.Field{K: "transfer.items", V: req.GetItems()},
	)

	md := telemetry.TransferMetadataFromContext(ctx)
	ctx, span := tracing.StartSpan(
		ctx,
		tracer,
		"api_server: MultiTransferByCustomer",
		&MultiTransferDataGetter{
			ItemDataGetter:          req,
			BasicTransferDataGetter: req,
		},
	)
	defer func() {
		tracing.FinishSpan(span, err)
	}()
	log.Debug("multiTransferByCustomer request received")
	tr, err := dtoBeginCustomerToModelMultiTransferRequest(req, api.actualChannels)
	if err != nil {
		err = errors.Errorf("parse transfer request: %w", err)
		return &dto.TransferStatusResponse{
				IdTransfer: req.GetIdTransfer(),
				Status:     dto.TransferStatusResponse_STATUS_ERROR,
				Message:    err.Error(),
			}, status.Error(
				codes.InvalidArgument,
				err.Error(),
			)
	}
	tr.Metadata = md

	if err = api.ctrl.TransferKeep(ctx, tr); err != nil {
		return nil, fmt.Errorf(
			"[APIServer] failed to save transfer request: %w",
			err,
		)
	}

	api.output <- tr

	return &dto.TransferStatusResponse{
		IdTransfer: string(tr.Transfer),
		Status:     dto.TransferStatusResponse_STATUS_IN_PROCESS,
	}, nil
}

// MultiTransferByAdmin registers a new transfer from one channel to another on
// behalf of the contract administrator, using business logic and maps the
// result to DTO request objects.
func (api *APIServer) MultiTransferByAdmin(
	ctx context.Context,
	req *dto.MultiTransferBeginAdminRequest,
) (*dto.TransferStatusResponse, error) {
	var err error
	if req == nil {
		return nil, ErrBadRequest
	}

	log := glog.FromContext(ctx)
	log.Set(
		glog.Field{K: "transfer.id", V: req.GetIdTransfer()},
		glog.Field{K: "transfer.from", V: req.GetGenerals().GetChannel()},
		glog.Field{K: "transfer.to", V: req.GetChannelTo()},
		glog.Field{K: "transfer.items", V: req.GetItems()},
	)

	md := telemetry.TransferMetadataFromContext(ctx)
	ctx, span := tracing.StartSpan(
		ctx,
		tracer,
		"api_server: MultiTransferByAdmin",
		&MultiTransferDataGetter{
			ItemDataGetter:          req,
			BasicTransferDataGetter: req,
		},
	)
	defer func() {
		tracing.FinishSpan(span, err)
	}()
	log.Debug("multiTransferByAdmin request received")
	tr, err := dtoBeginAdminToModelMultiTransferRequest(req, api.actualChannels)
	if err != nil {
		err = errors.Errorf("parse transfer request: %w", err)
		return &dto.TransferStatusResponse{
				IdTransfer: req.GetIdTransfer(),
				Status:     dto.TransferStatusResponse_STATUS_ERROR,
				Message:    err.Error(),
			}, status.Error(
				codes.InvalidArgument,
				err.Error(),
			)
	}
	tr.Metadata = md

	if err = api.ctrl.TransferKeep(ctx, tr); err != nil {
		return nil, fmt.Errorf(
			"[APIServer] failed to save transfer request: %w",
			err,
		)
	}

	api.output <- tr

	return &dto.TransferStatusResponse{
		IdTransfer: string(tr.Transfer),
		Status:     dto.TransferStatusResponse_STATUS_IN_PROCESS,
	}, nil
}

// TransferStatus returns the current status of the transfer. It requests the
// status from the business logic and maps it to DTO request objects.
func (api *APIServer) TransferStatus(
	ctx context.Context,
	req *dto.TransferStatusRequest,
) (*dto.TransferStatusResponse, error) {
	if req == nil {
		return nil, ErrBadRequest
	}

	log := glog.FromContext(ctx)
	log.Set(
		glog.Field{K: "transfer.id", V: req.GetIdTransfer()},
	)

	exclStatus, exclOk, err := extractExcludeStatus(req.GetOptions())
	if err != nil {
		err = errors.Errorf("define exclude option: %w", err)
		return &dto.TransferStatusResponse{
				IdTransfer: req.GetIdTransfer(),
				Status:     dto.TransferStatusResponse_STATUS_ERROR,
				Message:    err.Error(),
			}, status.Error(
				codes.InvalidArgument,
				err.Error(),
			)
	}

	for ctx.Err() == nil {
		response, err := api.transferStatus(ctx, req.GetIdTransfer())
		if err != nil {
			return nil, err
		}
		if !exclOk || exclStatus != response.GetStatus() {
			return response, nil
		}
	}

	return &dto.TransferStatusResponse{
		IdTransfer: req.GetIdTransfer(),
		Status:     dto.TransferStatusResponse_STATUS_UNDEFINED,
		Message:    ctx.Err().Error(),
	}, nil
}

func (api *APIServer) transferStatus(ctx context.Context, transferID string) (*dto.TransferStatusResponse, error) {
	var (
		tr  model.TransferRequest
		err error
	)

	ctx, span := tracer.Start(ctx,
		"api_server: transferStatus",
		trace.WithAttributes(
			attribute.String("id", transferID),
		),
	)
	defer func() {
		tracing.FinishSpan(span, err)
	}()
	tr, err = api.ctrl.TransferFetch(ctx, model.ID(transferID))
	if err != nil {
		if strings.Contains(err.Error(), data.ErrObjectNotFound.Error()) {
			return nil,
				status.Error(
					codes.InvalidArgument,
					err.Error(),
				)
		}
		return nil, errors.Errorf("fetch transfer request: %w", ErrInvalidStatusCode)
	}

	code, ok := dto.TransferStatusResponse_Status_value[tr.Status]
	if !ok {
		return nil, errors.Errorf("fetch transfer status: %w", ErrInvalidStatusCode)
	}

	return &dto.TransferStatusResponse{
		IdTransfer: transferID,
		Status:     dto.TransferStatusResponse_Status(code),
		Message:    tr.Message,
	}, nil
}

// ItemDataGetter defines the interface for accessing a list of transfer items.
type ItemDataGetter interface {
	GetItems() []*dto.TransferItem
}

// MultiTransferDataGetter is a struct that satisfies the [tracing.TransferDataGetter] interface.
// It combines basic transfer data with item data.
type MultiTransferDataGetter struct {
	ItemDataGetter
	tracing.BasicTransferDataGetter
}

// GetToken returns a string representation of the tokens from all items in the multi-transfer.
func (m *MultiTransferDataGetter) GetToken() string {
	itemList := make([]string, 0, len(m.GetItems()))
	for _, item := range m.GetItems() {
		itemList = append(itemList, item.GetToken())
	}
	return fmt.Sprintf("%+v", itemList)
}

// GetAmount returns a string representation of the amounts from all items in the multi-transfer.
func (m *MultiTransferDataGetter) GetAmount() string {
	itemList := make([]string, 0, len(m.GetItems()))
	for _, item := range m.GetItems() {
		itemList = append(itemList, item.GetAmount())
	}
	return fmt.Sprintf("%+v", itemList)
}
