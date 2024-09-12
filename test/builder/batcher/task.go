package batcher

import (
	fpb "github.com/anoideaopen/foundation/proto"
	"github.com/google/uuid"
)

type TaskBuilder struct {
	task *fpb.Task
}

func NewTaskBuilder() *TaskBuilder {
	return &TaskBuilder{
		task: &fpb.Task{
			Id:     "",
			Method: "",
			Args:   []string{},
		},
	}
}

func (b *TaskBuilder) SetID(id string) *TaskBuilder {
	b.task.Id = id
	return b
}

func (b *TaskBuilder) SetMethod(method string) *TaskBuilder {
	b.task.Method = method
	return b
}

func (b *TaskBuilder) SetArgs(args []string) *TaskBuilder {
	b.task.Args = args
	return b
}

func (b *TaskBuilder) Build() *fpb.Task {
	if b.task.GetId() == "" {
		b.task.Id = uuid.New().String()
	}
	return b.task
}
