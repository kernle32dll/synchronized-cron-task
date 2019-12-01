package crontask

import (
	"encoding/json"
	"errors"
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

// executionResultInternal is an internal wrapper, to allow
// correct un-/marshalling of errors.
type executionResultInternal struct {
	Name string

	LastExecution time.Time
	NextExecution time.Time
	LastDuration  time.Duration

	Error string
}

// MarshalBinary marshalls the ExecutionResult in JSON.
func (p ExecutionResult) MarshalBinary() ([]byte, error) {
	return json.Marshal(executionResultInternal{
		Name:          p.Name,
		LastExecution: p.LastExecution,
		NextExecution: p.NextExecution,
		LastDuration:  p.LastDuration,
		Error:         p.Error.Error(),
	})
}

// UnmarshalBinary unmarshalls an ExecutionResult from JSON.
func (p *ExecutionResult) UnmarshalBinary(data []byte) error {
	exec := &executionResultInternal{}
	if err := json.Unmarshal(data, exec); err != nil {
		return err
	}

	p.Name = exec.Name
	p.LastExecution = exec.LastExecution
	p.NextExecution = exec.NextExecution
	p.LastDuration = exec.LastDuration
	p.Error = errors.New(exec.Error)

	return nil
}
