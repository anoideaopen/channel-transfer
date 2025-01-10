package parser

import (
	"encoding/hex"

	"github.com/anoideaopen/channel-transfer/pkg/helpers/nerrors"
	"github.com/anoideaopen/channel-transfer/pkg/logger"
	"github.com/anoideaopen/channel-transfer/pkg/model"
	"github.com/anoideaopen/common-component/errorshlp"
	"github.com/anoideaopen/foundation/proto"
	"github.com/anoideaopen/glog"
	"github.com/go-errors/errors"
	"github.com/hyperledger/fabric-protos-go/common"
)

const (
	minUnicodeRuneValue = 0
)

type Parser struct {
	log         glog.Logger
	batchPrefix string
	channel     string
}

func NewParser(log glog.Logger, channel string, txPrefixes string) *Parser {
	return &Parser{
		log:         log.With(logger.Labels{Component: logger.ComponentParser, Channel: channel}.Fields()...),
		batchPrefix: txPrefixes,
		channel:     channel,
	}
}

func (p *Parser) ExtractData(block *common.Block) (*model.BlockData, error) {
	b, err := fromFabricBlock(block)
	if err != nil {
		return nil, errorshlp.WrapWithDetails(err, nerrors.ErrTypeParsing, nerrors.ComponentParser)
	}

	res := &model.BlockData{BlockNum: b.number}

	if b.isConfig {
		return res, nil
	}

	res.Txs, err = p.extractTxs(b.number, b.txs())
	if err != nil {
		return nil, errorshlp.WrapWithDetails(err, nerrors.ErrTypeParsing, nerrors.ComponentParser)
	}

	return res, nil
}

//nolint:gocognit,funlen
func (p *Parser) extractTxs(blockNum uint64, txs []prsTx) ([]model.Transaction, error) {
	tOperations := make([]model.Transaction, 0)
	for txNum, tx := range txs {
		if !tx.isValid() {
			p.log.Debugf("skip invalid transaction (number %d in block)", txNum)
			continue
		}

		channelHeader, err := tx.channelHeader()
		if err != nil {
			return nil, errors.Errorf("failed to get tx channel header: %w", err)
		}

		if common.HeaderType(channelHeader.GetType()) != common.HeaderType_ENDORSER_TRANSACTION {
			continue
		}

		actions, err := tx.getActions()
		if err != nil {
			return nil, errors.Errorf("failed to get tx action payload: %w", err)
		}

		for _, action := range actions {
			method, args, err := p.extractorActionPayload(action.payload)
			if err != nil {
				return nil, errors.Errorf("failed to get action args: %w", err)
			}

			if !new(model.TransactionKind).Is(method) && method != BatchExecuteMethod && method != ExecuteTasksMethod {
				continue
			}

			rwSets, err := action.rwSets()
			if err != nil {
				return nil, errors.Errorf("failed to get action rw_set: %w", err)
			}

			chaincodeAction, err := action.chaincodeAction()
			if err != nil {
				return nil, errors.Errorf("failed to get  chaincodeAction: %w", err)
			}

			if method == BatchExecuteMethod {
				for _, txID := range p.extractBatchPreImageTxIDs(rwSets) {
					batchResponse, err := p.extractBatchResponse(chaincodeAction.GetResponse().GetPayload())
					if err != nil {
						return nil, errors.Errorf("failed to extract batchResponse from chaincodeAction response payload: %w", err)
					}
					for _, tsResponse := range batchResponse.GetTxResponses() {
						if hex.EncodeToString(tsResponse.GetId()) == txID && new(model.TransactionKind).Is(tsResponse.GetMethod()) {
							tOperations = append(
								tOperations,
								model.Transaction{
									Channel:        p.channel,
									BlockNum:       blockNum,
									TxID:           txID,
									FuncName:       tsResponse.GetMethod(),
									Args:           nil,
									TimeNs:         0,
									ValidationCode: tx.validationCode,
									BatchResponse:  tsResponse,
									Response:       nil,
								},
							)
						}
					}
				}
				continue
			}

			if method == ExecuteTasksMethod {
				extractTaskRequest, err := p.extractTaskRequest(args[1])
				if err != nil {
					return nil, errors.Errorf("failed to extract taskRequest from chaincodeAction response payload: %w", err)
				}
				for _, task := range extractTaskRequest.GetTasks() {
					batchResponse, err := p.extractBatchResponse(chaincodeAction.GetResponse().GetPayload())
					if err != nil {
						return nil, errors.Errorf("failed to extract batchResponse from chaincodeAction response payload: %w", err)
					}
					for _, tsResponse := range batchResponse.GetTxResponses() {
						if string(tsResponse.GetId()) == task.GetId() && new(model.TransactionKind).Is(tsResponse.GetMethod()) {
							tOperations = append(
								tOperations,
								// add task as a separate operation
								model.Transaction{
									Channel:        p.channel,
									BlockNum:       blockNum,
									TxID:           task.GetId(),
									FuncName:       task.GetMethod(),
									Args:           argsFromTask(task),
									TimeNs:         uint64(channelHeader.GetTimestamp().AsTime().UnixNano()),
									ValidationCode: tx.validationCode,
									BatchResponse:  nil,
									Response:       nil,
								},
								// add task response as a separate operation
								model.Transaction{
									Channel:        p.channel,
									BlockNum:       blockNum,
									TxID:           task.GetId(),
									FuncName:       tsResponse.GetMethod(),
									Args:           nil,
									TimeNs:         0,
									ValidationCode: tx.validationCode,
									BatchResponse:  tsResponse,
									Response:       nil,
								},
							)
						}
					}
				}
				continue
			}

			tOperations = append(
				tOperations,
				model.Transaction{
					Channel:        p.channel,
					BlockNum:       blockNum,
					TxID:           channelHeader.GetTxId(),
					FuncName:       method,
					Args:           args,
					TimeNs:         uint64(channelHeader.GetTimestamp().AsTime().UnixNano()),
					ValidationCode: tx.validationCode,
					BatchResponse:  nil,
					Response:       chaincodeAction.GetResponse(),
				},
			)
		}
	}

	return tOperations, nil
}

func argsFromTask(task *proto.Task) [][]byte {
	argsBytes := [][]byte{[]byte(task.GetMethod())}
	for _, arg := range task.GetArgs() {
		argsBytes = append(argsBytes, []byte(arg))
	}
	return argsBytes
}
