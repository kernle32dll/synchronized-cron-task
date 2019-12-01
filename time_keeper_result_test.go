package crontask_test

import (
	crontask "github.com/kernle32dll/synchronized-cron-task"
	"reflect"

	"errors"
	"testing"
	"time"
)

const exampleJson = "{" +
	"\"Name\":\"some-task\"," +
	"\"LastExecution\":\"1991-05-23T01:02:03.000000004Z\"," +
	"\"NextExecution\":" +
	"\"1991-05-23T01:02:03.000000004Z\"," +
	"\"LastDuration\":3600000000000," +
	"\"Error\":\"some-error\"" +
	"}"

// Tests that the TimeKeeperRedisPrefix option correctly applies.
func Test_ExecutionResult_MarshalBinary(t *testing.T) {
	// given
	option := &crontask.ExecutionResult{
		Name:          "some-task",
		LastExecution: time.Date(1991, 5, 23, 1, 2, 3, 4, time.UTC),
		NextExecution: time.Date(1991, 5, 23, 1, 2, 3, 4, time.UTC),
		LastDuration:  time.Hour,
		Error:         errors.New("some-error"),
	}

	// when
	result, err := option.MarshalBinary()

	// then
	if err != nil {
		t.Errorf("unexpected error, got %s", err)
	}

	expected := exampleJson
	if res := string(result); res != expected {
		t.Errorf("unexpected marshalling result, got %q, wanted %q", res, expected)
	}
}

// Tests that the TimeKeeperRedisPrefix option correctly applies.
func Test_ExecutionResult_UnmarshalBinary(t *testing.T) {
	// given
	option := &crontask.ExecutionResult{}

	// when
	err := option.UnmarshalBinary([]byte(exampleJson))

	// then
	if err != nil {
		t.Errorf("unexpected error, got %s", err)
	}

	expected := &crontask.ExecutionResult{
		Name:          "some-task",
		LastExecution: time.Date(1991, 5, 23, 1, 2, 3, 4, time.UTC),
		NextExecution: time.Date(1991, 5, 23, 1, 2, 3, 4, time.UTC),
		LastDuration:  time.Hour,
		Error:         errors.New("some-error"),
	}
	if !reflect.DeepEqual(option, expected) {
		t.Errorf("unexpected marshalling result, got %q, wanted %q", option, expected)
	}
}

// Tests that the TimeKeeperRedisPrefix option correctly applies.
func Test_ExecutionResult_UnmarshalBinary_error(t *testing.T) {
	// given
	option := &crontask.ExecutionResult{}

	// when
	err := option.UnmarshalBinary([]byte("invalid"))

	// then
	if err == nil {
		t.Error("expected error, got nil")
	}

	expected := &crontask.ExecutionResult{}
	if !reflect.DeepEqual(option, expected) {
		t.Errorf("unexpected marshalling result, got %q, wanted %q", option, expected)
	}
}
