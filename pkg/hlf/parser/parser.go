package parser

import (
	"encoding/hex"
	"time"

	"github.com/anoideaopen/channel-transfer/pkg/helpers/nerrors"
	"github.com/anoideaopen/channel-transfer/pkg/logger"
	"github.com/anoideaopen/channel-transfer/pkg/model"
	"github.com/anoideaopen/common-component/errorshlp"
	"github.com/anoideaopen/glog"
	"github.com/hyperledger/fabric-protos-go/common"
	"github.com/pkg/errors"
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
			return nil, errors.Wrap(err, "failed to get tx channel header")
		}

		if common.HeaderType(channelHeader.Type) != common.HeaderType_ENDORSER_TRANSACTION {
			continue
		}

		actions, err := tx.getActions()
		if err != nil {
			return nil, errors.Wrap(err, "failed to get tx action payload")
		}

		for _, action := range actions {
			method, args, err := p.extractorActionPayload(action.payload)
			if err != nil {
				return nil, errors.Wrap(err, "failed to get action args")
			}

			if !new(model.TransactionKind).Is(method) && method != BatchExecuteMethod {
				continue
			}

			rwSets, err := action.rwSets()
			if err != nil {
				return nil, errors.Wrap(err, "failed to get action rw_set")
			}

			chaincodeAction, err := action.chaincodeAction()
			if err != nil {
				return nil, errors.Wrap(err, "failed to get  chaincodeAction")
			}

			if method == BatchExecuteMethod {
				for _, txID := range p.extractBatchPreImageTxIDs(rwSets) {
					batchResponse, err := p.extractBatchResponse(chaincodeAction.Response.Payload)
					if err != nil {
						return nil, errors.Wrap(err, "failed to extract batchResponse from chaincodeAction response payload")
					}
					for _, tsResponse := range batchResponse.TxResponses {
						if hex.EncodeToString(tsResponse.Id) == txID && new(model.TransactionKind).Is(tsResponse.Method) {
							tOperations = append(
								tOperations,
								model.Transaction{
									Channel:        p.channel,
									BlockNum:       blockNum,
									TxID:           txID,
									FuncName:       tsResponse.Method,
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
					TxID:           channelHeader.TxId,
					FuncName:       method,
					Args:           args,
					TimeNs:         uint64(time.Unix(channelHeader.Timestamp.Seconds, int64(channelHeader.Timestamp.Nanos)).UnixNano()),
					ValidationCode: tx.validationCode,
					BatchResponse:  nil,
					Response:       chaincodeAction.Response,
				},
			)
		}
	}

	return tOperations, nil
}
