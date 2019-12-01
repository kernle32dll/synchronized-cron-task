package crontask

import (
	"encoding/json"
	"time"
)

// ExecutionResult describes a single recorded run of a task
type ExecutionResult struct {
	Name string

	LastExecution time.Time
	NextExecution time.Time
	LastDuration  time.Duration

	Error error
}

// MarshalBinary marshalls the ExecutionResult in JSON.
func (p ExecutionResult) MarshalBinary() ([]byte, error) {
	return json.Marshal(p)
}

// UnmarshalBinary unmarshalls an ExecutionResult from JSON.
func (p *ExecutionResult) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, p)
}
