package transfer

import (
	"context"
	"fmt"
	"strings"

	"github.com/anoideaopen/channel-transfer/pkg/data"
	"github.com/anoideaopen/channel-transfer/pkg/model"
	dto "github.com/anoideaopen/channel-transfer/proto"
	"github.com/newity/glog"
	"github.com/pkg/errors"
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
func NewAPIServer(ctx context.Context, output chan<- model.TransferRequest, ctrl RequestController, actualChannels []string) *APIServer {
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
	if req == nil {
		return nil, ErrBadRequest
	}

	tr, err := dtoBeginCustomerToModelTransferRequest(req, api.actualChannels)
	if err != nil {
		err = errors.Wrap(err, "parse transfer request")
		return &dto.TransferStatusResponse{
				IdTransfer: req.IdTransfer,
				Status:     dto.TransferStatusResponse_STATUS_ERROR,
				Message:    err.Error(),
			}, status.Error(
				codes.InvalidArgument,
				err.Error(),
			)
	}

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
	if req == nil {
		return nil, ErrBadRequest
	}

	tr, err := dtoBeginAdminToModelTransferRequest(req, api.actualChannels)
	if err != nil {
		err = errors.Wrap(err, "parse transfer request")
		return &dto.TransferStatusResponse{
				IdTransfer: req.IdTransfer,
				Status:     dto.TransferStatusResponse_STATUS_ERROR,
				Message:    err.Error(),
			}, status.Error(
				codes.InvalidArgument,
				err.Error(),
			)
	}

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

	exclStatus, exclOk, err := extractExcludeStatus(req.Options)
	if err != nil {
		err = errors.Wrap(err, "define exclude option")
		return &dto.TransferStatusResponse{
				IdTransfer: req.IdTransfer,
				Status:     dto.TransferStatusResponse_STATUS_ERROR,
				Message:    err.Error(),
			}, status.Error(
				codes.InvalidArgument,
				err.Error(),
			)
	}

	for ctx.Err() == nil {
		response, err := api.transferStatus(ctx, req.IdTransfer)
		if err != nil {
			return nil, err
		}
		if !exclOk || (exclOk && exclStatus != response.Status) {
			return response, nil
		}
	}

	return &dto.TransferStatusResponse{
		IdTransfer: req.IdTransfer,
		Status:     dto.TransferStatusResponse_STATUS_UNDEFINED,
		Message:    ctx.Err().Error(),
	}, nil
}

func (api *APIServer) transferStatus(ctx context.Context, transferID string) (*dto.TransferStatusResponse, error) {
	tr, err := api.ctrl.TransferFetch(ctx, model.ID(transferID))
	if err != nil {
		if strings.Contains(err.Error(), data.ErrObjectNotFound.Error()) {
			return nil,
				status.Error(
					codes.InvalidArgument,
					err.Error(),
				)
		}
		return nil, errors.Wrap(ErrInvalidStatusCode, "fetch transfer request")
	}

	code, ok := dto.TransferStatusResponse_Status_value[tr.Status]
	if !ok {
		return nil, errors.Wrap(ErrInvalidStatusCode, "fetch transfer status")
	}

	return &dto.TransferStatusResponse{
		IdTransfer: transferID,
		Status:     dto.TransferStatusResponse_Status(code),
		Message:    tr.Message,
	}, nil
}
