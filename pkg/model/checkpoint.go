package model

import (
	"encoding/json"

	"github.com/anoideaopen/channel-transfer/pkg/data"
)

type Checkpoint struct {
	Channel                 ID
	Ver                     int64
	SrcCollectFromBlockNums uint64
}

func (cp *Checkpoint) MarshalBinary() (data []byte, err error) {
	return json.Marshal(cp)
}

func (cp *Checkpoint) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, cp)
}

func (cp *Checkpoint) Clone() data.Object {
	cpCopy := *cp
	return &cpCopy
}

func (cp *Checkpoint) Instance() data.Type {
	return data.InstanceOf(cp)
}
