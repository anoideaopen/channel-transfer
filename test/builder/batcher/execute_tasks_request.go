package batcher

import (
	fpb "github.com/anoideaopen/foundation/proto"
	"github.com/go-errors/errors"
	"github.com/golang/protobuf/proto" //nolint:staticcheck
)

type ExecuteTasksRequestBuilder struct {
	request *fpb.ExecuteTasksRequest
}

func NewExecuteTasksRequestBuilder() *ExecuteTasksRequestBuilder {
	return &ExecuteTasksRequestBuilder{
		request: &fpb.ExecuteTasksRequest{
			Tasks: []*fpb.Task{},
		},
	}
}

func (b *ExecuteTasksRequestBuilder) AddTask(task *fpb.Task) *ExecuteTasksRequestBuilder {
	b.request.Tasks = append(b.request.Tasks, task)
	return b
}

func (b *ExecuteTasksRequestBuilder) Build() *fpb.ExecuteTasksRequest {
	return b.request
}

func (b *ExecuteTasksRequestBuilder) Marshal() []byte {
	requestBytes, err := proto.Marshal(b.request)
	if err != nil {
		panic(errors.Errorf("Failed to marshal ExecuteTasksRequest: %v\n", err))
	}

	return requestBytes
}
