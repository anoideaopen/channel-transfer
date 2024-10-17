package model

import (
	"encoding/json"

	"github.com/anoideaopen/channel-transfer/pkg/data"
)

type Metadata struct {
	TraceID string
	SpanID  string
}

func (md *Metadata) MarshalBinary() (data []byte, err error) {
	return json.Marshal(md)
}

func (md *Metadata) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, md)
}

func (md *Metadata) Clone() data.Object {
	mdCopy := *md
	return &mdCopy
}

func (md *Metadata) Instance() data.Type {
	return data.InstanceOf(md)
}
