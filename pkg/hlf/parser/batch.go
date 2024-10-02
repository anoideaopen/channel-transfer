package parser

import (
	fpb "github.com/anoideaopen/foundation/proto"
	"github.com/golang/protobuf/proto" //nolint:staticcheck
	"github.com/hyperledger/fabric-protos-go/peer"
	"google.golang.org/protobuf/encoding/protojson"
)

const (
	BatchExecuteMethod = "batchExecute"
	ExecuteTasksMethod = "executeTasks"
)

func (p *Parser) extractBatchPreImageTxIDs(rwSets []prsRwSet) []string {
	txIDs := make([]string, 0)
	for _, rw := range rwSets {
		for _, write := range rw.kvRWSet.GetWrites() {
			if !write.GetIsDelete() {
				continue
			}
			pos, ok := p.hasPrefix(write.GetKey(), p.batchPrefix)
			if !ok {
				continue
			}
			txIDs = append(txIDs, write.GetKey()[pos:len(write.GetKey())-1])
		}
	}
	return txIDs
}

func (p *Parser) hasPrefix(compositeID, prefix string) (int, bool) {
	const countZeroRunes = 2
	if (len(compositeID) < len(prefix)+countZeroRunes) ||
		compositeID[0] != minUnicodeRuneValue ||
		compositeID[len(prefix)+1] != minUnicodeRuneValue ||
		compositeID[1:len(prefix)+1] != prefix {
		return 0, false
	}

	return len(prefix) + countZeroRunes, true
}

func (p *Parser) extractChaincodeArgs(input *peer.ChaincodeInput) (string, [][]byte) {
	return string(input.GetArgs()[0]), input.GetArgs()
}

func (p *Parser) extractBatchResponse(payload []byte) (*fpb.BatchResponse, error) {
	response := &fpb.BatchResponse{}
	if err := proto.Unmarshal(payload, response); err != nil {
		return nil, err
	}

	return response, nil
}

func (p *Parser) extractTaskRequest(payload []byte) (*fpb.ExecuteTasksRequest, error) {
	response := &fpb.ExecuteTasksRequest{}
	if err := proto.Unmarshal(payload, response); err != nil {
		if err = protojson.Unmarshal(payload, response); err != nil {
			return nil, err
		}
	}

	return response, nil
}
