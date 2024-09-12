package parser_test

import (
	"context"
	"testing"

	"github.com/anoideaopen/channel-transfer/pkg/hlf/parser"
	"github.com/anoideaopen/channel-transfer/test/builder/batcher"
	"github.com/anoideaopen/channel-transfer/test/builder/chaincode"
	fpb "github.com/anoideaopen/foundation/proto"
	"github.com/anoideaopen/glog"
	"github.com/hyperledger/fabric-protos-go/common"
	"github.com/hyperledger/fabric-protos-go/peer"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func newBlock(channel, txId string, txMethod string) *common.Block {
	task := batcher.NewTaskBuilder().
		SetID(txId).
		SetMethod(txMethod).
		SetArgs([]string{"arg1", "arg2"}).
		Build()

	executeTasksRequest := batcher.NewExecuteTasksRequestBuilder().
		AddTask(task).
		Marshal()

	ccInvocationSpec := chaincode.NewChaincodeInvocationSpecBuilder().
		SetChaincodeSpec(&peer.ChaincodeSpec{
			Type: peer.ChaincodeSpec_Type(1),
			ChaincodeId: &peer.ChaincodeID{
				Name:    "chaincodeName",
				Path:    "/path/to/chaincode",
				Version: "v1",
			},
			Input: &peer.ChaincodeInput{
				Args: [][]byte{[]byte(parser.ExecuteTasksMethod), executeTasksRequest},
			},
		}).
		Marshal()

	batchResponse := chaincode.NewBatchResponseBuilder().
		AddTxResponse(&fpb.TxResponse{
			Id:     []byte(txId),
			Method: txMethod,
		}).
		Marshal()

	chaincodeAction := chaincode.NewChaincodeActionBuilder().
		SetEvents([]byte("events")).
		SetResponse(&peer.Response{
			Payload: batchResponse,
		}).
		Marshal()

	chaincodeEndorsedAction := chaincode.NewChaincodeEndorsedActionBuilder().
		SetProposalResponsePayload([]byte("proposalHash"), chaincodeAction).
		AddEndorsement([]byte("signature_1"), []byte("endorser_1")).
		AddEndorsement([]byte("signature_2"), []byte("endorser_2")).
		Build()

	chaincodeActionPayload := chaincode.NewChaincodeActionPayloadBuilder().
		SetChaincodeProposalPayload(&peer.ChaincodeProposalPayload{
			Input: ccInvocationSpec,
		}).
		SetEndorsedAction(chaincodeEndorsedAction).
		Build()

	transaction := chaincode.NewTransactionBuilder().
		AddAction(
			[]byte("header1"),
			chaincodeActionPayload,
		).
		Build()

	envelope := chaincode.NewEnvelopeBuilder().
		SetChannelHeader(&common.ChannelHeader{
			Type:      int32(common.HeaderType_ENDORSER_TRANSACTION),
			Version:   1,
			Timestamp: timestamppb.Now(),
			ChannelId: channel,
			TxId:      "12345",
		}).
		SetSignatureHeader(&common.SignatureHeader{
			Creator: []byte("creator-identity"),
			Nonce:   []byte("random-nonce"),
		}).
		SetData(transaction).
		SetSignature([]byte("signature-data")).
		Build()

	block := batcher.NewBlockBuilder().
		SetHeader(1, []byte("previousHash"), []byte("dataHash")).
		AddData(envelope).
		AddMetadata(
			int(common.BlockMetadataIndex_SIGNATURES),
			[]byte("value"),
			[]byte("signature metadata"),
			[]byte("signatureHeader metadata"),
			[]byte("identifierHeader metadata"),
		).
		AddMetadata(
			int(common.BlockMetadataIndex_TRANSACTIONS_FILTER),
			nil,
			nil,
			nil,
			nil,
		).
		Build()

	return block
}

func TestExtractData_ExecuteTasksMethod(t *testing.T) {
	var (
		channel        string = "test-channel"
		blockNum       uint64 = 1
		txId           string = "unique-task-id"
		funcName       string = "deleteCCTransferTo"
		args           [][]uint8
		timeNs         uint64 = 0
		validationCode int32  = 0
		response       *peer.Response
	)

	block := newBlock(channel, txId, funcName)

	ctx := context.Background()

	log := glog.FromContext(ctx)

	p := parser.NewParser(log, "test-channel", "batchPrefix")

	blockData, err := p.ExtractData(block)
	if err != nil {
		t.Fatalf("ExtractData failed: %v", err)
	}

	assert.NotNil(t, blockData, "BlockData should not be nil")
	assert.NotEmpty(t, blockData.Txs, "Transactions should not be empty")
	assert.Len(t, blockData.Txs, 1)

	tx := blockData.Txs[0]
	assert.Equal(t, channel, tx.Channel)
	assert.Equal(t, blockNum, tx.BlockNum)
	assert.Equal(t, txId, tx.TxID)
	assert.Equal(t, funcName, tx.FuncName)
	assert.Equal(t, args, tx.Args)
	assert.Equal(t, timeNs, tx.TimeNs)
	assert.Equal(t, validationCode, tx.ValidationCode)
	assert.NotNil(t, tx.BatchResponse)
	assert.Equal(t, "deleteCCTransferTo", tx.BatchResponse.Method)
	assert.Equal(t, response, tx.Response)
}
