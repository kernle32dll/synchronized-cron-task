package crontask

import (
	"encoding/json"
	"time"
)

type ExecutionResult struct {
	Name string

	LastExecution time.Time
	NextExecution time.Time
	LastDuration  time.Duration

	Error error
}

func (p ExecutionResult) MarshalBinary() ([]byte, error) {
	return json.Marshal(p)
}

func (p *ExecutionResult) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, p)
}
