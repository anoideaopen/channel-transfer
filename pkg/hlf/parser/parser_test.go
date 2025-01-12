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

func newBlock(channel, txId string, blockTime *timestamppb.Timestamp, txMethod string, args []string) *common.Block {
	task := batcher.NewTaskBuilder().
		SetID(txId).
		SetMethod(txMethod).
		SetArgs(args).
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
			Timestamp: blockTime,
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
		channel               = "test-channel"
		txId                  = "unique-task-id"
		funcName              = "deleteCCTransferTo"
		args                  = []string{"arg1", "arg2"}
		blockNum       uint64 = 1
		timeNs         uint64 = 0
		validationCode int32  = 0
	)

	expectedArgs := [][]byte{[]byte(funcName)}
	for _, arg := range args {
		expectedArgs = append(expectedArgs, []byte(arg))
	}

	blockTime := timestamppb.Now()
	timeNs = uint64(blockTime.AsTime().UnixNano())

	block := newBlock(channel, txId, blockTime, funcName, args)

	ctx := context.Background()

	log := glog.FromContext(ctx)

	p := parser.NewParser(log, "test-channel", "batchPrefix")

	blockData, err := p.ExtractData(block)
	if err != nil {
		t.Fatalf("ExtractData failed: %v", err)
	}

	assert.NotNil(t, blockData, "BlockData should not be nil")
	assert.NotEmpty(t, blockData.Txs, "Transactions should not be empty")
	assert.Len(t, blockData.Txs, 2)

	operationTask := blockData.Txs[0]
	assert.Equal(t, channel, operationTask.Channel)
	assert.Equal(t, blockNum, operationTask.BlockNum)
	assert.Equal(t, txId, operationTask.TxID)
	assert.Equal(t, funcName, operationTask.FuncName)
	assert.Equal(t, expectedArgs, operationTask.Args)
	assert.Equal(t, timeNs, operationTask.TimeNs)
	assert.Equal(t, validationCode, operationTask.ValidationCode)
	assert.Nil(t, operationTask.BatchResponse)

	operationResponse := blockData.Txs[1]
	assert.Equal(t, channel, operationResponse.Channel)
	assert.Equal(t, blockNum, operationResponse.BlockNum)
	assert.Equal(t, txId, operationResponse.TxID)
	assert.Equal(t, funcName, operationResponse.FuncName)
	assert.Equal(t, validationCode, operationResponse.ValidationCode)
	assert.NotNil(t, operationResponse.BatchResponse)
	assert.Equal(t, funcName, operationResponse.BatchResponse.Method)
	assert.Nil(t, operationResponse.Args)
	assert.Nil(t, operationResponse.Response)
	assert.Zero(t, operationResponse.TimeNs)
}
