package transfer

import (
	"encoding/json"
	"strconv"
	"strings"

	"github.com/anoideaopen/channel-transfer/pkg/model"
	dto "github.com/anoideaopen/channel-transfer/proto"
	"github.com/anoideaopen/foundation/proto"
	"github.com/go-errors/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	proto2 "google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/known/typepb"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

const validProcessingType = "google.protobuf.StringValue"

func dtoBeginAdminToModelTransferRequest(
	in *dto.TransferBeginAdminRequest,
	channels map[string]struct{},
) (model.TransferRequest, error) {
	if in.GetGenerals() == nil {
		return model.TransferRequest{}, ErrBadRequest
	}

	if err := checkAdminRequestTransfer(in, channels); err != nil {
		return model.TransferRequest{}, err
	}

	return model.TransferRequest{
		Request:   model.ID(in.GetGenerals().GetRequestId()),
		Method:    in.GetGenerals().GetMethodName(),
		Chaincode: in.GetGenerals().GetChaincode(),
		Channel:   in.GetGenerals().GetChannel(),
		Nonce:     in.GetGenerals().GetNonce(),
		PublicKey: in.GetGenerals().GetPublicKey(),
		Sign:      in.GetGenerals().GetSign(),
		Transfer:  model.ID(in.GetIdTransfer()),
		To:        in.GetChannelTo(),
		Token:     in.GetToken(),
		Amount:    in.GetAmount(),
		User:      model.ID(in.GetAddress()),
		TransferResult: model.TransferResult{
			Status:  dto.TransferStatusResponse_STATUS_IN_PROCESS.String(),
			Message: "",
		},
	}, nil
}

func dtoBeginCustomerToModelTransferRequest(
	in *dto.TransferBeginCustomerRequest,
	channels map[string]struct{},
) (model.TransferRequest, error) {
	if in.GetGenerals() == nil {
		return model.TransferRequest{}, ErrBadRequest
	}

	if err := checkCustomerRequestTransfer(in, channels); err != nil {
		return model.TransferRequest{}, err
	}

	return model.TransferRequest{
		Request:   model.ID(in.GetGenerals().GetRequestId()),
		Method:    in.GetGenerals().GetMethodName(),
		Chaincode: in.GetGenerals().GetChaincode(),
		Channel:   in.GetGenerals().GetChannel(),
		Nonce:     in.GetGenerals().GetNonce(),
		PublicKey: in.GetGenerals().GetPublicKey(),
		Sign:      in.GetGenerals().GetSign(),
		Transfer:  model.ID(in.GetIdTransfer()),
		To:        in.GetChannelTo(),
		Token:     in.GetToken(),
		Amount:    in.GetAmount(),
		TransferResult: model.TransferResult{
			Status:  dto.TransferStatusResponse_STATUS_IN_PROCESS.String(),
			Message: "",
		},
	}, nil
}

func dtoBeginAdminToModelMultiTransferRequest(
	in *dto.MultiTransferBeginAdminRequest,
	channels map[string]struct{},
) (model.TransferRequest, error) {
	if in.GetGenerals() == nil {
		return model.TransferRequest{}, ErrBadRequest
	}

	if err := checkAdminRequestMultiTransfer(in, channels); err != nil {
		return model.TransferRequest{}, err
	}

	mappedItems := make([]model.TransferItem, len(in.GetItems()))
	for i, item := range in.GetItems() {
		mappedItems[i] = model.TransferItem{
			Token:  item.GetToken(),
			Amount: item.GetAmount(),
		}
	}

	return model.TransferRequest{
		Request:   model.ID(in.GetGenerals().GetRequestId()),
		Method:    in.GetGenerals().GetMethodName(),
		Chaincode: in.GetGenerals().GetChaincode(),
		Channel:   in.GetGenerals().GetChannel(),
		Nonce:     in.GetGenerals().GetNonce(),
		PublicKey: in.GetGenerals().GetPublicKey(),
		Sign:      in.GetGenerals().GetSign(),
		Transfer:  model.ID(in.GetIdTransfer()),
		To:        in.GetChannelTo(),
		Items:     mappedItems,
		User:      model.ID(in.GetAddress()),
		TransferResult: model.TransferResult{
			Status:  dto.TransferStatusResponse_STATUS_IN_PROCESS.String(),
			Message: "",
		},
	}, nil
}

func dtoBeginCustomerToModelMultiTransferRequest(
	in *dto.MultiTransferBeginCustomerRequest,
	channels map[string]struct{},
) (model.TransferRequest, error) {
	if in.GetGenerals() == nil {
		return model.TransferRequest{}, ErrBadRequest
	}

	if err := checkCustomerRequestMultiTransfer(in, channels); err != nil {
		return model.TransferRequest{}, err
	}

	mappedItems := make([]model.TransferItem, len(in.GetItems()))
	for i, item := range in.GetItems() {
		mappedItems[i] = model.TransferItem{
			Token:  item.GetToken(),
			Amount: item.GetAmount(),
		}
	}

	return model.TransferRequest{
		Request:   model.ID(in.GetGenerals().GetRequestId()),
		Method:    in.GetGenerals().GetMethodName(),
		Chaincode: in.GetGenerals().GetChaincode(),
		Channel:   in.GetGenerals().GetChannel(),
		Nonce:     in.GetGenerals().GetNonce(),
		PublicKey: in.GetGenerals().GetPublicKey(),
		Sign:      in.GetGenerals().GetSign(),
		Transfer:  model.ID(in.GetIdTransfer()),
		To:        in.GetChannelTo(),
		Items:     mappedItems,
		TransferResult: model.TransferResult{
			Status:  dto.TransferStatusResponse_STATUS_IN_PROCESS.String(),
			Message: "",
		},
	}, nil
}

func LedgerBlockToTransferBlock(channel string, block model.BlockData) map[model.ID]*model.TransferBlock {
	transferBlocks := make(map[model.ID]*model.TransferBlock)
	for _, tx := range block.Txs {
		if tx.FuncName != model.TxChannelTransferByCustomer.String() &&
			tx.FuncName != model.TxChannelTransferByAdmin.String() &&
			tx.FuncName != model.TxCreateCCTransferTo.String() &&
			tx.FuncName != model.TxChannelMultiTransferByCustomer.String() &&
			tx.FuncName != model.TxChannelMultiTransferByAdmin.String() {
			continue
		}

		key := transferID(tx)
		if _, ok := transferBlocks[key]; !ok {
			transferBlocks[key] = &model.TransferBlock{
				Channel:  model.ID(channel),
				Transfer: key,
			}
		}
		transferBlocks[key].Transactions = append(transferBlocks[key].Transactions, model.Transaction{
			Channel:        tx.Channel,
			BlockNum:       tx.BlockNum,
			TxID:           tx.TxID,
			FuncName:       tx.FuncName,
			Args:           tx.Args,
			TimeNs:         tx.TimeNs,
			ValidationCode: tx.ValidationCode,
			BatchResponse:  tx.BatchResponse,
			Response:       tx.Response,
		})
	}
	return transferBlocks
}

func transferID(tx model.Transaction) model.ID {
	if len(tx.Args) < 7 { //nolint:gomnd
		if tx.FuncName == model.TxCreateCCTransferTo.String() && len(tx.Args) > 1 {
			ccTransfer := proto.CCTransfer{}
			if err := json.Unmarshal(tx.Args[1], &ccTransfer); err != nil {
				return ""
			}
			return model.ID(ccTransfer.GetId())
		}
		return ""
	}
	return model.ID(tx.Args[4])
}

func BlockToRequest(block model.TransferBlock) (request model.TransferRequest) {
	request.Transfer = block.Transfer
	request.Status = dto.TransferStatusResponse_STATUS_UNDEFINED.String()
	for _, transaction := range block.Transactions {
		if transaction.FuncName != model.TxChannelTransferByCustomer.String() &&
			transaction.FuncName != model.TxChannelTransferByAdmin.String() &&
			transaction.FuncName != model.TxChannelMultiTransferByCustomer.String() &&
			transaction.FuncName != model.TxChannelMultiTransferByAdmin.String() {
			continue
		}

		request.Transfer = block.Transfer
		offset := 0

		if transaction.FuncName == model.TxChannelTransferByAdmin.String() ||
			transaction.FuncName == model.TxChannelMultiTransferByAdmin.String() {
			request.User = model.ID(transaction.Args[6])
			offset = 1
		}

		if len(transaction.Args) < 11+offset {
			continue
		}
		// TODO: ???
		request.Method = string(transaction.Args[0])
		request.Request = model.ID(transaction.Args[1])
		request.Channel = string(transaction.Args[2])
		request.Chaincode = string(transaction.Args[3])
		request.To = string(transaction.Args[5])
		request.Token = string(transaction.Args[6+offset])
		request.Amount = string(transaction.Args[7+offset])
		request.Nonce = string(transaction.Args[8+offset])
		request.PublicKey = string(transaction.Args[9+offset])
		request.Sign = string(transaction.Args[10+offset])

		break
	}

	return
}

func checkGeneral(gp *dto.GeneralParams, actualChannels map[string]struct{}) error {
	if gp.GetMethodName() == "" {
		return ErrMethod
	}
	if gp.GetMethodName() != model.TxChannelTransferByAdmin.String() &&
		gp.GetMethodName() != model.TxChannelTransferByCustomer.String() &&
		gp.GetMethodName() != model.TxChannelMultiTransferByAdmin.String() &&
		gp.GetMethodName() != model.TxChannelMultiTransferByCustomer.String() {
		return ErrUnknownMethod
	}
	if _, ok := actualChannels[gp.GetChannel()]; !ok {
		return ErrBadChannel
	}
	if gp.GetChaincode() == "" {
		return ErrChaincode
	}
	if gp.GetSign() == "" {
		return ErrSign
	}
	if gp.GetNonce() == "" {
		return ErrNonce
	}
	if gp.GetPublicKey() == "" {
		return ErrPubKey
	}

	return nil
}

func checkAdminRequestTransfer(
	tAdminRequest *dto.TransferBeginAdminRequest,
	actualChannels map[string]struct{},
) error {
	if err := checkGeneral(tAdminRequest.GetGenerals(), actualChannels); err != nil {
		return err
	}
	if tAdminRequest.GetAddress() == "" {
		return errors.New("address undefined")
	}
	if tAdminRequest.GetChannelTo() == "" {
		return errors.New("channel TO undefined")
	}
	if tAdminRequest.GetToken() == "" {
		return errors.New("token undefined")
	}
	if tAdminRequest.GetAmount() == "" {
		return errors.New("amount undefined")
	}
	if _, err := strconv.ParseInt(tAdminRequest.GetAmount(), 10, 64); err != nil {
		return errors.New("amount is not a number")
	}

	return verifyChannels(
		tAdminRequest.GetGenerals().GetChannel(),
		tAdminRequest.GetChannelTo(),
		tAdminRequest.GetToken(),
	)
}

func checkCustomerRequestTransfer(
	tCustomerRequest *dto.TransferBeginCustomerRequest,
	actualChannels map[string]struct{},
) error {
	if err := checkGeneral(tCustomerRequest.GetGenerals(), actualChannels); err != nil {
		return err
	}
	if tCustomerRequest.GetChannelTo() == "" {
		return errors.New("channel TO undefined")
	}
	if tCustomerRequest.GetToken() == "" {
		return errors.New("token undefined")
	}
	if tCustomerRequest.GetAmount() == "" {
		return errors.New("amount undefined")
	}
	if _, err := strconv.ParseInt(tCustomerRequest.GetAmount(), 10, 64); err != nil {
		return errors.New("amount is not a number")
	}

	return verifyChannels(
		tCustomerRequest.GetGenerals().GetChannel(),
		tCustomerRequest.GetChannelTo(),
		tCustomerRequest.GetToken(),
	)
}

func checkAdminRequestMultiTransfer(
	tAdminRequest *dto.MultiTransferBeginAdminRequest,
	actualChannels map[string]struct{},
) error {
	if err := checkGeneral(tAdminRequest.GetGenerals(), actualChannels); err != nil {
		return err
	}
	if tAdminRequest.GetAddress() == "" {
		return errors.New("address undefined")
	}
	if tAdminRequest.GetChannelTo() == "" {
		return errors.New("channel TO undefined")
	}
	if len(tAdminRequest.GetItems()) == 0 {
		return errors.New("items is empty")
	}
	for _, item := range tAdminRequest.GetItems() {
		if item.GetToken() == "" {
			return errors.New("token undefined")
		}
		if item.GetAmount() == "" {
			return errors.New("amount undefined")
		}
		if _, err := strconv.ParseInt(item.GetAmount(), 10, 64); err != nil {
			return errors.New("amount is not a number")
		}
	}

	for _, item := range tAdminRequest.GetItems() {
		err := verifyChannels(
			tAdminRequest.GetGenerals().GetChannel(),
			tAdminRequest.GetChannelTo(),
			item.GetToken(),
		)
		if err != nil {
			return err
		}
	}

	return nil
}

func checkCustomerRequestMultiTransfer(
	tCustomerRequest *dto.MultiTransferBeginCustomerRequest,
	actualChannels map[string]struct{},
) error {
	if err := checkGeneral(tCustomerRequest.GetGenerals(), actualChannels); err != nil {
		return err
	}
	if tCustomerRequest.GetChannelTo() == "" {
		return errors.New("channel TO undefined")
	}
	if len(tCustomerRequest.GetItems()) == 0 {
		return errors.New("items is empty")
	}
	for _, item := range tCustomerRequest.GetItems() {
		if item.GetToken() == "" {
			return errors.New("token undefined")
		}
		if item.GetAmount() == "" {
			return errors.New("amount undefined")
		}
		if _, err := strconv.ParseInt(item.GetAmount(), 10, 64); err != nil {
			return errors.New("amount is not a number")
		}
	}

	for _, item := range tCustomerRequest.GetItems() {
		err := verifyChannels(
			tCustomerRequest.GetGenerals().GetChannel(),
			tCustomerRequest.GetChannelTo(),
			item.GetToken(),
		)
		if err != nil {
			return err
		}
	}

	return nil
}

func tokenSymbol(token string) string {
	parts := strings.Split(token, "_")
	return parts[0]
}

func verifyChannels(from, to, token string) error {
	if to == from {
		return errors.New("channel TO set incorrectly")
	}

	tokenChannel := tokenSymbol(token)

	if !strings.EqualFold(from, tokenChannel) &&
		!strings.EqualFold(to, tokenChannel) {
		return errors.New("token set incorrectly")
	}

	return nil
}

func extractExcludeStatus(options []*typepb.Option) (dto.TransferStatusResponse_Status, bool, error) {
	for _, option := range options {
		s, ok, err := statusOption(option)
		if err != nil || ok {
			return s, ok, err
		}
	}
	return dto.TransferStatusResponse_STATUS_UNDEFINED, false, nil
}

func statusOption(option *typepb.Option) (dto.TransferStatusResponse_Status, bool, error) {
	if option.GetName() != StatusOptionFilterName {
		return dto.TransferStatusResponse_STATUS_UNDEFINED, false, nil
	}
	msg, err := option.GetValue().UnmarshalNew()
	if err != nil {
		return dto.TransferStatusResponse_STATUS_UNDEFINED,
			false,
			errors.Errorf("unmarshal protobuf option value: %w", err)
	}

	mt, err := protoregistry.GlobalTypes.FindMessageByName(proto2.MessageName(msg))
	if err != nil {
		return dto.TransferStatusResponse_STATUS_UNDEFINED,
			false,
			errors.Errorf("look up protobuf message by its full name: %w", err)
	}
	if mt.Descriptor().FullName() != validProcessingType {
		return dto.TransferStatusResponse_STATUS_UNDEFINED,
			false,
			status.Errorf(codes.InvalidArgument, "status option not "+validProcessingType)
	}

	nm, ok := msg.(*wrapperspb.StringValue)
	if !ok {
		return dto.TransferStatusResponse_STATUS_UNDEFINED,
			false,
			errors.New("impossible to convert status option to " + validProcessingType)
	}

	if s, ok := dto.TransferStatusResponse_Status_value[nm.GetValue()]; ok {
		transferStatus := dto.TransferStatusResponse_Status(s)
		if transferStatus == dto.TransferStatusResponse_STATUS_IN_PROCESS {
			return transferStatus, true, nil
		}
		return dto.TransferStatusResponse_STATUS_UNDEFINED,
			false,
			status.Error(codes.InvalidArgument, "exclude status not valid")
	}

	return dto.TransferStatusResponse_STATUS_UNDEFINED,
		false,
		status.Error(codes.InvalidArgument, "exclude status not found")
}
